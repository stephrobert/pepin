#!/usr/bin/env python3
"""LA PORTE DE RELEASE (issue #178) : un seul verdict, GO ou NO-GO, et son rapport.

    mise run release-gate -- v0.4.0
    mise run release-gate -- v0.4.0 --stages 1
    mise run release-gate -- v0.4.0 --skip '4=aucun tag antérieur sur ce clone'
    mise run gate:selftest

# Le problème qu'elle ferme

`release-check` et le canari font ce qu'ils disent. Ce que le dépôt n'avait pas,
c'est un verdict UNIQUE qu'un mainteneur lit comme GO, produit depuis ce qu'un
utilisateur vivra vraiment : les artefacts publiés, les commandes documentées, un
tenant réel, et les affirmations que la documentation fait.

Un audit externe d'une journée sur `main` a trouvé À LA MAIN un faux négatif sur une
VM à deux cartes, une évasion en `/2`, une page d'installation épinglée sur une
version que le dépôt qualifie lui-même de cassée, une qualification régionale
déclarée pour tout un fournisseur, et un bloc de rapport imprimant la remédiation
d'un autre contrôle. AUCUN n'était visible des portes existantes. C'est cette classe
de défaut que la porte doit trouver avant un tag, et non un auditeur après.

# Les étapes, dans cet ordre

    1  hors ligne          ce que le dépôt possède déjà, lancé comme un tout,
                           plus les affirmations que la DOCUMENTATION fait
    3  compte cloud        le tenant de qualification, opt-in — il vit dans
                           tools/qualification/ et s'appelle par `mise run qualify`
    4  surfaces et notes   ce qu'un changement de verdict doit avoir écrit

Hors ligne d'abord : une porte qui exige le réseau pour dire qu'un CHANGELOG manque
est une porte qu'on lance moins souvent.

L'ÉTAPE 2 N'EXISTE PAS ENCORE. Elle vérifiera les artefacts tels qu'un utilisateur
les reçoit — le bloc de vérification du README rejoué mot pour mot sur les assets du
tag précédent, `install.sh` accepté intact et refusé sur un octet corrompu, l'image
scannant un plan d'exemple. Son numéro est RÉSERVÉ plutôt que réattribué : un rapport
de la porte doit vouloir dire la même chose d'une release à l'autre, et une étape 2
qui désignerait deux choses différentes selon la version rendrait les rapports
archivés illisibles. Elle n'est pas déclarée « sautée » non plus — un saut se motive,
et « pas encore écrite » n'est pas un motif d'exécution, c'est un état du dépôt.

# Les trois règles qui la rendent opposable

RIEN NE DIT GO TANT QU'UNE ÉTAPE EST ROUGE. Le verdict final n'est pas une synthèse
au jugé : il est mécanique, et un seul contrôle rouge suffit.

UNE ÉTAPE SAUTÉE PORTE UN MOTIF ÉCRIT, ET CE MOTIF EST DANS LE RAPPORT. Sauter est
parfois légitime — pas de réseau, pas de compte —, mais un saut muet transforme
silencieusement la porte en formalité. Un `--skip` sans motif est donc refusé.

UNE PORTE QU'ON N'A JAMAIS VUE ROUGE NE GARDE RIEN. `selftest` casse délibérément
chacune des règles ci-dessus et exige un NO-GO. C'est la doctrine de
tools/falsify/falsify.py et de tools/qualification/qualify.py, appliquée à la porte
qui les orchestre.

# Convention de sortie des OUTILS de release

0 GO · 1 NO-GO · 2 erreur d'usage.
"""
import argparse
import datetime as dt
import json
import pathlib
import re
import subprocess
import sys
import time

import yaml

ROOT = pathlib.Path(__file__).resolve().parents[2]
SORTIE = ROOT / "release-gate"

# Les verdicts d'un contrôle, et c'est tout le vocabulaire.
GO = "go"
NOGO = "no-go"
SKIPPED = "skipped"
# `reported` : mesuré, imprimé, JAMAIS bloquant. Réservé à ce que le dépôt documente
# déjà comme indicatif (la couverture de remédiation). Un verdict qui bloque parfois
# est un verdict qu'on apprend à ignorer.
REPORTED = "reported"


