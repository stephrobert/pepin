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
    2  réseau              les artefacts tels qu'un utilisateur les reçoit, avec
                           les commandes EXTRAITES de la documentation
    3  compte cloud        le tenant de qualification, opt-in — il vit dans
                           tools/qualification/ et s'appelle par `mise run qualify`
    4  surfaces et notes   ce qu'un changement de verdict doit avoir écrit

Hors ligne d'abord : une porte qui exige le réseau pour dire qu'un CHANGELOG manque
est une porte qu'on lance moins souvent.

Et une étape dont TOUT ce qui mesure a été sauté ne vaut pas GO. L'étape 2 dépend
d'outils — `cosign`, `docker`, `gh` — que la machine du mainteneur peut ne pas avoir ;
chaque contrôle sait alors se sauter en le disant, et une étape entièrement sautée se
déclare SAUTÉE. Un vert qui n'a rien mesuré est le défaut que ce produit reproche aux
autres, et il n'a pas sa place dans sa propre porte.

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
import os
import pathlib
import re
import shutil
import socket
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

    Et une étape dont TOUT ce qui mesure a été sauté ne vaut pas GO. C'est le piège
    que l'étape 2 apporte : elle dépend d'outils et du réseau, chaque contrôle sait se
    sauter proprement quand son outil manque, et une machine sans `cosign` ni `docker`
    aurait donc rendu une étape verte n'ayant rien mesuré. Un vert qui ne mesure rien
    est exactement le défaut que ce produit reproche aux autres.
    """
    if motif_saut is not None:
        return SKIPPED if str(motif_saut).strip() else NOGO
    if any(c["verdict"] == NOGO for c in controles):
        return NOGO
    mesurants = [c for c in controles if c["verdict"] != REPORTED]
    if mesurants and all(c["verdict"] == SKIPPED for c in mesurants):
        return SKIPPED
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



# ── Étape 2 — les artefacts tels qu'un utilisateur les reçoit ──────────────────
#
# Ce que cette étape mesure, et pourquoi elle ne peut pas se contenter des tests.
#
# La CI prouve que les MÉCANISMES fonctionnent : elle sert des artefacts construits
# localement sur une boucle locale, et vérifie que l'installeur accepte l'un et refuse
# l'autre. Elle ne dit rien de la CHAÎNE PUBLIÉE — la signature qui vit chez Sigstore,
# l'attestation qui vit chez GitHub, l'image qui vit sur ghcr.io. Ces trois-là peuvent
# cesser de se vérifier sans qu'une ligne du dépôt ait bougé.
#
# D'où la règle de cette étape : les commandes ne sont pas RECOPIÉES ici, elles sont
# EXTRAITES de la documentation et exécutées telles quelles. Recopier prouverait que
# ma copie marche ; extraire prouve que la page marche. C'est toute la différence, et
# c'est l'invariant de l'ADR-0016 : « la documentation nomme la commande qui vérifie ».

TITRE_MD = re.compile(r"^#{1,6}\s+(.*?)\s*#*\s*$")


def bloc_shell_sous(chemin, titre):
    """Le premier bloc clôturé sous un titre donné, tel qu'il est écrit.

    Rend None si le titre ou le bloc manquent — l'appelant en fait un refus nommé
    plutôt qu'une exception : une documentation réorganisée doit faire rougir la
    porte, pas la faire planter.
    """
    lignes = (ROOT / chemin).read_text(encoding="utf-8").splitlines()
    i = next((n for n, l in enumerate(lignes)
              if (m := TITRE_MD.match(l)) and m.group(1).strip() == titre), None)
    if i is None:
        return None
    debut = next((n for n in range(i + 1, len(lignes)) if lignes[n].startswith("```")), None)
    if debut is None:
        return None
    fin = next((n for n in range(debut + 1, len(lignes)) if lignes[n].startswith("```")), None)
    if fin is None:
        return None
    return "\n".join(lignes[debut + 1:fin])


def version_du_bloc(bloc):
    """Le tag que le bloc de vérification du README épingle (`V=v0.3.0`)."""
    m = re.search(r"^\s*V=(v\d+\.\d+\.\d+)\s*$", bloc or "", re.M)
    return m.group(1) if m else None


def outil_absent(*noms):
    """Le motif de saut si un outil manque, None s'ils sont tous là."""
    manquants = [n for n in noms if shutil.which(n) is None]
    return f"outil(s) absent(s) : {', '.join(manquants)}" if manquants else None


