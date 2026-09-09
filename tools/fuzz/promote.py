#!/usr/bin/env python3
"""Promeut les entrées intéressantes d'une campagne de fuzzing vers le corpus versionné.

Le problème qu'il résout. Une campagne accumule ses trouvailles dans `GOCACHE`, qui
disparaît avec la machine — et un runner de CI démarre toujours à froid. Mesuré sur ce
dépôt : 125 entrées intéressantes en vingt-cinq secondes, deux graines versionnées.
Chaque campagne repartait donc de deux graines et refaisait le chemin que la précédente
avait déjà parcouru. Ce n'est pas une perte de qualité mesurée, c'est une perte de
TRAVAIL, et elle se répète à chaque exécution.

Les trois règles, et chacune répond à une question que l'issue #124 pose.

LESQUELLES PROMOUVOIR. Pas toutes : des centaines de fichiers dont la valeur décroît
vite noieraient le corpus et allongeraient chaque `go test`. Le plafond est explicite
(`--max`), et la sélection est DÉTERMINISTE — les entrées sont triées par leur nom, qui
est le hachage de leur contenu, puis échantillonnées à pas régulier. Deux promotions
sur le même cache donnent le même corpus : une sélection au hasard rendrait le diff
illisible et le résultat irreproductible.

QU'UNE GRAINE N'ENTRE PAS AVANT SA CORRECTION. Le corpus versionné est EXÉCUTÉ par
`go test` : une graine qui déclenche un défaut non corrigé rend `main` rouge. Chaque
candidate est donc éprouvée contre l'arbre courant AVANT d'entrer, et celles qui font
tomber la cible sont écartées et NOMMÉES. Elles entreront avec la PR qui corrige, ce
qui est l'ordre exigé par l'issue — jamais avant.

QUI PROMEUT. Ce script, lancé à la main ou par un mainteneur après une campagne. Pas le
workflow : une graine qui entre au dépôt est du contenu committé, et il se relit.

    mise run fuzz-promote            # promeut, plafond par défaut
    mise run fuzz-promote -- --dry-run --max 40
"""

import argparse
import hashlib
import pathlib
import shutil
import subprocess
import sys
import tempfile

# Les cibles, et où vit leur corpus versionné. Écrites plutôt que découvertes : une
# cible ajoutée doit être un geste conscient, pas un effet de bord d'un nom de fonction.
CIBLES = [
    ("./internal/tfparse/", "FuzzParsePlan", "internal/tfparse/testdata/fuzz/FuzzParsePlan"),
    ("./cmd/", "FuzzInventoryWalk", "cmd/testdata/fuzz/FuzzInventoryWalk"),
]

# Plafond par défaut, PAR CIBLE. Choisi pour que le corpus reste lisible dans un diff et
# que `go test` ne s'allonge pas notablement : chaque graine est rejouée à chaque
# exécution, donc le corpus est un coût permanent, pas un stock gratuit.
MAX_PAR_CIBLE = 32


def racine() -> pathlib.Path:
    return pathlib.Path(__file__).resolve().parent.parent.parent


def cache_de(module: str, paquet: str, cible: str) -> pathlib.Path:
    gocache = subprocess.run(
        ["go", "env", "GOCACHE"], capture_output=True, text=True, check=True
    ).stdout.strip()
    rel = paquet.removeprefix("./").rstrip("/")
    return pathlib.Path(gocache) / "fuzz" / module / rel / cible


def module_courant(root: pathlib.Path) -> str:
    for ligne in (root / "go.mod").read_text().splitlines():
        if ligne.startswith("module "):
            return ligne.split(None, 1)[1].strip()
    raise SystemExit("go.mod sans directive module")


def echantillonne(entrees: list[pathlib.Path], plafond: int) -> list[pathlib.Path]:
    """Prend au plus `plafond` entrées, à pas régulier sur la liste triée.

    Le pas régulier plutôt que les N premières : les noms sont des hachages, donc leur
    ordre n'a aucun rapport avec ce que les entrées explorent. Prendre le début du tri
    reviendrait à prendre un sous-ensemble arbitraire mais CORRÉLÉ, là où le pas balaie
    tout l'espace des hachages.
    """
    if len(entrees) <= plafond:
        return entrees
    pas = len(entrees) / plafond
    return [entrees[int(i * pas)] for i in range(plafond)]