# ═══════════════════════════════════════════════════════════════════════════════
# Fonctions PURES — celles que `selftest` éprouve
# ═══════════════════════════════════════════════════════════════════════════════


def verdict_de_letape(controles, motif_saut=None):
    """Le verdict d'une étape, mécaniquement.

    Un seul contrôle rouge suffit. Un saut sans motif écrit est un NO-GO : sauter en
    silence est la façon dont une porte devient une formalité, et c'est exactement ce
    qu'on ne peut pas se permettre juste avant un tag.
    """
    if motif_saut is not None:
        return SKIPPED if str(motif_saut).strip() else NOGO
    if any(c["verdict"] == NOGO for c in controles):
        return NOGO
    return GO


def verdict_final(etapes):
    """GO seulement si aucune étape n'est rouge. Une étape sautée ne vaut pas GO."""
    if any(e["verdict"] == NOGO for e in etapes):
        return NOGO
    return GO


def version_tuple(v):
    """`v1.2.3` → (1, 2, 3). None si ce n'est pas une version sémantique."""
    m = re.fullmatch(r"v?(\d+)\.(\d+)\.(\d+)", str(v).strip())
    return tuple(int(x) for x in m.groups()) if m else None


# Une version de PÉPIN, c'est-à-dire une version qui apparaît dans un fragment
# désignant ce dépôt. `v0.18.2` d'un provider Terraform vendu dans un exemple n'a rien
# à voir avec cet outil, et le confondre rendrait la porte bruyante — donc désarmée.
DEPOT = "stephrobert/pepin"
SEMVER = re.compile(r"v\d+\.\d+\.\d+")


def versions_de_pepin(texte):
    """Les versions de Pépin épinglées dans un texte, ligne par ligne.

    Rend des couples (numéro de ligne, version). La ligne ENTIÈRE doit nommer le
    dépôt : c'est ce qui distingue `.../stephrobert/pepin/v0.2.0/...` d'un numéro de
    version quelconque, sans avoir à écrire une grammaire d'URL.
    """
    out = []
    for n, ligne in enumerate(texte.splitlines(), start=1):
        if DEPOT not in ligne:
            continue
        for v in SEMVER.findall(ligne):
            out.append((n, v))
    return out


def trop_ancienne(version, minimum):
    """La version épinglée est-elle antérieure au minimum déclaré sûr ?"""
    a, b = version_tuple(version), version_tuple(minimum)
    if a is None or b is None:
        return False
    return a < b


# ── Les liens de la documentation ──────────────────────────────────────────────

FENCE = re.compile(r"^\s{0,3}(```|~~~)")
LIEN = re.compile(r"!?\[[^\]]*\]\(([^)\s]+)(?:\s+\"[^\"]*\")?\)")
TITRE = re.compile(r"^(#{1,6})\s+(.*?)\s*#*\s*$")


def hors_blocs_de_code(texte):
    """Le texte débarrassé de ses blocs clôturés.

    Un lien dans un exemple de commande n'est pas un lien, et un `# titre` dans un
    bloc shell n'est pas un titre. Les compter produirait des faux positifs dans les
    deux sens — une ancre inventée, un lien mort signalé sur une ligne d'exemple.
    """
    out, dans = [], None
    for ligne in texte.splitlines():
        m = FENCE.match(ligne)
        if m:
            if dans is None:
                dans = m.group(1)
            elif ligne.strip().startswith(dans):
                dans = None
            out.append("")
            continue
        out.append("" if dans is not None else ligne)
    return "\n".join(out)


def ancre_de(titre):
    """Le fragment qu'un titre engendre, à la façon de GitHub.

    Minuscules, la mise en forme retirée, tout ce qui n'est ni alphanumérique ni
    espace ni tiret supprimé, les espaces devenus tirets. Les lettres accentuées
    SURVIVENT — `#ce-que-le-scan-à-rôle-réduit-a-mesuré` est une ancre valide, et
    les retirer casserait la moitié des liens d'un dépôt francophone.
    """
    t = re.sub(r"`([^`]*)`", r"\1", titre)
    t = re.sub(r"\*\*([^*]*)\*\*|\*([^*]*)\*|__([^_]*)__|_([^_]*)_", lambda m: next(g for g in m.groups() if g is not None), t)
    t = re.sub(r"!?\[([^\]]*)\]\([^)]*\)", r"\1", t)  # un lien dans un titre
    t = t.strip().lower()
    t = "".join(c for c in t if c.isalnum() or c in " -_")
    return t.replace(" ", "-")