def saute(nom, motif):
    return controle(nom, SKIPPED, motif)


def shell(nom, script, echec, cwd=None, env=None, timeout=900):
    """Exécute un script bash et en fait un contrôle.

    `bash --noprofile --norc` : le script vient de la documentation, il ne doit rien
    devoir au profil de la machine qui lance la porte.
    """
    t0 = time.time()
    e = dict(os.environ)
    e.update(env or {})
    try:
        r = subprocess.run(["bash", "--noprofile", "--norc", "-c", script],
                           cwd=cwd or ROOT, capture_output=True, text=True,
                           env=e, timeout=timeout)
    except subprocess.TimeoutExpired:
        return controle(nom, NOGO, f"{echec}\ndépassement de {timeout} s", time.time() - t0)
    d = time.time() - t0
    if r.returncode == 0:
        return controle(nom, GO, script.strip().splitlines()[0] if script.strip() else "", d)
    fin = (r.stdout + r.stderr).strip().splitlines()
    return controle(nom, NOGO, f"{echec}\n" + "\n".join(fin[-12:]), d)


def check_binaires_de_release(version):
    """Les binaires publiés, reconstruits ici avec la LIGNE de build du workflow.

    Le workflow vérifie que le binaire porte son tag et garde ses codes de sortie
    juste avant de publier — la dernière porte avant l'irréversible. La rejouer ici
    la déplace AVANT le tag, où elle peut encore servir à quelque chose.
    """
    nom = "les binaires se construisent, portent le tag et gardent leurs codes"
    t0 = time.time()
    dist = SORTIE / "stage2-dist"
    script = f"""
set -eu
rm -rf {dist} && mkdir -p {dist}
for cible in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
  GOOS="${{cible%%/*}}" GOARCH="${{cible##*/}}" CGO_ENABLED=0 \
    go build -trimpath \
      -ldflags="-s -w -X github.com/stephrobert/pepin/cmd.version={version}" \
      -o "{dist}/pepin-${{cible%%/*}}-${{cible##*/}}" .
done
cd {dist} && sha256sum pepin-* > checksums.txt

got="$(PEPIN_LANG=fr {dist}/pepin-linux-amd64 version)"
[ "$got" = "pépin {version}" ] || {{ echo "en français : '$got', tag '{version}'"; exit 1; }}
got="$(PEPIN_LANG=en {dist}/pepin-linux-amd64 version)"
[ "$got" = "pepin {version}" ] || {{ echo "en anglais : '$got', tag '{version}'"; exit 1; }}

cd {ROOT}
rc=0; {dist}/pepin-linux-amd64 scan scaleway examples/scaleway/inventory.json --format json >/dev/null || rc=$?
[ "$rc" -eq 1 ] || {{ echo "inventaire non conforme : exit $rc, attendu 1"; exit 1; }}
rc=0; {dist}/pepin-linux-amd64 scan scaleway examples/scaleway/inventory-ok.json --format json >/dev/null || rc=$?
[ "$rc" -eq 0 ] || {{ echo "inventaire conforme : exit $rc, attendu 0"; exit 1; }}
"""
    c = shell(nom, script, "la surface publiée a bougé, ou le binaire ne porte pas son tag")
    c["duration_s"] = round(time.time() - t0, 1)
    if c["verdict"] == GO:
        c["evidence"] = f"4 cibles construites dans {dist.relative_to(ROOT)}, tag et codes vérifiés"
    return c