def graine_passe(root: pathlib.Path, paquet: str, cible: str, contenu: bytes) -> bool:
    """Éprouve UNE graine contre l'arbre courant, dans un corpus temporaire.

    On n'écrit pas dans le corpus versionné pour tester : une graine fautive y laisserait
    `main` rouge le temps du test, et un script interrompu la laisserait pour de bon.
    """
    dossier = root / paquet.removeprefix("./").rstrip("/") / "testdata" / "fuzz" / cible
    dossier.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile(dir=dossier, prefix="candidate-", delete=False) as f:
        chemin = pathlib.Path(f.name)
        f.write(contenu)
    try:
        r = subprocess.run(
            ["go", "test", paquet, "-run", f"^{cible}$", "-count=1"],
            cwd=root, capture_output=True, text=True,
        )
        return r.returncode == 0
    finally:
        chemin.unlink(missing_ok=True)


def promeut(root: pathlib.Path, module: str, paquet: str, cible: str, dest_rel: str,
            plafond: int, dry: bool) -> tuple[int, int, list[str]]:
    cache = cache_de(module, paquet, cible)
    dest = root / dest_rel
    if not cache.is_dir():
        print(f"  {cible} : aucun cache de campagne ({cache})")
        return 0, 0, []
    deja = [p for p in dest.glob("*") if p.is_file()]
    connues = {p.read_bytes() for p in deja}
    candidates = sorted(p for p in cache.iterdir() if p.is_file())
    nouvelles = [p for p in candidates if p.read_bytes() not in connues]
    # Le plafond borne le CORPUS, pas l'exécution. Un plafond par exécution laisserait
    # le corpus grossir sans fin au fil des campagnes — et chaque graine est rejouée à
    # chaque `go test`, donc un corpus qui enfle est un coût permanent qui enfle avec
    # lui. Saturé, on le DIT : la sélection est finie, elle ne s'est pas tue.
    reste = max(0, plafond - len(deja))
    retenues = echantillonne(nouvelles, reste)
    etat = f"corpus {len(deja)}/{plafond}"
    if reste == 0:
        etat += " — SATURÉ, relever --max pour en accepter davantage"
    print(f"  {cible} : {len(candidates)} en cache, {len(nouvelles)} nouvelles, "
          f"{len(retenues)} retenues ({etat})")

    promues, ecartees = 0, []
    for p in retenues:
        contenu = p.read_bytes()
        if not graine_passe(root, paquet, cible, contenu):
            # NOMMÉE plutôt que jetée : une graine qui fait tomber la cible est une
            # trouvaille, pas un déchet. Elle entrera avec la PR qui corrige.
            ecartees.append(hashlib.sha256(contenu).hexdigest()[:16])
            continue
        promues += 1
        if not dry:
            dest.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(p, dest / p.name)
    return promues, len(nouvelles), ecartees


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--max", type=int, default=MAX_PAR_CIBLE,
                    help=f"plafond de graines promues par cible (défaut {MAX_PAR_CIBLE})")
    ap.add_argument("--dry-run", action="store_true",
                    help="ne rien écrire, dire seulement ce qui serait promu")
    args = ap.parse_args(argv)

    root = racine()
    module = module_courant(root)
    print(f"promotion du corpus de fuzzing — module {module}")
    total, ecartees_globales = 0, []
    for paquet, cible, dest in CIBLES:
        promues, _, ecartees = promeut(root, module, paquet, cible, dest, args.max, args.dry_run)
        total += promues
        ecartees_globales += [f"{cible}:{e}" for e in ecartees]

    print()
    if args.dry_run:
        print(f"{total} graine(s) seraient promues (--dry-run : rien n'a été écrit)")
    else:
        print(f"{total} graine(s) promues au corpus versionné")
    if ecartees_globales:
        print()
        print("ÉCARTÉES — elles font tomber leur cible sur l'arbre courant :")
        for e in ecartees_globales:
            print(f"  {e}")
        print("  Une graine qui échoue entre au corpus AVEC la correction qu'elle motive,")
        print("  jamais avant : le corpus versionné est rejoué par `go test`.")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