def ancres_de(texte):
    """Toutes les ancres qu'un document offre, doublons numérotés comme GitHub."""
    vues, out = {}, set()
    for ligne in hors_blocs_de_code(texte).splitlines():
        m = TITRE.match(ligne)
        if not m:
            continue
        base = ancre_de(m.group(2))
        n = vues.get(base, 0)
        vues[base] = n + 1
        out.add(base if n == 0 else f"{base}-{n}")
    return out


def hors_code_inline(texte):
    """Le texte débarrassé de ses fragments `entre accents graves`.

    Un lien CITÉ dans une phrase n'est pas un lien. `CLAUDE.md` documente le sélecteur
    de langue en écrivant `[🇫🇷 Français](*.fr.md)` entre accents graves : le motif est
    l'objet du propos, pas une cible. Le contrôle le signalait comme mort, ce qui est
    la façon la plus sûre de faire désarmer un contrôle de liens.

    Ce retrait ne vaut QUE pour l'extraction des liens : une ancre, elle, se calcule
    sur le titre avec le contenu de ses accents graves, comme le fait GitHub.
    """
    return re.sub(r"`+[^`\n]*`+", "", texte)


def liens_relatifs(texte):
    """Les cibles de lien qui désignent le dépôt, pas le web."""
    out = []
    for cible in LIEN.findall(hors_code_inline(hors_blocs_de_code(texte))):
        if re.match(r"^[a-zA-Z][a-zA-Z0-9+.-]*:", cible) or cible.startswith("//"):
            continue  # http:, https:, mailto:, ...
        out.append(cible)
    return out


def resout(cible, fichier, existe, ancres):
    """Un lien relatif résout-il ? Rend None si oui, le motif du refus sinon.

    `existe` et `ancres` sont injectés : c'est ce qui rend cette fonction pure, donc
    éprouvable sans arborescence, et c'est la partie où les erreurs se cachent.
    """
    chemin, _, fragment = cible.partition("#")
    if chemin:
        cible_p = (fichier.parent / chemin).resolve()
        if not existe(cible_p):
            return f"cible absente : {chemin}"
    else:
        cible_p = fichier
    if not fragment:
        return None
    if not str(cible_p).endswith(".md"):
        return None  # une ancre dans un non-markdown ne se vérifie pas d'ici
    dispo = ancres(cible_p)
    if dispo is None:
        return None  # fichier illisible : l'absence est déjà signalée plus haut
    if fragment.lower() not in dispo:
        return f"ancre absente : #{fragment}"
    return None


# ═══════════════════════════════════════════════════════════════════════════════
# Exécution
# ═══════════════════════════════════════════════════════════════════════════════


def controle(nom, verdict, preuve, duree=0.0):
    return {"name": nom, "verdict": verdict, "evidence": preuve, "duration_s": round(duree, 1)}


def lance(nom, cmd, echec, env=None, reported=False):
    """Lance une commande et en fait un contrôle.

    La PREUVE d'un échec est la fin de la sortie, pas un « échoué » : une porte qui
    dit qu'elle a refusé sans dire ce qu'elle a vu se relance à l'aveugle.
    """
    t0 = time.time()
    r = subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True, env=env)
    d = time.time() - t0
    if r.returncode == 0:
        return controle(nom, GO, " ".join(cmd), d)
    fin = (r.stdout + r.stderr).strip().splitlines()
    fin = "\n".join(fin[-12:]) if fin else "(aucune sortie)"
    return controle(nom, REPORTED if reported else NOGO, f"{echec}\n{fin}", d)


def fichiers_suivis(prefixe):
    """Les fichiers que git suit sous un préfixe.

    `git ls-files` plutôt qu'un parcours du disque : un `.terraform/` téléchargé
    localement porte des versions de tiers, et la porte les compterait comme des
    versions de Pépin périmées. Ce qui n'est pas committé n'est pas publié.
    """
    r = subprocess.run(["git", "ls-files", "-z", "--", prefixe],
                       cwd=ROOT, capture_output=True, text=True, check=True)
    return [ROOT / p for p in r.stdout.split("\0") if p]