def check_bloc_verification_readme():
    """Le bloc « Verify what you downloaded » du README, exécuté TEL QU'IL EST ÉCRIT.

    Recopier ces commandes ici prouverait que ma copie marche. Les extraire prouve que
    la PAGE marche — et c'est exactement ce que l'ADR-0016 exige : la documentation
    nomme la commande qui vérifie, donc cette commande doit vérifier.
    """
    nom = "le bloc de vérification du README s'exécute tel qu'il est écrit"
    bloc = bloc_shell_sous("README.md", "Verify what you downloaded")
    if bloc is None:
        return controle(nom, NOGO,
                        "README.md : titre « Verify what you downloaded » ou son bloc introuvable — "
                        "la page a été réorganisée et la porte ne sait plus quoi rejouer")
    if (motif := outil_absent("gh", "cosign", "sha256sum")):
        return saute(nom, motif)
    tmp = SORTIE / "stage2-readme"
    return shell(nom, f"rm -rf {tmp} && mkdir -p {tmp} && cd {tmp}\n" + bloc,
                 "la chaîne publiée ne se vérifie plus avec les commandes du README")


def check_bloc_readme_pointe_le_dernier_tag():
    """Le bloc épingle-t-il encore le dernier tag publié ?

    Une page qui montre `V=v0.1.0` fait vérifier à un lecteur une release que personne
    n'utilise, et elle le fait avec l'aplomb d'une instruction officielle. C'est la
    même famille que le défaut #H, sur une autre page.
    """
    nom = "le bloc de vérification du README épingle le dernier tag publié"
    bloc = bloc_shell_sous("README.md", "Verify what you downloaded")
    v = version_du_bloc(bloc)
    prec = tag_precedent()
    if not prec:
        return saute(nom, "aucun tag antérieur : rien à comparer")
    if v is None:
        return controle(nom, NOGO, "le bloc n'épingle aucune version (`V=vX.Y.Z` attendu)")
    if v != prec:
        return controle(nom, NOGO,
                        f"le README fait vérifier {v}, le dernier tag publié est {prec}")
    return controle(nom, GO, f"V={v}")


def check_image_publiee():
    """L'image publiée se vérifie et scanne, avec les commandes de docs/install.md."""
    nom = "l'image publiée se vérifie et scanne un plan, avec les commandes documentées"
    bloc = bloc_shell_sous("docs/install.md", "The container image")
    if bloc is None:
        return controle(nom, NOGO, "docs/install.md : le bloc de « The container image » est introuvable")
    if (motif := outil_absent("cosign", "docker")):
        return saute(nom, motif)
    # Le bloc documenté scanne `/work/plan.json` : on lui en donne un, celui du dépôt.
    tmp = SORTIE / "stage2-image"
    prep = (f"rm -rf {tmp} && mkdir -p {tmp} && "
            f"cp {ROOT}/examples/scaleway/terraform/plan.json {tmp}/plan.json && cd {tmp}\n")
    # `pepin scan` rend 1 sur un plan non conforme : c'est un succès du chemin, pas un
    # échec de la commande. On ne tolère que le 1, jamais le 2 (erreur technique).
    return shell(nom, prep + bloc + "\nrc=$?\n[ $rc -le 1 ] || exit $rc\n",
                 "l'image publiée ne se vérifie plus, ou ne scanne plus un plan")