# ── Étape 1 — hors ligne ───────────────────────────────────────────────────────


def check_versions_epinglees():
    """#H : aucune version de Pépin épinglée ne précède le minimum déclaré sûr."""
    t0 = time.time()
    spec = yaml.safe_load((ROOT / "references/release/pinning.yaml").read_text(encoding="utf-8"))
    minimum = spec["minimum_sur"]

    fautes = []
    for page in spec["pages"]:
        if minimum not in (ROOT / page).read_text(encoding="utf-8"):
            fautes.append(f"{page} : ne cite pas le minimum déclaré sûr ({minimum})")

    for surface in spec["surfaces"]:
        for f in fichiers_suivis(surface):
            try:
                texte = f.read_text(encoding="utf-8")
            except (UnicodeDecodeError, OSError):
                continue
            for n, v in versions_de_pepin(texte):
                if trop_ancienne(v, minimum):
                    fautes.append(f"{f.relative_to(ROOT)}:{n} : {v} précède le minimum sûr {minimum}")

    d = time.time() - t0
    if fautes:
        return controle("les versions épinglées valent au moins le minimum déclaré sûr",
                        NOGO,
                        "\n".join(fautes) + f"\n  motif du minimum : {spec['motif'].strip()}", d)
    return controle("les versions épinglées valent au moins le minimum déclaré sûr",
                    GO, f"minimum {minimum}, aucune occurrence antérieure", d)


def check_liens_documentation():
    """Tout lien relatif de la documentation résout, ancre comprise."""
    t0 = time.time()
    cache = {}

    def existe(p):
        return p.exists()

    def ancres(p):
        if p not in cache:
            try:
                cache[p] = ancres_de(p.read_text(encoding="utf-8"))
            except OSError:
                cache[p] = None
        return cache[p]

    fichiers = fichiers_suivis("docs") + [p for p in fichiers_suivis(".") if p.parent == ROOT and p.suffix == ".md"]
    fautes = []
    for f in sorted(set(fichiers)):
        if f.suffix != ".md":
            continue
        texte = f.read_text(encoding="utf-8")
        for cible in liens_relatifs(texte):
            motif = resout(cible, f, existe, ancres)
            if motif:
                fautes.append(f"{f.relative_to(ROOT)} → {cible} : {motif}")

    d = time.time() - t0
    if fautes:
        return controle("tout lien relatif de la documentation résout", NOGO,
                        f"{len(fautes)} lien(s) morts\n" + "\n".join(fautes[:25]), d)
    return controle("tout lien relatif de la documentation résout", GO,
                    f"{len(set(fichiers))} fichier(s) markdown parcourus", d)


def check_description_racine():
    """#150 : la phrase d'accueil nomme EXACTEMENT les fournisseurs enregistrés.

    Mesuré sur le binaire, pas lu dans le code : une liste recopiée se périme au
    premier fournisseur ajouté, et c'est la première phrase qu'un nouvel utilisateur
    lit. Elle annonçait OVH — une entrée de feuille de route — et omettait Kubernetes.
    """
    t0 = time.time()
    binaire = ROOT / "pepin"
    if not binaire.exists():
        return controle("la description racine nomme les fournisseurs enregistrés", NOGO,
                        "binaire absent : `mise run build` d'abord", time.time() - t0)
    accueil = subprocess.run([str(binaire)], cwd=ROOT, capture_output=True, text=True).stdout
    liste = subprocess.run([str(binaire), "provider", "list"], cwd=ROOT,
                           capture_output=True, text=True).stdout

    enregistres = {m.group(1) for m in re.finditer(r"^\s*(\S+)", liste, re.M) if m.group(1).islower()}
    enregistres = {p for p in enregistres if p.isidentifier() or "-" in p}
    m = re.search(r"\(([^)]*)\)", accueil)
    annonces = {p.strip().lower() for p in m.group(1).split(",")} if m else set()
    annonces.discard("…")
    annonces.discard("...")

    d = time.time() - t0
    manquants = enregistres - annonces
    inventes = annonces - enregistres
    if manquants or inventes:
        return controle("la description racine nomme les fournisseurs enregistrés", NOGO,
                        f"annoncés : {sorted(annonces)}\nenregistrés : {sorted(enregistres)}\n"
                        f"omis : {sorted(manquants)} · inventés : {sorted(inventes)}", d)
    return controle("la description racine nomme les fournisseurs enregistrés", GO,
                    f"{sorted(annonces)}", d)


def etape1(version):
    """Ce que le dépôt possède déjà, lancé comme un tout, plus ce que la doc affirme."""
    c = [
        lance("release-check : tout ce qui doit être vrai avant un tag",
              ["mise", "run", "release-check", "--", version],
              "le préflight refuse ce tag"),
        lance("mise run audit (vet + lint + gosec + govulncheck + osv)",
              ["mise", "run", "audit"], "l'audit refuse cet arbre"),
        lance("mise run adr-drift", ["mise", "run", "adr-drift"],
              "un ADR cite un fichier disparu, ou un invariant n'a plus de test"),
        lance("mise run falsify:all : chaque porte a été vue rouge",
              ["mise", "run", "falsify:all"],
              "une porte versionnée ne sait plus échouer"),
        lance("campagne de fuzzing bornée",
              ["go", "test", "./internal/tfparse/", "-run", "^$",
               "-fuzz", "FuzzParsePlan", "-fuzztime=20s"],
              "une entrée fait paniquer le parseur de plan"),
        lance("les blocs générés de la documentation sont à jour",
              ["go", "test", "./internal/docgen/", "-run", "TestGeneratedDocsAreUpToDate", "-count=1"],
              "un bloc pepin:gen a dérivé de ce que le code calcule"),
        check_versions_epinglees(),
        check_liens_documentation(),
        check_description_racine(),
        # Indicatif, et le dépôt le documente comme tel : la couverture de remédiation
        # est une mesure de progrès, pas une promesse envers un consommateur.
        lance("couverture de remédiation (indicatif)", ["mise", "run", "check-remediation"],
              "couverture de remédiation en baisse", reported=True),
    ]
    return c


# ── Étape 4 — surfaces et notes ────────────────────────────────────────────────


def tag_precedent():
    r = subprocess.run(["git", "describe", "--tags", "--abbrev=0"],
                       cwd=ROOT, capture_output=True, text=True)
    return r.stdout.strip() if r.returncode == 0 else ""


def check_verdict_change_a_sa_ligne():
    """Un verdict qui bouge sur un tenant INCHANGÉ a sa ligne au CHANGELOG.

    C'est la promesse que l'en-tête du CHANGELOG fait explicitement. `expected.yaml`
    est l'endroit où un tel changement se voit : il fige, contrôle par sujet, ce que
    la qualification attend d'un tenant qui, lui, n'a pas bougé.

    Ce que la porte peut vérifier mécaniquement : que le CHANGELOG a BOUGÉ dans le
    même intervalle. Que la ligne EXPLIQUE le changement se lit — le diff est imprimé
    pour ça. Prétendre mesurer la seconde chose serait un contrôle qui rassure sans
    rien tenir.
    """
    t0 = time.time()
    prec = tag_precedent()
    if not prec:
        return controle("un verdict qui bouge à sa ligne au CHANGELOG", SKIPPED,
                        "aucun tag antérieur : rien à comparer", time.time() - t0)

    def changes(*chemins):
        r = subprocess.run(["git", "diff", "--name-only", f"{prec}..HEAD", "--", *chemins],
                           cwd=ROOT, capture_output=True, text=True)
        return [l for l in r.stdout.splitlines() if l.strip()]

    attendus = changes("references/qualification/*/expected.yaml")
    d = time.time() - t0
    if not attendus:
        return controle("un verdict qui bouge à sa ligne au CHANGELOG", GO,
                        f"aucun expected.yaml modifié depuis {prec}", d)
    if not changes("CHANGELOG.md", "CHANGELOG.fr.md"):
        return controle("un verdict qui bouge à sa ligne au CHANGELOG", NOGO,
                        f"depuis {prec} : " + ", ".join(attendus) +
                        "\naucun CHANGELOG modifié — un verdict qui change sur un tenant "
                        "inchangé oblige son lecteur à expliquer à un auditeur un changement "
                        "qu'il n'a pas fait", d)
    diff = subprocess.run(["git", "diff", f"{prec}..HEAD", "--",
                           "references/qualification/*/expected.yaml"],
                          cwd=ROOT, capture_output=True, text=True).stdout
    return controle("un verdict qui bouge à sa ligne au CHANGELOG", GO,
                    f"depuis {prec}, à relire dans le rapport :\n" +
                    "\n".join(diff.splitlines()[:40]), d)