def check_installeur():
    """`install.sh` accepte le binaire publié, et refuse le même altéré d'un octet.

    Les deux moitiés sont nécessaires et c'est tout l'intérêt : un installeur qui
    accepte tout a exactement l'aspect d'un installeur qui vérifie. La CI éprouve déjà
    ce couple sur des artefacts construits localement ; ici, la moitié « accepte »
    porte sur la chaîne RÉELLEMENT PUBLIÉE — attestation de provenance comprise.
    """
    nom = "l'installeur de l'action accepte le binaire publié et refuse le même altéré"
    if (motif := outil_absent("gh", "curl", "python3")):
        return saute(nom, motif)
    prec = tag_precedent()
    if not prec:
        return saute(nom, "aucun tag antérieur : rien à installer")
    sans_v = prec.removeprefix("v")
    tmp = SORTIE / "stage2-install"
    # Le port se choisit ICI plutôt qu'en grattant le journal du serveur : une porte
    # dont la fiabilité dépend du format d'un message d'outil tiers est une porte qui
    # rougira un matin pour une raison étrangère à ce qu'elle mesure.
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        port = sock.getsockname()[1]
    script = f"""
set -eu
rm -rf {tmp} && mkdir -p {tmp}/bin {tmp}/faux
cd {tmp}

# 1. La chaîne publiée, en entier : provenance puis empreinte.
{ROOT}/.github/actions/pepin-scan/install.sh "{sans_v}" "{tmp}/bin"
[ -x "{tmp}/bin/pepin" ] || {{ echo "installeur : rien d'installé"; exit 1; }}

# 2. Le MÊME binaire, un octet changé, servi en boucle locale avec les vraies sommes.
#    L'attestation est explicitement neutralisée pour isoler ce que ce cas mesure :
#    le refus vient-il bien de l'empreinte ?
cd {tmp}/faux
gh release download "{prec}" --repo stephrobert/pepin \
  --pattern 'pepin-linux-amd64' --pattern 'checksums.txt'
printf '\\0' >> pepin-linux-amd64
python3 -m http.server {port} --bind 127.0.0.1 >{tmp}/httpd.log 2>&1 &
srv=$!
trap 'kill $srv 2>/dev/null || true' EXIT
pret=0
for _ in $(seq 1 50); do
  # `-fs` sans `S` : la sonde de disponibilité échoue par construction tant que le
  # serveur n'a pas démarré, et cinquante messages d'erreur noieraient la preuve.
  curl -fs -o /dev/null "http://127.0.0.1:{port}/checksums.txt" && {{ pret=1; break; }}
  sleep 0.1
done
[ "$pret" = 1 ] || {{ echo "serveur local non démarré"; cat {tmp}/httpd.log; exit 1; }}

rc=0
PEPIN_SKIP_ATTESTATION=1 {ROOT}/.github/actions/pepin-scan/install.sh \
  "{sans_v}" "{tmp}/bin" "http://127.0.0.1:{port}" >{tmp}/altere.log 2>&1 || rc=$?
[ "$rc" -ne 0 ] || {{ echo "l'installeur a ACCEPTÉ un binaire altéré"; cat {tmp}/altere.log; exit 1; }}
"""
    return shell(nom, script,
                 "l'installeur n'accepte plus le binaire publié, ou n'en refuse plus un altéré")


def check_template_gitlab():
    """Le `before_script` du template, dans l'image qu'il déclare lui-même."""
    nom = "le template GitLab installe pepin dans l'image qu'il déclare"
    if (motif := outil_absent("docker")):
        return saute(nom, motif)
    chemin = ROOT / "examples/gitlab-ci/pepin.gitlab-ci.yml"
    tpl = yaml.safe_load(chemin.read_text(encoding="utf-8"))
    base = tpl.get(".pepin", {})
    image = base.get("image")
    avant = base.get("before_script")
    if not image or not avant:
        return controle(nom, NOGO, f"{chemin.relative_to(ROOT)} : `.pepin` sans `image` ou sans `before_script`")
    variables = {**tpl.get("variables", {}), **base.get("variables", {})}
    # GitLab résout ses variables les unes dans les autres : on fait pareil, une passe
    # suffit pour la profondeur que ce template utilise.
    resolues = {}
    for k, v in variables.items():
        for k2, v2 in variables.items():
            v = str(v).replace("${" + k2 + "}", str(v2))
        resolues[k] = v
    exports = "\n".join(f'export {k}="{v}"' for k, v in resolues.items())
    corps = "\n".join(str(l) for l in avant)
    dedans = f"set -eu\n{exports}\n{corps}\npepin version\n"
    script = (f"docker run --rm -i {image} sh -s <<'SCRIPT_DU_TEMPLATE'\n"
              f"{dedans}SCRIPT_DU_TEMPLATE\n")
    return shell(nom, script,
                 f"le before_script du template ne s'exécute plus dans {image}")


def etape2(version):
    return [
        check_binaires_de_release(version),
        check_bloc_verification_readme(),
        check_bloc_readme_pointe_le_dernier_tag(),
        check_image_publiee(),
        check_installeur(),
        check_template_gitlab(),
    ]


ETAPES = {
    1: ("hors ligne : le dépôt, et ce que sa documentation affirme", etape1),
    2: ("réseau : les artefacts tels qu'un utilisateur les reçoit", etape2),
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

    commit = subprocess.run(["git", "rev-parse", "--short", "HEAD"], cwd=ROOT,
                            capture_output=True, text=True).stdout.strip()
    lignes = [
        f"# Porte de release — {version}",
        "",
        f"Lancée le {dt.datetime.now(dt.timezone.utc).isoformat(timespec='seconds')}, "
        f"sur `{commit}`.",
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

    resume, fuites = resume_publiable(version, etapes, final, commit)
    (SORTIE / "SUMMARY.md").write_text(resume, encoding="utf-8")
    if fuites:
        (SORTIE / "SUMMARY.md").write_text(
            "<!-- NON PUBLIABLE — ce résumé nomme des chemins locaux :\n" +
            "\n".join(fuites) + "\n-->\n" + resume, encoding="utf-8")
    return SORTIE / "REPORT.md", fuites


# ── Ce qui peut être PUBLIÉ, et ce qui ne le peut pas ──────────────────────────
#
# Le rapport complet ne se publie pas, et pour deux raisons indépendantes qui
# convergent.
#
# LA PREMIÈRE EST UNE QUESTION DE DONNÉES. `release-gate/` est ignoré par git parce
# que ses artefacts portent les identifiants de ressources d'un compte réel (étape 3),
# des chemins de la machine du mainteneur, et les sorties brutes des outils. Rien de
# tout cela n'a à partir chez un tiers, et le geste « joindre le rapport » est
# exactement celui qui l'y enverrait sans que personne ne l'ait décidé.
#
# LA SECONDE EST UNE QUESTION DE CHAÎNE. L'ADR-0016 pose que TOUT artefact publié
# figure dans `checksums.txt`, donc sous la signature — et le rapport ne peut pas y
# entrer : il est produit LOCALEMENT, avant le tag, alors que `checksums.txt` est
# engendré en CI depuis ce que la CI détient. Un `.md` joint à la release serait un
# artefact couvert par aucun chemin de vérification documenté, et il passerait sous
# `TestEveryPublishedArtefactIsChecksummed`, dont le motif ne reconnaît que
# json/jsonl/txt/bundle. Une garde écrite parce que le SBOM avait glissé sous un
# commentaire ne doit pas laisser glisser un rapport sous une extension.
#
# Ce qui se publie, c'est donc le VERDICT, dans le CORPS de la release — là où vivent
# déjà les notes de version, qui sont une affirmation humaine non signée et que
# personne n'a jamais présentée autrement. Le résumé n'ajoute aucune promesse de
# confiance nouvelle : il dit ce qui a été mesuré, quand, et sur quel commit.
#
# Et il le prouve : `resume_publiable` REND les fuites qu'il détecte au lieu de les
# taire. Un motif de saut est écrit à la main par le mainteneur — c'est précisément par
# là qu'un chemin local entrerait.

FUITE = re.compile(r"(/home/[^\s`\"']+|/Users/[^\s`\"']+|(?<![\w/])/(?:tmp|var|opt|srv)/[^\s`\"']+)")


def fuites_locales(texte):
    """Les fragments d'un texte destiné à la publication qui nomment cette machine."""
    return sorted({m.group(0) for m in FUITE.finditer(texte)})


def resume_publiable(version, etapes, final, commit):
    """Le verdict, sans aucune preuve : nom des contrôles, verdicts, date, commit.

    Rend le couple (markdown, fuites). Une fuite n'est pas corrigée en silence — la
    corriger reviendrait à décider à la place du mainteneur ce qu'il voulait écrire.
    """
    lignes = [
        f"### Porte de release — {version}",
        "",
        f"Verdict : **{'GO' if final == GO else 'NO-GO'}** · "
        f"commit `{commit}` · {dt.datetime.now(dt.timezone.utc).strftime('%Y-%m-%d')}",
        "",
    ]
    for e in etapes:
        lignes.append(f"**Étape {e['stage']} — {e['title']} : {e['verdict'].upper()}**")
        lignes.append("")
        if e["verdict"] == SKIPPED and not e["checks"]:
            lignes += [f"- — sautée : {e.get('skip_reason', '')}", ""]
            continue
        for c in e["checks"]:
            ligne = f"- {SYMBOLE[c['verdict']]} {c['name']}"
            # Un saut est la SEULE chose dont le motif se publie : c'est lui qui dit ce
            # que la porte n'a pas mesuré, et le taire vaudrait un vert sans mesure.
            if c["verdict"] == SKIPPED:
                ligne += f" — sauté : {c['evidence']}"
            lignes.append(ligne)
        lignes.append("")
    lignes.append("_Le rapport détaillé reste local : il porte des chemins de la machine "
                  "qui a lancé la porte, et les identifiants de ressources du tenant de "
                  "qualification._")
    texte = "\n".join(lignes) + "\n"
    return texte, fuites_locales(texte)


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

    # ── Une étape qui n'a RIEN mesuré ne vaut pas GO ───────────────────────────
    #
    # Le piège que l'étape 2 apporte : ses contrôles savent se sauter proprement quand
    # `cosign` ou `docker` manquent, et une machine sans ces outils aurait rendu une
    # étape verte n'ayant rien vérifié. C'est le faux vert que ce produit reproche aux
    # autres, dans sa propre porte.
    tout_saute = [controle("a", SKIPPED, "cosign absent"), controle("b", SKIPPED, "docker absent")]
    veut("tout sauté ne vaut pas GO", verdict_de_letape(tout_saute), SKIPPED)
    veut("un seul mesuré suffit à mesurer",
         verdict_de_letape(tout_saute + [controle("c", GO, "")]), GO)
    veut("un rouge l'emporte sur des sauts",
         verdict_de_letape(tout_saute + [controle("c", NOGO, "cassé")]), NOGO)
    # Un contrôle INDICATIF ne mesure pas au sens de la porte : une étape qui n'a que
    # lui et des sauts n'a toujours rien établi.
    veut("un indicatif ne rachète pas des sauts",
         verdict_de_letape(tout_saute + [controle("c", REPORTED, "en baisse")]), SKIPPED)
    veut("une étape vide reste GO", verdict_de_letape([]), GO)

    # ── L'extraction des commandes DOCUMENTÉES ─────────────────────────────────
    #
    # Recopier ces commandes dans la porte prouverait que la copie marche. Les
    # extraire prouve que la PAGE marche. Une page réorganisée doit donc faire rougir
    # la porte, pas la faire planter : `bloc_shell_sous` rend None, et l'appelant en
    # fait un refus nommé.
    veut("le bloc du README est retrouvé",
         bloc_shell_sous("README.md", "Verify what you downloaded") is not None, True)
    veut("un titre absent rend None",
         bloc_shell_sous("README.md", "Titre qui n'existe pas"), None)
    veut("la version épinglée par le bloc est lue",
         version_du_bloc("V=v0.3.0\ngh release download \"$V\""), "v0.3.0")
    veut("un bloc sans version rend None", version_du_bloc("gh release download"), None)
    veut("un bloc absent ne fait pas planter", version_du_bloc(None), None)

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

    # ── Le résumé PUBLIABLE ne publie jamais les preuves ───────────────────────
    #
    # La propriété qui décide : ce fichier part chez un tiers. Le rapport détaillé
    # porte des chemins de la machine du mainteneur, les sorties brutes des outils et,
    # à l'étape 3, les identifiants de ressources d'un compte réel. Le résumé ne doit
    # rien en emporter — c'est un contrôle de FUITE, pas de mise en forme.
    etapes_test = [{
        "stage": 2, "title": "réseau", "verdict": NOGO, "duration_s": 3,
        "checks": [
            controle("un contrôle vert", GO, "/home/mainteneur/secret/chemin"),
            controle("un contrôle rouge", NOGO, "identifiant-de-ressource-i-0123456789"),
            controle("un contrôle sauté", SKIPPED, "outil(s) absent(s) : cosign"),
        ],
    }]
    resume, fuites = resume_publiable("v1.2.3", etapes_test, NOGO, "abc1234")
    veut("aucune preuve dans le résumé", "/home/mainteneur" in resume, False)
    veut("aucun identifiant de ressource dans le résumé",
         "i-0123456789" in resume, False)
    veut("les noms de contrôle, eux, y sont", "un contrôle rouge" in resume, True)
    veut("le motif d'un SAUT s'y publie", "outil(s) absent(s) : cosign" in resume, True)
    veut("le verdict y est", "NO-GO" in resume, True)
    veut("le commit y est", "abc1234" in resume, True)
    veut("un résumé propre ne signale aucune fuite", fuites, [])

    # Un motif de `--skip` est écrit à la main : c'est par là qu'un chemin local entre.
    etape_sautee = [{"stage": 2, "title": "réseau", "verdict": SKIPPED, "duration_s": 0,
                     "checks": [], "skip_reason": "cosign absent de /home/bob/.local/bin"}]
    _, fuites = resume_publiable("v1.2.3", etape_sautee, GO, "abc1234")
    veut("un chemin local dans un motif de saut est signalé",
         fuites, ["/home/bob/.local/bin"])
    veut("un chemin macOS aussi", fuites_locales("voir /Users/qui/dossier"), ["/Users/qui/dossier"])
    veut("un chemin temporaire aussi", fuites_locales("dans /tmp/run-42"), ["/tmp/run-42"])
    # Le contre-exemple : une porte qui crierait sur tout serait désarmée. Un chemin du
    # DÉPÔT est relatif, et il n'a rien de local.
    veut("un chemin du dépôt ne fuit rien",
         fuites_locales("voir docs/install.md et .github/workflows/release.yml"), [])
    veut("une URL ne fuit rien",
         fuites_locales("https://github.com/stephrobert/pepin/releases"), [])

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
    chemin, fuites = ecris_rapport(version, resultats, final)
    rouges = [str(e["stage"]) for e in resultats if e["verdict"] == NOGO]
    print(f"\n{chemin.relative_to(ROOT)}  (détaillé, LOCAL)")
    print(f"{(SORTIE / 'SUMMARY.md').relative_to(ROOT)}  (verdict seul, à joindre au corps de la release)")
    if fuites:
        print("  ATTENTION : le résumé nomme des chemins de cette machine, il n'est pas "
              "publiable tel quel —")
        for f in fuites:
            print(f"    {f}")
        print("  ils viennent d'un motif de --skip : le réécrire sans chemin local.")
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