def etape4(version):
    return [
        lance("les surfaces gelées correspondent à leur fixture",
              ["go", "test", "./cmd/", "-count=1", "-run",
               "TestTheFrozenSurfacesStillMatchTheirFixture|TestFrozenHistoryOnlyGrows"],
              "une surface gelée a bougé sans que sa constante soit incrémentée"),
        check_verdict_change_a_sa_ligne(),
    ]


ETAPES = {
    1: ("hors ligne : le dépôt, et ce que sa documentation affirme", etape1),
    4: ("surfaces et notes : ce qu'un changement de verdict doit avoir écrit", etape4),
}


# ═══════════════════════════════════════════════════════════════════════════════
# Rapport
# ═══════════════════════════════════════════════════════════════════════════════

SYMBOLE = {GO: "✔", NOGO: "✘", SKIPPED: "—", REPORTED: "·"}


def ecris_rapport(version, etapes, final):
    SORTIE.mkdir(parents=True, exist_ok=True)
    for e in etapes:
        (SORTIE / f"stage{e['stage']}.json").write_text(
            json.dumps(e, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")

    lignes = [
        f"# Porte de release — {version}",
        "",
        f"Lancée le {dt.datetime.now(dt.timezone.utc).isoformat(timespec='seconds')}, "
        f"sur `{subprocess.run(['git', 'rev-parse', '--short', 'HEAD'], cwd=ROOT, capture_output=True, text=True).stdout.strip()}`.",
        "",
        "| Étape | Verdict | Durée |",
        "|---|:-:|--:|",
    ]
    for e in etapes:
        lignes.append(f"| {e['stage']} — {e['title']} | **{e['verdict'].upper()}** | {e['duration_s']:.0f} s |")
    lignes.append("")

    for e in etapes:
        lignes += [f"## Étape {e['stage']} — {e['title']}", ""]
        if e["verdict"] == SKIPPED:
            lignes += [f"**Sautée.** Motif : {e['skip_reason']}", ""]
            continue
        for c in e["checks"]:
            lignes.append(f"- {SYMBOLE[c['verdict']]} **{c['name']}** ({c['duration_s']:.0f} s)")
            if c["verdict"] in (GO,):
                continue
            for l in str(c["evidence"]).splitlines():
                lignes.append(f"      {l}")
        lignes.append("")

    rouges = [str(e["stage"]) for e in etapes if e["verdict"] == NOGO]
    lignes.append(f"**{'GO' if final == GO else 'NO-GO: étape(s) ' + ', '.join(rouges)}**")
    (SORTIE / "REPORT.md").write_text("\n".join(lignes) + "\n", encoding="utf-8")
    return SORTIE / "REPORT.md"


# ═══════════════════════════════════════════════════════════════════════════════
# selftest — la porte mise en échec
# ═══════════════════════════════════════════════════════════════════════════════


def selftest():
    """Casse chaque règle de la porte et exige qu'elle refuse.

    Une porte qu'on n'a jamais vue rouge ne garde rien : elle a le même aspect qu'une
    porte cassée. Chaque cas ci-dessous est un défaut RÉEL — celui qu'un audit a
    trouvé, ou celui qu'une correction hâtive introduirait.
    """
    echecs = []

    def veut(nom, obtenu, attendu):
        if obtenu != attendu:
            echecs.append(f"{nom} : obtenu {obtenu!r}, attendu {attendu!r}")

    # ── La règle du verdict : un seul rouge suffit ──────────────────────────────
    vert = [controle("a", GO, ""), controle("b", GO, "")]
    rouge = [controle("a", GO, ""), controle("b", NOGO, "casse")]
    indicatif = [controle("a", GO, ""), controle("b", REPORTED, "en baisse")]
    veut("une étape toute verte", verdict_de_letape(vert), GO)
    veut("un seul contrôle rouge", verdict_de_letape(rouge), NOGO)
    veut("un contrôle indicatif ne bloque pas", verdict_de_letape(indicatif), GO)

    # ── La règle du saut : muet = refus ────────────────────────────────────────
    veut("saut motivé", verdict_de_letape(vert, motif_saut="pas de réseau"), SKIPPED)
    veut("saut muet", verdict_de_letape(vert, motif_saut=""), NOGO)
    veut("saut d'espaces", verdict_de_letape(vert, motif_saut="   "), NOGO)

    # ── Le verdict final ───────────────────────────────────────────────────────
    veut("toutes vertes", verdict_final([{"verdict": GO}, {"verdict": GO}]), GO)
    veut("une rouge", verdict_final([{"verdict": GO}, {"verdict": NOGO}]), NOGO)
    # Une étape sautée ne vaut pas GO en soi, mais elle n'interdit pas GO : c'est le
    # motif écrit qui porte l'information, et il est dans le rapport.
    veut("une sautée motivée", verdict_final([{"verdict": GO}, {"verdict": SKIPPED}]), GO)

    # ── #H : les versions épinglées ────────────────────────────────────────────
    lignes = (
        "  - remote: 'https://raw.githubusercontent.com/stephrobert/pepin/v0.1.1/x.yml'\n"
        "  uses: stephrobert/pepin/.github/actions/pepin-scan@abc # v0.3.0\n"
        "- Installing exoscale/exoscale v0.18.2...\n"
        "docker run ghcr.io/stephrobert/pepin:v0.3.0 scan\n"
    )
    trouve = versions_de_pepin(lignes)
    veut("les versions de Pépin sont repérées", sorted(v for _, v in trouve),
         ["v0.1.1", "v0.3.0", "v0.3.0"])
    veut("une version de tiers n'est pas comptée",
         any(v == "v0.18.2" for _, v in trouve), False)
    veut("une version antérieure au minimum est refusée", trop_ancienne("v0.1.1", "v0.2.0"), True)
    veut("le minimum lui-même passe", trop_ancienne("v0.2.0", "v0.2.0"), False)
    veut("une version postérieure passe", trop_ancienne("v0.3.0", "v0.2.0"), False)
    veut("une chaine qui n'est pas une version ne refuse rien",
         trop_ancienne("vX.Y.Z", "v0.2.0"), False)

    # ── Les ancres, telles que GitHub les fabrique ─────────────────────────────
    veut("ancre accentuée", ancre_de("Ce que le scan à rôle réduit a mesuré"),
         "ce-que-le-scan-à-rôle-réduit-a-mesuré")
    veut("apostrophe retirée", ancre_de("A complete scan needs the account owner's keys"),
         "a-complete-scan-needs-the-account-owners-keys")
    veut("code et gras retirés", ancre_de("Le drapeau `--redact` est **obligatoire**"),
         "le-drapeau---redact-est-obligatoire")
    veut("doublons numérotés",
         sorted(ancres_de("# Titre\n\n## Titre\n\n### Titre\n")),
         ["titre", "titre-1", "titre-2"])
    veut("un titre dans un bloc de code n'est pas un titre",
         ancres_de("# Vrai\n\n```sh\n# Faux\n```\n"), {"vrai"})

    # ── Les liens ──────────────────────────────────────────────────────────────
    veut("un lien absolu est ignoré", liens_relatifs("[x](https://exemple.fr/a)"), [])
    veut("un lien relatif est retenu", liens_relatifs("[x](../adr/0012.md#invariants)"),
         ["../adr/0012.md#invariants"])
    veut("un lien dans un bloc de code est ignoré",
         liens_relatifs("```\n[x](mort.md)\n```\n"), [])
    # Le cas RÉEL trouvé au premier lancement : CLAUDE.md cite le sélecteur de langue
    # entre accents graves, et le motif `*.fr.md` était signalé comme lien mort.
    veut("un lien cité entre accents graves est ignoré",
         liens_relatifs("Le sélecteur : `[FR](*.fr.md)` en tête de page."), [])
    veut("un titre garde le contenu de ses accents graves",
         ancres_de("## Le drapeau `--redact`\n"), {"le-drapeau---redact"})
    veut("une image est un lien", liens_relatifs("![x](assets/a.png)"), ["assets/a.png"])

    faux = pathlib.Path("/depot/docs/page.md")
    presents = {pathlib.Path("/depot/docs/voisine.md")}
    ancres_test = {pathlib.Path("/depot/docs/voisine.md"): {"un-titre"},
                   faux: {"ici"}}
    ex = presents.__contains__
    an = ancres_test.get
    veut("cible absente", resout("absente.md", faux, ex, an), "cible absente : absente.md")
    veut("cible présente", resout("voisine.md", faux, ex, an), None)
    veut("ancre présente", resout("voisine.md#un-titre", faux, ex, an), None)
    veut("ancre absente", resout("voisine.md#autre", faux, ex, an), "ancre absente : #autre")
    veut("ancre du même fichier", resout("#ici", faux, ex, an), None)
    veut("ancre absente du même fichier", resout("#ailleurs", faux, ex, an),
         "ancre absente : #ailleurs")

    if echecs:
        print("selftest de la porte : ÉCHEC")
        for e in echecs:
            print(f"  ✘ {e}")
        return 1
    print("selftest de la porte : les règles refusent bien ce qu'elles doivent refuser")
    return 0


# ═══════════════════════════════════════════════════════════════════════════════


def cmd_run(args):
    version = args.version
    if not re.fullmatch(r"v\d+\.\d+\.\d+", version):
        print(f"usage : {version!r} n'est pas un tag vX.Y.Z", file=sys.stderr)
        return 2

    demandees = [int(s) for s in args.stages.split(",")] if args.stages else sorted(ETAPES)
    inconnues = [s for s in demandees if s not in ETAPES]
    if inconnues:
        print(f"usage : étape(s) inconnue(s) {inconnues} (disponibles : {sorted(ETAPES)})",
              file=sys.stderr)
        return 2

    sauts = {}
    for s in args.skip or []:
        num, _, motif = s.partition("=")
        try:
            sauts[int(num)] = motif
        except ValueError:
            print(f"usage : --skip attend N=motif, reçu {s!r}", file=sys.stderr)
            return 2

    print(f"porte de release pour {version} — étapes {demandees}")
    resultats = []
    for num in demandees:
        titre, fn = ETAPES[num]
        print(f"\n── étape {num} — {titre}")
        t0 = time.time()
        if num in sauts:
            v = verdict_de_letape([], motif_saut=sauts[num])
            e = {"stage": num, "title": titre, "verdict": v, "checks": [],
                 "skip_reason": sauts[num], "duration_s": 0.0}
            print(f"   {SYMBOLE.get(v, '?')} sautée — motif : {sauts[num]!r}")
            if v == NOGO:
                print("   un saut sans motif écrit est refusé : c'est ainsi qu'une porte "
                      "devient une formalité")
            resultats.append(e)
            continue
        controles = fn(version)
        v = verdict_de_letape(controles)
        for c in controles:
            print(f"   {SYMBOLE[c['verdict']]} {c['name']}")
            if c["verdict"] != GO:
                for l in str(c["evidence"]).splitlines()[:12]:
                    print(f"       {l}")
        resultats.append({"stage": num, "title": titre, "verdict": v, "checks": controles,
                          "duration_s": round(time.time() - t0, 1)})

    final = verdict_final(resultats)
    chemin = ecris_rapport(version, resultats, final)
    rouges = [str(e["stage"]) for e in resultats if e["verdict"] == NOGO]
    print(f"\n{chemin.relative_to(ROOT)}")
    print("GO" if final == GO else f"NO-GO: étape(s) {', '.join(rouges)}")
    return 0 if final == GO else 1


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)
    r = sub.add_parser("run", help="lance la porte pour un tag")
    r.add_argument("version")
    r.add_argument("--stages", help="étapes à lancer, séparées par des virgules")
    r.add_argument("--skip", action="append", metavar="N=motif",
                   help="saute une étape AVEC un motif écrit (un motif vide est refusé)")
    sub.add_parser("selftest", help="casse chaque règle de la porte et exige un refus")
    args = ap.parse_args()
    return selftest() if args.cmd == "selftest" else cmd_run(args)


if __name__ == "__main__":
    sys.exit(main())
