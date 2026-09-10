#!/usr/bin/env python3
"""Étape 3 de la porte de release (issue #178) : le TENANT DE QUALIFICATION.

    PEPIN_GATE_LIVE=1 python3 tools/qualification/qualify.py run scaleway
    python3 tools/qualification/qualify.py run scaleway --plan-only
    python3 tools/qualification/qualify.py compare --run-dir release-gate/qualification-scaleway
    python3 tools/qualification/qualify.py selftest

Une stack Terraform par fournisseur, committée sous references/qualification/<p>/,
délibérément mal configurée : une ressource par contrôle, un contre-exemple par
contrôle. Ce runner l'APPLIQUE sur un compte réel, lance `pepin scan --live` dans
tous les formats, scelle un bundle, le vérifie et le re-dérive, scanne le MÊME plan
avec --terraform, DÉTRUIT, prouve la destruction, puis compare l'assessment à
`expected.yaml` (contrôle × source × sujet → statut). Toute différence est un NO-GO.

# Ce que ce programme refuse, et pourquoi

- De démarrer sans `PEPIN_GATE_LIVE=1` : appliquer un tenant fautif est un geste de
  mainteneur, opt-in, jamais un effet de bord d'une tâche qu'on lance par habitude.
- De démarrer si le compte que les identifiants natifs désignent (demandé à l'API,
  pas lu dans un profil) n'est pas celui qu'`expected.yaml` épingle. Un tenant qui
  ouvre SSH au monde ne s'applique pas sur une production par erreur.
- De dire GO si une ressource du tenant survit au destroy : « Destroy complete! »
  n'est pas une preuve, un listing par famille filtré sur le tag du tenant et un
  delta avant/après le sont. Une ressource laissée vivante coûte plus cher que
  l'absence de la porte (CLAUDE.md §1.1, ADR-0012).
- De dire GO si la porte n'a pas pu être mise en échec : à la fin de chaque run, une
  attente est cassée en mémoire et la comparaison DOIT dire NO-GO. Une porte qu'on
  n'a jamais vue rouge ne garde rien (doctrine de tools/falsify/falsify.py).

# Ce qu'il ne détient jamais

Aucun identifiant : le provider Terraform, `pepin scan --live` et les crochets du
tenant lisent les variables natives du fournisseur ou son fichier de configuration
(ADR-0012). Le mot de passe des bases managées est engendré ici, passé par
l'environnement, et ne sort d'aucune sortie. L'état Terraform vit dans le dossier
de run, non versionné, et tout ce que la porte écrit va dans release-gate/, ignoré
par git : ces artefacts portent les identifiants de ressources d'un compte réel.

# Convention de sortie des OUTILS de release

0 GO · 1 NO-GO (ou refus de démarrer) · 2 erreur d'usage.
"""
import argparse
import copy
import datetime as dt
import json
import os
import pathlib
import re
import secrets
import shutil
import subprocess
import sys
import time

import yaml

ROOT = pathlib.Path(__file__).resolve().parents[2]
STATUSES = {"fail", "pass", "not-evaluated", "not-applicable"}
# `absent` : le contrôle ne doit PAS apparaître dans l'assessment de cette source
# (aucune ressource de son type n'y existe, et l'assessment ne le liste pas).
# `evaluated` : pass OU fail, pour un contrôle dont les sujets sont hors du tenant.
OUTPUT_REF = re.compile(r"^\$\{output\.([A-Za-z0-9_]+)\}$")


# ═══════════════════════════════════════════════════════════════════════════════
# Comparaison — fonctions PURES, éprouvées par `selftest`
# ═══════════════════════════════════════════════════════════════════════════════

# Classes de défaut connu, et ce que chacune vaut pour une RELEASE.
#
# La porte comparait un run à `expected.yaml` et disait GO dès qu'ils coïncidaient.
# C'est un contrat de NON-RÉGRESSION, et c'en est un bon. Mais un défaut épinglé s'y
# reproduit à l'identique, la comparaison ne trouve aucune différence, et la porte
# concluait GO : elle vérifiait que le produit ment de la même façon qu'hier.
#
# Les deux questions sont distinctes et méritent deux verdicts :
#
#   contrat de non-régression : le run dit-il ce qu'on attendait ?
#   verdict de release        : ce qu'on attendait est-il publiable ?
#
# Un `not-evaluated` justifié est une limite de couverture nommée : elle se documente et
# se publie. Un `pass` que rien n'établit, non — c'est l'affirmation que le produit
# promet de ne jamais faire.
CLASSES_DEFAUT = {
    # Un `pass` que rien n'établit. TOUJOURS bloquant, sans dérogation possible :
    # c'est la promesse centrale du produit, et la renier une fois suffit à la perdre.
    "false_green": {"bloquant_toujours": True},
    # Un écart que l'utilisateur ne peut pas faire disparaître — parce que la
    # plateforme l'impose, ou que la remédiation proposée n'aboutit jamais. Bloquant
    # par défaut : dix de ces findings apprennent à ignorer l'outil.
    "false_positive": {"bloquant_toujours": False},
    # Une donnée que la source n'expose pas, rendue « non évalué ». Honnête, nommée,
    # publiable — c'est exactement ce que l'ADR-0006 demande.
    "coverage_gap": {"bloquant_toujours": False, "bloquant_defaut": False},
    # Le verdict est juste, sa DÉCLARATION ne l'est pas (référentiel, matrice).
    # Trompeur pour qui lit la doc, sans jamais mentir sur un tenant.
    "declaration_gap": {"bloquant_toujours": False, "bloquant_defaut": False},
}


def defaut_connu(want):
    """Normalise l'épinglage d'un défaut connu, ou None.

    Deux formes acceptées. Une CHAÎNE, historique : elle ne dit pas sa classe, donc
    elle est traitée comme BLOQUANTE — un épinglage qui ne se prononce pas ne doit pas
    valoir laissez-passer. Un MAPPING `{issue, class, release_blocker, note}` dit ce
    qu'il est, et c'est la forme à écrire.
    """
    kd = want.get("known_defect") if isinstance(want, dict) else None
    if not kd:
        return None
    if isinstance(kd, str):
        return {"issue": None, "class": None, "bloquant": True, "note": kd}
    classe = kd.get("class")
    regles = CLASSES_DEFAUT.get(classe)
    if regles is None:
        return {"issue": kd.get("issue"), "class": classe, "bloquant": True,
                "note": f"classe inconnue {classe!r} (valeurs : {', '.join(sorted(CLASSES_DEFAUT))}) — traitée comme bloquante",
                "invalide": True}
    bloquant = kd.get("release_blocker", regles.get("bloquant_defaut", True))
    if regles["bloquant_toujours"] and not bloquant:
        return {"issue": kd.get("issue"), "class": classe, "bloquant": True,
                "note": f"un défaut de classe {classe} ne se dérroge pas : release_blocker ignoré",
                "invalide": True}
    return {"issue": kd.get("issue"), "class": classe, "bloquant": bool(bloquant), "note": kd.get("note", "")}


class Verdict:
    def __init__(self):
        self.problems = []   # ce qui rend NO-GO le contrat de non-régression
        self.infos = []      # ce qui mérite d'être lu sans rendre NO-GO
        self.defauts = []    # défauts connus rencontrés, normalisés

    @property
    def ok(self):
        return not self.problems

    def problem(self, msg):
        self.problems.append(msg)

    def info(self, msg):
        self.infos.append(msg)


def resolve_subject(subject, outputs):
    """`${output.x}` → sa valeur ; un littéral reste tel quel. None si irrésoluble."""
    m = OUTPUT_REF.match(str(subject))
    if not m:
        return str(subject)
    return outputs.get(m.group(1))


def tenant_subject(subject, prefix, owned):
    """Un sujet appartient-il au tenant : préfixe de nom, valeur d'une sortie, ou
    adresse d'une ressource du plan (la source Terraform désigne par l'adresse ce
    dont l'identifiant n'est pas encore connu)."""
    s = str(subject)
    return s.startswith(prefix) or s in owned


def owned_identifiers(outputs, plan=None):
    """Ce que le tenant possède : les valeurs de ses sorties et les adresses du plan."""
    owned = {str(v) for v in outputs.values()}
    if plan:
        def walk(module):
            for r in module.get("resources", []) or []:
                owned.add(r.get("address", ""))
            for child in module.get("child_modules", []) or []:
                walk(child)
        walk((plan.get("planned_values") or {}).get("root_module") or {})
    return owned


def published_status(results):
    """Le statut qu'un contrôle publie : `fail` si un seul écart, sinon le statut unique."""
    statuses = {r["status"] for r in results}
    if "fail" in statuses:
        return "fail"
    if len(statuses) == 1:
        return statuses.pop()
    return "/".join(sorted(statuses))


def compare(expected, results, source, exit_code, outputs, prefix, owned=None):
    """Confronte un assessment (liste de résultats) à expected.yaml pour UNE source."""
    v = Verdict()
    owned = set(owned) if owned is not None else owned_identifiers(outputs)
    want_rc = expected.get("exit_codes", {}).get(source)
    if want_rc is not None and exit_code != want_rc:
        v.problem(f"[{source}] code de sortie {exit_code}, attendu {want_rc}")

    by_control = {}
    for r in results:
        by_control.setdefault(r["control"], []).append(r)

    pinned = expected.get("controls") or {}
    for code in sorted(set(by_control) - set(pinned)):
        v.problem(f"[{source}] contrôle non épinglé dans expected.yaml : {code} "
                  f"(statut publié {published_status(by_control[code])})")

    for code, spec in sorted(pinned.items()):
        want = spec.get(source) if isinstance(spec, dict) else None
        if want is None:
            v.problem(f"[{source}] {code} : expected.yaml ne dit rien pour cette source")
            continue
        got = by_control.get(code)
        if want == "absent" or (isinstance(want, dict) and want.get("status") == "absent"):
            if got is not None:
                v.problem(f"[{source}] {code} : attendu ABSENT de l'assessment, obtenu {published_status(got)}")
            continue
        if got is None:
            v.problem(f"[{source}] {code} : absent de l'assessment")
            continue
        # Un défaut CONNU est épinglé tel qu'il est mesuré, avec le numéro de l'issue
        # qui le suit : la porte reste GO, et la correction la fera rougir sciemment.
        # Il est dit à chaque run, pour ne jamais passer pour une attente ordinaire.
        d = defaut_connu(want)
        if d:
            d = dict(d, control=code, source=source)
            v.defauts.append(d)
            marque = "BLOQUE LA RELEASE" if d["bloquant"] else "n'empêche pas la release"
            v.info(f"[{source}] {code} : DÉFAUT CONNU ({d['class'] or 'classe non dite'}, {marque})"
                   + (f" — {d['note']}" if d.get("note") else "")
                   + (f" — issue #{d['issue']}" if d.get("issue") else ""))
        # Un sujet que l'attente NOMME est du tenant, même sans préfixe : le
        # fournisseur lui-même, sujet du contrôle de souveraineté, en est le cas.
        named = set()
        if isinstance(want, dict):
            for key in ("fail", "silent", "inconclusive"):
                named |= {resolve_subject(s, outputs) for s in (want.get(key) or [])} - {None}
        tenant_fails = [r for r in got if r["status"] == "fail" and (tenant_subject(r.get("subject", ""), prefix, owned) or r.get("subject", "") in named)]
        foreign_fails = [r for r in got if r["status"] == "fail" and r not in tenant_fails]
        for r in foreign_fails:
            v.info(f"[{source}] {code} : écart HORS tenant sur « {r.get('subject')} » (non gardé)")

        # Forme 1 : un statut au niveau du contrôle (chaîne, ou mapping avec `status:`).
        #
        # Une attente qui NOMME des sujets (`fail`/`silent`/`inconclusive`) relève de la
        # forme 2, même si elle porte aussi `status:`. La forme 2 est strictement plus
        # précise — elle dit QUI doit être fautif —, et la faire avaler par la forme 1
        # perdait les sujets en silence.
        #
        # Trouvé par le premier run réel Exoscale : l'attente de
        # `kubernetes_cluster_audit_logging_enabled` porte `status: fail` ET
        # `fail: [${output.sks_weak_name}]`. La forme 1 la prenait, constatait un écart
        # sur le tenant, et le déclarait « faux positif ou attente périmée » — alors que
        # cet écart est très exactement ce que `fail` veut dire. Le produit avait
        # raison, le runner avait tort, et le NO-GO était le sien.
        nomme_des_sujets = isinstance(want, dict) and any(
            k in want for k in ("fail", "silent", "inconclusive"))
        if not nomme_des_sujets and (isinstance(want, str) or (isinstance(want, dict) and "status" in want)):
            status = want if isinstance(want, str) else want["status"]
            if status not in STATUSES | {"evaluated"}:
                v.problem(f"[{source}] {code} : statut attendu inconnu « {status} »")
                continue
            # `fail` attendu : un écart sur le tenant CONFIRME l'attente, il ne la
            # contredit pas. C'est son absence qui est le faux vert. Le cas ne se
            # présentait pas encore sans sujets nommés, et c'est justement pourquoi il
            # fallait le traiter : le piège attendait la prochaine attente écrite.
            if status == "fail":
                if not tenant_fails:
                    v.problem(f"[{source}] {code} : FAUX VERT — attendu fail, aucun écart "
                              "sur le tenant")
                continue
            if tenant_fails:
                v.problem(f"[{source}] {code} : attendu {status}, mais écart sur "
                          + ", ".join(sorted({r.get('subject', '') for r in tenant_fails}))
                          + " — faux positif, ou attente périmée")
                continue
            # Sans écart sur le tenant, le statut publié est celui des résultats
            # restants ; un écart hors tenant prouve seulement que le contrôle a conclu.
            rest = [r for r in got if r["status"] != "fail"]
            actual = published_status(rest) if rest else "evaluated"
            if status == "evaluated":
                if actual not in ("pass", "evaluated") and not foreign_fails:
                    v.problem(f"[{source}] {code} : attendu qu'il CONCLUE (pass/fail), obtenu {actual}")
            elif actual != status:
                v.problem(f"[{source}] {code} : attendu {status}, obtenu {actual}"
                          + (" — un pass qui apparaît sans changement du collecteur est un pass non prouvé" if actual == "pass" and status == "not-evaluated" else "")
                          + (" — régression de couverture" if actual == "not-evaluated" and status in ("pass", "fail") else ""))
            continue

        # Forme 2 : des sujets fautifs (`fail:`) et des contre-exemples (`silent:`).
        if not isinstance(want, dict) or "fail" not in want:
            v.problem(f"[{source}] {code} : attente illisible ({want!r})")
            continue
        want_fail, unresolved = [], []
        for s in want.get("fail") or []:
            r = resolve_subject(s, outputs)
            (want_fail if r is not None else unresolved).append(r if r is not None else s)
        want_silent, unresolved_silent = [], []
        for s in want.get("silent") or []:
            r = resolve_subject(s, outputs)
            (want_silent if r is not None else unresolved_silent).append(r if r is not None else s)
        for s in unresolved + unresolved_silent:
            v.problem(f"[{source}] {code} : sujet irrésoluble {s} (sortie Terraform absente)")
        failing = {r.get("subject", "") for r in tenant_fails}
        for s in want_fail:
            if s not in failing:
                v.problem(f"[{source}] {code} : FAUX VERT — aucun écart sur « {s} », qui est fautif")
        for s in sorted(failing - set(want_fail)):
            v.problem(f"[{source}] {code} : FAUX POSITIF — écart sur « {s} », que rien n'attend")
        for s in want_silent:
            if s in failing:
                v.problem(f"[{source}] {code} : le CONTRE-EXEMPLE « {s} » parle — la règle ne discrimine pas")
        # `inconclusive:` — les sujets du tenant sur lesquels la règle DIT ne pas savoir
        # conclure (ADR-0015 : un finding inconcluant devient un not-evaluated à sujet).
        # Chacun doit être là, et aucun autre sujet du tenant ne doit l'être.
        want_inc = []
        for s in want.get("inconclusive") or []:
            r = resolve_subject(s, outputs)
            if r is None:
                v.problem(f"[{source}] {code} : sujet irrésoluble {s} (sortie Terraform absente)")
            else:
                want_inc.append(r)
        inconclusive = {r.get("subject", "") for r in got
                        if r["status"] == "not-evaluated" and tenant_subject(r.get("subject", ""), prefix, owned)}
        for s in want_inc:
            if s not in inconclusive:
                v.problem(f"[{source}] {code} : attendu « ne sait pas conclure » sur « {s} », rien d'inconcluant")
        for s in sorted(inconclusive - set(want_inc)):
            v.problem(f"[{source}] {code} : « ne sait pas conclure » sur « {s} », que rien n'attend")
        others = {r["status"] for r in got if r["status"] != "fail" and r.get("subject", "") not in inconclusive}
        if others and not tenant_fails and not foreign_fails:
            v.problem(f"[{source}] {code} : attendu des écarts, obtenu {published_status(got)}")
    return v


def load_results(path):
    doc = json.loads(pathlib.Path(path).read_text())
    return [{"control": r["control"], "status": r["status"], "subject": r.get("subject", "")}
            for r in doc.get("results", [])]


def falsify(expected, results, source, exit_code, outputs, prefix, owned=None):
    """Casse une attente et exige le NO-GO. Rend (ok, description)."""
    mutants = []
    for code, spec in (expected.get("controls") or {}).items():
        want = spec.get(source)
        if isinstance(want, dict) and want.get("fail"):
            m = copy.deepcopy(expected)
            m["controls"][code][source] = "pass"
            mutants.append((f"{code} : les écarts attendus deviennent « pass »", m))
            break
    for code, spec in (expected.get("controls") or {}).items():
        if spec.get(source) == "not-evaluated":
            m = copy.deepcopy(expected)
            m["controls"][code][source] = "pass"
            mutants.append((f"{code} : « not-evaluated » attendu devient « pass »", m))
            break
    m = copy.deepcopy(expected)
    m.setdefault("exit_codes", {})[source] = 0
    mutants.append(("code de sortie attendu 0", m))
    outcomes = []
    for label, mutant in mutants:
        v = compare(mutant, results, source, exit_code, outputs, prefix, owned)
        outcomes.append({"mutation": label, "no_go": not v.ok, "first_problem": (v.problems or [""])[0]})
    return all(o["no_go"] for o in outcomes), outcomes


# ═══════════════════════════════════════════════════════════════════════════════
# Le run
# ═══════════════════════════════════════════════════════════════════════════════

class Run:
    def __init__(self, provider, run_dir, plan_only):
        self.provider = provider
        self.tenant_dir = ROOT / "references" / "qualification" / provider
        self.run_dir = run_dir
        self.plan_only = plan_only
        self.stages = []
        self.t0 = time.time()
        self.applied = False
        self.destroyed = False
        self.wait_until = None
        self.tenant = yaml.safe_load((self.tenant_dir / "tenant.yaml").read_text())
        self.expected = yaml.safe_load((self.tenant_dir / "expected.yaml").read_text())
        self.outputs = {}
        self.pepin = run_dir / "pepin"
        self.env = dict(os.environ, TF_IN_AUTOMATION="1", CHECKPOINT_DISABLE="1",
                        NO_COLOR="1", TERM="dumb", PEPIN_LANG="en")
        # Une socket unix est bornée à 108 octets : un TMPDIR profond fait échouer
        # le greffon du provider avec un « bind: invalid argument » sans rapport.
        if len(self.env.get("TMPDIR", "")) > 40:
            self.env["TMPDIR"] = "/tmp"

    # ─── journal ───────────────────────────────────────────────────────────
    def stage(self, name, fn):
        t = time.time()
        print(f"\n── {name}")
        try:
            note = fn()
            rc = 0
        except StageFailed as e:
            note, rc = str(e), 1
            print(f"  ✘ {e}")
        self.stages.append({"name": name, "rc": rc, "seconds": round(time.time() - t, 1), "note": note})
        return rc == 0

    def sh(self, args, cwd=None, capture=True, check=True, env=None, log=None):
        r = subprocess.run(args, cwd=cwd or self.tenant_dir, env=env or self.env,
                           capture_output=capture, text=True)
        if log:
            (self.run_dir / log).write_text((r.stdout or "") + (r.stderr or ""))
        if check and r.returncode != 0:
            tail = ((r.stderr or r.stdout or "")[-1500:]).strip()
            raise StageFailed(f"`{' '.join(map(str, args[:3]))}…` rc={r.returncode}\n{tail}")
        return r

    def hook(self, *args):
        r = self.sh([sys.executable, str(self.tenant_dir / "hooks.py"), *args], cwd=ROOT)
        return json.loads(r.stdout)

    # ─── étapes ────────────────────────────────────────────────────────────
    def preflight(self):
        # Un tenant dont tenant.yaml dit `live: unavailable` n'a pas de compte : il
        # s'éprouve en plan seul, et un run réel est REFUSÉ — il n'y aurait ni
        # identité à confronter ni preuve de destruction possible.
        self.no_account = str(self.tenant.get("live", "")).lower() == "unavailable"
        if self.no_account and not self.plan_only:
            raise StageFailed("REFUS : tenant.yaml déclare `live: unavailable` (aucun compte) — seul `--plan-only` est possible")
        for tool in ("terraform", "go"):
            if not shutil.which(tool):
                raise StageFailed(f"{tool} absent du PATH (requis)")
        for tool in self.tenant.get("required_tools") or []:
            if not shutil.which(tool):
                raise StageFailed(f"{tool} absent du PATH (tenant.yaml `required_tools` : chemin de secours du nettoyage)")
        lock = self.tenant_dir / ".terraform.lock.hcl"
        want = str(self.tenant["provider_version"])
        if not lock.exists() or f'version     = "{want}"' not in lock.read_text() and f'version = "{want}"' not in lock.read_text():
            raise StageFailed(f"le lockfile n'épingle pas le provider {want} : une contrainte flottante est exactement ce que ce tenant refuse")
        self.sh(["go", "build", "-trimpath", "-o", str(self.pepin), "."], cwd=ROOT)
        ver = self.sh([str(self.pepin), "version"], cwd=ROOT).stdout.strip()
        tf = json.loads(self.sh(["terraform", "version", "-json"]).stdout)["terraform_version"]
        return f"pepin {ver} · terraform {tf} · provider {want}"

    def identity(self):
        """Le compte que les identifiants natifs ouvrent, demandé à l'API par le crochet
        du tenant, confronté champ par champ à ce que l'environnement attend. Les champs
        sont ceux qu'`expected.yaml` nomme (`organization_id`/`project_id` chez Scaleway,
        `account_id` chez Outscale…) : le runner n'en connaît aucun par avance."""
        if self.no_account:
            (self.run_dir / "identity.json").write_text("{}")
            return "sauté : aucun compte (tenant.yaml `live: unavailable`), plan seul"
        who = self.hook("identity")
        (self.run_dir / "identity.json").write_text(json.dumps(who, indent=2, sort_keys=True))
        acct = self.compte_attendu()
        for champ, attendu in acct.items():
            observe = str(who.get(champ, ""))
            if observe != attendu:
                raise StageFailed(
                    f"REFUS : les identifiants natifs désignent {champ}={observe or '?'}, et le "
                    f"compte attendu est {champ}={attendu}. Rien n'a été appliqué.")
            self.env[f"TF_VAR_{champ}"] = attendu
        return "compte confirmé par l'API : " + ", ".join(f"{k}={v}" for k, v in acct.items()) \
            + (f" ({who['label']})" if who.get("label") else "")

    def variables(self):
        """Les variables PROPRES au tenant (échéances, secrets jetables, principaux…),
        calculées par son crochet `variables` et passées à Terraform par
        l'environnement. Le crochet peut aussi demander une ATTENTE avant le scan
        (`wait_until`, RFC 3339) : une clé qui doit être expirée au moment du scan,
        par exemple. Rien de tout cela n'est écrit ni journalisé."""
        got = self.hook("variables", "--plan-only" if self.plan_only else "--live", str(self.run_dir / "identity.json"))
        for k, v in (got.get("tf_vars") or {}).items():
            self.env[f"TF_VAR_{k}"] = str(v)
        for k, v in (got.get("env") or {}).items():
            self.env[k] = str(v)
        self.wait_until = None
        if got.get("wait_until"):
            self.wait_until = dt.datetime.fromisoformat(got["wait_until"].replace("Z", "+00:00"))
        return got.get("note") or f"{len(got.get('tf_vars') or {})} variable(s) posée(s)"

    def compte_attendu(self):
        """Le compte sur lequel la porte accepte de tourner, lu dans l'ENVIRONNEMENT.

        `expected.yaml` nomme les variables plutôt que les identifiants : le dépôt est
        public, et un `organization_id` committé nommerait durablement le tenant du
        mainteneur. Ce n'est pas un secret — il n'ouvre rien sans identifiants —, mais
        un identifiant publié ne se dépublie pas.

        La garde ne perd rien : elle compare toujours ce que l'API répond à ce qui est
        attendu. Seule la PROVENANCE de l'attendu change. Une variable absente est un
        refus, jamais un laissez-passer : sans attendu, il n'y a rien à confronter, et
        appliquer un tenant délibérément fautif sur un compte non confirmé est
        exactement ce que cette étape existe pour empêcher.
        """
        acct = self.expected.get("account") or {}
        out = {}
        champs = [k[:-4] for k in acct if k.endswith("_env")]
        if not champs:
            raise StageFailed("expected.yaml ne nomme aucune variable de compte (clés `<champ>_env`)")
        for champ in champs:
            var = acct[f"{champ}_env"]
            val = os.environ.get(var, "").strip()
            if not val:
                raise StageFailed(
                    f"REFUS : la variable {var} n'est pas posée, donc le compte attendu est "
                    "inconnu. Rien n'a été appliqué — un tenant délibérément fautif ne "
                    "s'applique pas sur un compte que personne n'a confirmé.")
            out[champ] = val
        return out

    def snapshot_before(self):
        if self.no_account:
            return "sauté : aucun compte, rien à inventorier"
        inv = self.hook("inventory", str(self.tenant_dir / "tenant.yaml"))
        (self.run_dir / "inventory-before.json").write_text(json.dumps(inv, indent=2, sort_keys=True))
        owned = {f: v["tenant"] for f, v in inv.items() if v["tenant"]}
        if owned:
            raise StageFailed("des ressources du tenant existent DÉJÀ (run précédent non nettoyé ?) : "
                              + ", ".join(f"{f}×{len(v)}" for f, v in owned.items())
                              + " — lancer `hooks.py cleanup` avant de recommencer")
        return "aucune ressource du tenant avant apply ; " + ", ".join(f"{f}={len(v['all'])}" for f, v in sorted(inv.items()) if v["all"]) or "compte vide"

    def plan(self):
        state = self.run_dir / "terraform.tfstate"
        # `-reconfigure` : le backend local pointe vers le dossier du run ; un init
        # précédent (à la main, dans le dossier du tenant) aurait laissé une autre
        # configuration, et Terraform refuserait de la changer sans le dire.
        self.sh(["terraform", "init", "-input=false", "-lockfile=readonly", "-reconfigure",
                 f"-backend-config=path={state}"], log="terraform-init.log")
        self.sh(["terraform", "validate"])

        # DEUX plans, parce que les deux passes ne mesurent pas la même chose. Le plan
        # COMPLET porte aussi ce que seul le chemin Terraform sait lire (bases managées,
        # réseaux privés, politiques IAM) : c'est lui que `--terraform` scanne. Le plan
        # APPLIQUÉ s'en tient aux types que la collecte live produit : provisionner le
        # reste coûterait de l'argent qu'aucun scan live ne mesurerait.
        def one(label, extra, out, js):
            self.sh(["terraform", "plan", "-input=false", *extra, f"-out={self.run_dir / out}"], log=f"terraform-plan-{label}.log")
            plan = self.sh(["terraform", "show", "-json", str(self.run_dir / out)]).stdout
            (self.run_dir / js).write_text(plan)
            d = json.loads(plan)
            counts = {}
            for r in d.get("resource_changes", []):
                if "create" in r["change"]["actions"]:
                    counts[r["type"]] = counts.get(r["type"], 0) + 1
            return d, counts

        full, self.counts = one("full", [], "tfplan-full.bin", "plan.json")
        for name, o in (full.get("planned_values", {}).get("outputs") or {}).items():
            if "value" in o:
                self.outputs[name] = o["value"]
        note = f"plan complet : {sum(self.counts.values())} ressources (" + ", ".join(f"{t}×{n}" for t, n in sorted(self.counts.items())) + ")"
        if not self.plan_only:
            _, self.counts_applied = one("applied", self.applied_vars(), "tfplan.bin", "plan-applied.json")
            note += f" ; plan appliqué : {sum(self.counts_applied.values())} ressources"
        return note

    def applied_vars(self, extra=None):
        """Les `-var` de la stack APPLIQUÉE (tenant.yaml `applied_vars`), plus ceux
        d'une étape (pré-destroy). Le plan complet, lui, n'en reçoit aucun."""
        vals = dict(self.tenant.get("applied_vars") or {})
        vals.update(extra or {})
        out = []
        for k, v in vals.items():
            out += ["-var", f"{k}={json.dumps(v) if isinstance(v, bool) else v}"]
        return out

    def apply(self):
        self.applied = True  # dès qu'on tente : un apply à moitié fait se détruit aussi
        self.apply_started = time.time()
        self.sh(["terraform", "apply", "-input=false", "-auto-approve", str(self.run_dir / "tfplan.bin")], log="terraform-apply.log")
        out = json.loads(self.sh(["terraform", "output", "-json"]).stdout)
        self.outputs = {k: v["value"] for k, v in out.items() if not v.get("sensitive")}
        (self.run_dir / "outputs.json").write_text(json.dumps(self.outputs, indent=2, sort_keys=True))
        return f"{len(self.outputs)} sorties capturées"

    def extra_apply(self):
        """Ce que Terraform ne sait pas exprimer chez ce fournisseur (un bucket OOS, une
        politique EIM inline…), créé par le crochet `extra` du tenant, APRÈS l'apply et
        à partir de ses sorties. Le crochet rend la liste de ce qu'il a créé ; la preuve
        de destruction le relit comme le reste."""
        self.extra_applied = True
        got = self.hook("extra", "apply", str(self.run_dir / "outputs.json"))
        created = got.get("created") or []
        (self.run_dir / "extra.json").write_text(json.dumps(got, indent=2, sort_keys=True))
        return f"{len(created)} ressource(s) hors Terraform" + (" : " + ", ".join(created[:8]) if created else "")

    def extra_destroy(self):
        got = self.hook("extra", "destroy", str(self.run_dir / "outputs.json"))
        left = got.get("left") or []
        if left:
            raise StageFailed("le crochet n'a pas pu supprimer : " + ", ".join(left))
        return f"{len(got.get('deleted') or [])} ressource(s) hors Terraform supprimée(s)"

    def scan(self, args, out, log):
        r = self.sh([str(self.pepin), "scan", self.provider, *args], cwd=ROOT, check=False)
        (self.run_dir / out).write_text(r.stdout)
        (self.run_dir / log).write_text(r.stderr)
        return r.returncode

    def wait_for(self):
        """Ce que le crochet `variables` a demandé d'attendre avant de scanner (une
        échéance de clé, par exemple), plus une marge."""
        if not self.wait_until:
            return "rien à attendre"
        target = self.wait_until + dt.timedelta(seconds=20)
        delay = (target - dt.datetime.now(dt.timezone.utc)).total_seconds()
        if delay > 0:
            time.sleep(delay)
        return f"échéance {self.wait_until.strftime('%H:%M:%SZ')} passée" + (f" (attente {round(delay)}s)" if delay > 0 else "")

    def scan_live(self):
        region = self.tenant["region"]
        base = ["--live", "--region", region]
        bundle = self.run_dir / "bundle"
        self.rc_live = self.scan(base + ["--format", "assessment", "--seal", str(bundle)], "assessment-live.json", "scan-live-assessment.err")
        rcs = {"assessment": self.rc_live}
        for fmt, ext in (("table", "txt"), ("json", "json"), ("oscal", "json"), ("sarif", "json")):
            rcs[fmt] = self.scan(base + ["--format", fmt], f"live-{fmt}.{ext}", f"scan-live-{fmt}.err")
        if len(set(rcs.values())) != 1:
            raise StageFailed(f"les formats ne rendent pas le même code de sortie : {rcs}")
        json.loads((self.run_dir / "assessment-live.json").read_text())  # doit être lisible
        return f"5 formats, code de sortie {self.rc_live} partout"

    def bundle(self):
        bundle = self.run_dir / "bundle"
        r = self.sh([str(self.pepin), "verify", str(bundle), "--re-derive"], cwd=ROOT, check=False)
        if r.returncode != 0:
            raise StageFailed(f"verify --re-derive rc={r.returncode} : {r.stdout[-400:]}{r.stderr[-400:]}")
        # Et le refus doit exister : un verify qui accepte tout ne prouve rien (ADR-0018).
        tampered = self.run_dir / "bundle-tampered"
        if tampered.exists():
            shutil.rmtree(tampered)
        shutil.copytree(bundle, tampered)
        with open(tampered / "assessment.json", "a") as f:
            f.write(" ")
        r = self.sh([str(self.pepin), "verify", str(tampered)], cwd=ROOT, check=False)
        if r.returncode == 0:
            raise StageFailed("verify ACCEPTE un bundle altéré d'un octet")
        return "scellé, vérifié, re-dérivé ; l'altération d'un octet est refusée"

    def scan_terraform(self):
        plan = self.run_dir / "plan.json"
        self.rc_tf = self.scan(["--terraform", str(plan), "--format", "assessment"], "assessment-terraform.json", "scan-terraform-assessment.err")
        self.scan(["--terraform", str(plan), "--format", "table"], "terraform-table.txt", "scan-terraform-table.err")
        json.loads((self.run_dir / "assessment-terraform.json").read_text())
        return f"code de sortie {self.rc_tf}"

    def destroy(self):
        if not self.applied:
            return "rien n'a été appliqué"
        note = ""
        # Ce qu'il faut DÉFAIRE avant de détruire : une protection contre la suppression
        # (Outscale refuse de supprimer une VM protégée, provider #88) se lève par un
        # apply, jamais par le destroy. tenant.yaml le déclare (`pre_destroy_vars`).
        pre = self.tenant.get("pre_destroy_vars") or {}
        if pre:
            r0 = self.sh(["terraform", "apply", "-input=false", "-auto-approve", *self.applied_vars(pre)], check=False, log="terraform-pre-destroy.log")
            note += f"pré-destroy ({', '.join(f'{k}={v}' for k, v in pre.items())}) rc={r0.returncode} ; "
        r = self.sh(["terraform", "destroy", "-input=false", "-auto-approve", *self.applied_vars(pre)], check=False, log="terraform-destroy.log")
        note += f"terraform destroy rc={r.returncode}"
        if r.returncode != 0:
            print("  ✘ destroy en échec : nettoyage de secours par l'API, puis second destroy")
            rescue = subprocess.run([sys.executable, str(self.tenant_dir / "hooks.py"), "cleanup",
                                     str(self.tenant_dir / "tenant.yaml")], cwd=ROOT, env=self.env)
            r2 = self.sh(["terraform", "destroy", "-input=false", "-auto-approve", "-refresh=true", *self.applied_vars(pre)], check=False, log="terraform-destroy-2.log")
            note += f" ; cleanup rc={rescue.returncode} ; second destroy rc={r2.returncode}"
        left = self.sh(["terraform", "state", "list"], check=False).stdout.strip().splitlines()
        self.destroy_finished = time.time()
        self.destroyed = not left
        if left:
            raise StageFailed(note + f" ; {len(left)} ressource(s) encore dans l'état : " + ", ".join(left[:8]))
        # L'état et sa sauvegarde portent les clés secrètes des clés d'API du tenant —
        # mortes avec lui, mais inutiles sur disque une fois l'état vide.
        purged = 0
        for f in self.run_dir.glob("terraform.tfstate*"):
            f.unlink()
            purged += 1
        return note + f" ; état vide ; {purged} fichier(s) d'état purgé(s)"

    def leftovers(self):
        before = json.loads((self.run_dir / "inventory-before.json").read_text())

        def measure():
            inv = self.hook("inventory", str(self.tenant_dir / "tenant.yaml"))
            owned = {f: v["tenant"] for f, v in inv.items() if v["tenant"]}
            delta = {}
            for f, v in inv.items():
                new = sorted(set(v["all"]) - set(before.get(f, {}).get("all", [])))
                if new:
                    delta[f] = new
            return inv, owned, delta

        # Une suppression est ASYNCHRONE chez plus d'un fournisseur : mesuré chez Outscale,
        # load balancers, volumes et snapshots restaient listés 26 s après un destroy
        # réussi et « n'existaient plus » deux minutes plus tard. Le vide se constate
        # donc en relisant, bornée : ce qui reste au bout du délai est un reste.
        deadline = time.time() + (self.tenant.get("settle_seconds") or 240)
        waited = 0
        while True:
            inv, owned, delta = measure()
            if not owned and not delta or time.time() >= deadline:
                break
            time.sleep(20)
            waited += 20
        (self.run_dir / "inventory-after.json").write_text(json.dumps(inv, indent=2, sort_keys=True))
        self.leftover_report = {"tagged": owned, "delta": delta}
        if owned or delta:
            # Un reste est un défaut du chemin de destruction, donc un NO-GO. Mais il
            # coûte à l'heure : le crochet de nettoyage est tenté sur ce qui porte le tag
            # ET sur ce qui est apparu pendant le run, puis le compte est relu.
            (self.run_dir / "leftovers.json").write_text(json.dumps(self.leftover_report, indent=2, sort_keys=True))
            rescue = subprocess.run([sys.executable, str(self.tenant_dir / "hooks.py"), "cleanup",
                                     str(self.tenant_dir / "tenant.yaml"), str(self.run_dir / "leftovers.json")], cwd=ROOT, env=self.env)
            after = self.hook("inventory", str(self.tenant_dir / "tenant.yaml"))
            still = {f: v["tenant"] for f, v in after.items() if v["tenant"]}
            still_delta = {f: sorted(set(v["all"]) - set(before.get(f, {}).get("all", []))) for f, v in after.items()}
            still_delta = {f: v for f, v in still_delta.items() if v}
            self.leftover_report["after_cleanup"] = {"tagged": still, "delta": still_delta, "cleanup_rc": rescue.returncode}
            raise StageFailed("RESTES après destroy — " + "; ".join(
                [f"{f} (tag) : {[x['id'] for x in v]}" for f, v in owned.items()]
                + [f"{f} (delta) : {v}" for f, v in delta.items()])
                + (" — nettoyage de secours : compte propre" if not still and not still_delta
                   else f" — nettoyage de secours INCOMPLET : {still} {still_delta}"))
        families = sum(1 for _ in inv)
        return f"{families} familles listées, aucune ressource du tenant, aucun delta avant/après" + (f" (après {waited}s de suppressions asynchrones)" if waited else "")

    def compare_all(self):
        prefix = self.tenant["name_prefix"]
        owned = owned_identifiers(self.outputs, json.loads((self.run_dir / "plan.json").read_text()))
        self.comparison = {}
        problems = 0
        sources = [("terraform", "assessment-terraform.json", getattr(self, "rc_tf", None))]
        if not self.plan_only:
            sources.insert(0, ("live", "assessment-live.json", getattr(self, "rc_live", None)))
        for source, fname, rc in sources:
            results = load_results(self.run_dir / fname)
            v = compare(self.expected, results, source, rc, self.outputs, prefix, owned)
            self.comparison[source] = {"ok": v.ok, "problems": v.problems, "infos": v.infos,
                                       "known_defects": v.defauts,
                                       "results": len(results), "exit_code": rc}
            for p in v.problems:
                print(f"  ✘ {p}")
            for i in v.infos:
                print(f"  · {i}")
            problems += len(v.problems)
            # La porte se prouve en échouant : une attente cassée doit rendre NO-GO.
            ok, outcomes = falsify(self.expected, results, source, rc, self.outputs, prefix, owned)
            self.comparison[source]["falsification"] = outcomes
            if not ok:
                raise StageFailed(f"[{source}] la porte n'a PAS su dire NO-GO sur une attente cassée : {outcomes}")
            print(f"  ✔ [{source}] {len(outcomes)} attentes cassées en mémoire, {len(outcomes)} NO-GO : la porte sait rougir")
        if problems:
            raise StageFailed(f"{problems} différence(s) avec expected.yaml")
        return "assessments conformes à expected.yaml sur " + " et ".join(s for s, _, _ in sources)

    # ─── rapport ───────────────────────────────────────────────────────────
    def cost(self):
        if not getattr(self, "apply_started", None):
            return None
        hours = (getattr(self, "destroy_finished", time.time()) - self.apply_started) / 3600
        rate = sum(l["count"] * l["unit_price"] for l in self.tenant.get("cost", {}).get("hourly", []))
        return {"currency": self.tenant.get("cost", {}).get("currency", "EUR"),
                "billed_hours": round(hours, 3), "hourly_rate": round(rate, 4),
                "estimate": round(hours * rate, 4),
                "note": "estimation : durée apply→destroy × tarifs horaires de tenant.yaml ; la facture à 48 h tranche"}

    def report(self, verdict, release=None, bloquants=None):
        duration = round(time.time() - self.t0, 1)
        release = release or verdict
        bloquants = bloquants or []
        tous = [d for src in getattr(self, "comparison", {}).values() for d in src.get("known_defects", [])]
        par_classe = {}
        for d in tous:
            par_classe.setdefault(d["class"] or "classe non dite", []).append(d)
        rep = {
            "stage": f"qualification-{self.provider}",
            "regression_contract": verdict,
            "verdict": release,
            "known_defects": tous,
            "plan_only": self.plan_only,
            "started": dt.datetime.fromtimestamp(self.t0, dt.timezone.utc).isoformat(timespec="seconds"),
            "duration_seconds": duration,
            "budget_minutes": self.tenant.get("budget_minutes"),
            "account": self.expected.get("account"),
            "resources_planned": getattr(self, "counts", {}),
            "resources_applied": getattr(self, "counts_applied", {}),
            "stages": self.stages,
            "comparison": getattr(self, "comparison", {}),
            "leftovers": getattr(self, "leftover_report", None),
            "cost": self.cost(),
            "destroyed": self.destroyed,
        }
        out = self.run_dir.parent / f"qualification-{self.provider}.json"
        out.write_text(json.dumps(rep, indent=2, ensure_ascii=False))
        lines = [f"# Qualification {self.provider} — {release}", ""]
        for s in self.stages:
            lines.append(f"- {'✔' if s['rc'] == 0 else '✘'} {s['name']} ({s['seconds']}s) — {s['note']}")
        # Les deux verdicts, côte à côte : le contrat de non-régression répond « le run
        # dit-il ce qu'on attendait », le verdict de release « ce qu'on attend est-il
        # publiable ». Les afficher ensemble empêche de lire l'un pour l'autre.
        lines += ["", f"- contrat de non-régression : {verdict}"]
        if tous:
            lines.append("- défauts connus :")
            for classe in sorted(par_classe):
                ds = par_classe[classe]
                bloque = sum(1 for d in ds if d["bloquant"])
                marque = "❌" if bloque else "⚠"
                lines.append(f"    {marque} {classe} : {len(ds)}" + (f" (dont {bloque} bloquant(s))" if bloque else ""))
            for d in bloquants:
                ref = f" (#{d['issue']})" if d.get("issue") else ""
                lines.append(f"    ↳ BLOQUE : {d['control']} [{d['source']}]{ref}"
                             + (f" — {d['note']}" if d.get("note") else ""))
        else:
            lines.append("- défauts connus : aucun")
        lines.append(f"- VERDICT DE RELEASE : {release}")
        c = rep["cost"]
        if c:
            lines.append(f"- coût estimé : {c['estimate']} {c['currency']} ({c['billed_hours']} h × {c['hourly_rate']} {c['currency']}/h)")
        lines.append(f"- durée totale : {duration}s (budget {rep['budget_minutes']} min)")
        (self.run_dir.parent / f"qualification-{self.provider}.md").write_text("\n".join(lines) + "\n")
        print("\n" + "\n".join(lines))
        print(f"\nrapport : {out}")

    # ─── enchaînement ──────────────────────────────────────────────────────
    def go(self):
        self.run_dir.mkdir(parents=True, exist_ok=True)
        ok = True
        try:
            ok &= self.stage("préflight : outils, lockfile, binaire", self.preflight)
            ok = ok and self.stage("identité : le compte que les identifiants désignent", self.identity)
            ok = ok and self.stage("variables du tenant", self.variables)
            ok = ok and self.stage("inventaire AVANT", self.snapshot_before)
            ok = ok and self.stage("plan", self.plan)
            if ok and not self.plan_only:
                ok = ok and self.stage("apply", self.apply)
                ok = ok and self.stage("hors Terraform : ce que le crochet crée", self.extra_apply)
                ok = ok and self.stage("attente demandée par le tenant", self.wait_for)
                ok = ok and self.stage("scan --live, 5 formats, bundle scellé", self.scan_live)
                ok = ok and self.stage("bundle : verify --re-derive, refus de l'altération", self.bundle)
            if ok:
                ok = ok and self.stage("scan --terraform sur le même plan", self.scan_terraform)
        finally:
            if getattr(self, "extra_applied", False):
                ok = self.stage("hors Terraform : ce que le crochet supprime", self.extra_destroy) and ok
            if self.applied:
                ok = self.stage("destroy", self.destroy) and ok
                ok = self.stage("preuve de destruction : listing par famille + delta", self.leftovers) and ok
        if ok:
            ok = self.stage("comparaison à expected.yaml, et falsification de la porte", self.compare_all)
        budget = self.tenant.get("budget_minutes")
        if ok and budget and (time.time() - self.t0) > budget * 60:
            self.stages.append({"name": "budget", "rc": 1, "seconds": 0, "note": f"{round((time.time() - self.t0) / 60, 1)} min > {budget} min"})
            ok = False
        # DEUX verdicts, et la distinction est le sujet.
        #
        # Le contrat de NON-RÉGRESSION dit si le run coïncide avec `expected.yaml`.
        # Le verdict de RELEASE dit si ce qu'on attend est publiable. Un défaut connu
        # épinglé satisfait le premier — c'est même sa raison d'être, il se reproduit à
        # l'identique — et ne dit rien du second. Les confondre revenait à conclure GO
        # parce que le produit ment de la même façon qu'hier.
        bloquants = self.defauts_bloquants()
        verdict = "GO" if ok else ("NO-GO" if self.destroyed or not self.applied else "NO-GO (RESTES À NETTOYER)")
        release = verdict
        if ok and bloquants:
            release = "NO-GO"
            self.stages.append({"name": "défauts connus bloquants", "rc": 1, "seconds": 0,
                                "note": "; ".join(f"{d['control']} [{d['source']}] {d['class'] or 'classe non dite'}"
                                                  + (f" #{d['issue']}" if d.get("issue") else "") for d in bloquants)})
        self.report(verdict, release, bloquants)
        return 0 if release == "GO" else 1

    def defauts_bloquants(self):
        """Les défauts connus qui interdisent une release, tous tenants confondus."""
        out = []
        for src in getattr(self, "comparison", {}).values():
            out += [d for d in src.get("known_defects", []) if d["bloquant"]]
        return out


class StageFailed(Exception):
    pass


# ═══════════════════════════════════════════════════════════════════════════════
# selftest — la comparaison se prouve sur des cas synthétiques
# ═══════════════════════════════════════════════════════════════════════════════

def selftest():
    outputs = {"vm": "11111111-aaaa", "bucket": "pepin-qual-x-public"}
    prefix = "pepin-qual-"
    expected = {
        "exit_codes": {"live": 1},
        "controls": {
            "a_fail": {"live": {"fail": ["${output.vm}"], "silent": ["pepin-qual-vm-ok"]}},
            "b_notev": {"live": "not-evaluated"},
            "c_pass": {"live": "pass"},
            "d_eval": {"live": {"status": "evaluated", "reason": "hors tenant"}},
            "e_na": {"live": "not-applicable"},
            "f_absent": {"live": "absent"},
            "g_defect": {"live": {"fail": ["pepin-qual-vm-defect"], "known_defect": "#0 exemple"}},
            "h_inc": {"live": {"fail": ["pepin-qual-vm-prod"], "inconclusive": ["pepin-qual-vm-noenv"]}},
            # #228 — les deux formes que la comparaison confondait.
            # `i_both` porte un statut ET des sujets : c'est la forme de
            # `kubernetes_cluster_audit_logging_enabled` chez Exoscale, la seule du
            # dépôt, et c'est elle qui a fait dire NO-GO à une correction juste.
            "i_both": {"live": {"status": "fail", "fail": ["pepin-qual-vm-audit"]}},
            # `j_fail` porte un statut `fail` SANS sujet : la forme qui n'existait pas
            # encore, et dont le piège attendait la prochaine attente écrite.
            "j_fail": {"live": "fail"},
        },
    }
    good = [
        {"control": "a_fail", "status": "fail", "subject": "11111111-aaaa"},
        {"control": "b_notev", "status": "not-evaluated", "subject": "scaleway"},
        {"control": "c_pass", "status": "pass", "subject": "scaleway"},
        {"control": "d_eval", "status": "fail", "subject": "someone@example.org"},
        {"control": "e_na", "status": "not-applicable", "subject": "scaleway"},
        {"control": "g_defect", "status": "fail", "subject": "pepin-qual-vm-defect"},
        {"control": "h_inc", "status": "fail", "subject": "pepin-qual-vm-prod"},
        {"control": "h_inc", "status": "not-evaluated", "subject": "pepin-qual-vm-noenv"},
        {"control": "i_both", "status": "fail", "subject": "pepin-qual-vm-audit"},
        {"control": "j_fail", "status": "fail", "subject": "pepin-qual-vm-any"},
    ]

    def mutate(fn):
        r = copy.deepcopy(good)
        fn(r)
        return r

    cases = [
        ("attendu exact", expected, good, 1, True),
        ("faux vert : l'écart attendu manque", expected, mutate(lambda r: r.__setitem__(0, {"control": "a_fail", "status": "pass", "subject": "scaleway"})), 1, False),
        ("faux positif : écart sur un sujet du tenant que rien n'attend", expected, good + [{"control": "c_pass", "status": "fail", "subject": "pepin-qual-vm-other"}], 1, False),
        ("le contre-exemple parle", expected, good + [{"control": "a_fail", "status": "fail", "subject": "pepin-qual-vm-ok"}], 1, False),
        ("régression de couverture : pass → not-evaluated", expected, mutate(lambda r: r.__setitem__(2, {"control": "c_pass", "status": "not-evaluated", "subject": "scaleway"})), 1, False),
        ("pass non prouvé : not-evaluated → pass", expected, mutate(lambda r: r.__setitem__(1, {"control": "b_notev", "status": "pass", "subject": "scaleway"})), 1, False),
        ("contrôle non épinglé", expected, good + [{"control": "z_new", "status": "pass", "subject": "scaleway"}], 1, False),
        ("contrôle épinglé absent de l'assessment", expected, good[1:], 1, False),
        ("code de sortie inattendu", expected, good, 0, False),
        ("écart hors tenant sur un contrôle « pass » : information, pas NO-GO", expected, good + [{"control": "c_pass", "status": "fail", "subject": "foreign-vm"}], 1, True),
        ("sujet irrésoluble", expected, good, 1, False, {"bucket": "x"}),
        ("« evaluated » refuse un not-evaluated", expected, mutate(lambda r: r.__setitem__(3, {"control": "d_eval", "status": "not-evaluated", "subject": "scaleway"})), 1, False),
        ("« absent » refuse un contrôle qui apparaît", expected, good + [{"control": "f_absent", "status": "pass", "subject": "scaleway"}], 1, False),
        ("un défaut connu corrigé rend NO-GO (la correction se dit)", expected, mutate(lambda r: r.__setitem__(5, {"control": "g_defect", "status": "pass", "subject": "scaleway"})), 1, False),
        ("un inconcluant attendu qui disparaît rend NO-GO", expected, mutate(lambda r: r.__setitem__(7, {"control": "h_inc", "status": "fail", "subject": "pepin-qual-vm-noenv"})), 1, False),
        ("un inconcluant que rien n'attend rend NO-GO", expected, good + [{"control": "a_fail", "status": "not-evaluated", "subject": "pepin-qual-vm-x"}], 1, False),
        # ── #228 : l'écart ATTENDU ne doit pas être pris pour une régression ──────
        #
        # Le premier cas est celui qui a rougi à tort sur un run réel : l'attente porte
        # `status: fail` ET `fail: [...]`, l'assessment rend très exactement cet écart,
        # et la comparaison le déclarait « faux positif ou attente périmée ». Une porte
        # qui crie au loup sur une correction juste s'apprend à être discutée, ce qui
        # coûte aussi cher que la porte qui dit GO sur un faux vert.
        ("un fail attendu, sujets nommés : l'écart CONFIRME l'attente", expected, good, 1, True),
        ("un fail attendu, sujets nommés : c'est son ABSENCE qui rend NO-GO", expected,
         mutate(lambda r: r.__setitem__(8, {"control": "i_both", "status": "pass", "subject": "scaleway"})), 1, False),
        ("un fail attendu SANS sujet : l'écart confirme", expected, good, 1, True),
        ("un fail attendu SANS sujet : aucun écart rend NO-GO", expected,
         mutate(lambda r: r.__setitem__(9, {"control": "j_fail", "status": "pass", "subject": "scaleway"})), 1, False),
        # Le contre-exemple : la forme 2 l'emporte, donc un sujet du tenant que rien ne
        # nomme reste un faux positif — le gain de précision ne doit pas se payer d'un
        # relâchement.
        ("la forme qui nomme garde sa rigueur sur les sujets", expected,
         good + [{"control": "i_both", "status": "fail", "subject": "pepin-qual-vm-autre"}], 1, False),
    ]
    failures = 0
    # ── Ce qu'un défaut connu vaut pour une RELEASE ────────────────────────────
    #
    # Le contrat de non-régression et le verdict de release répondent à deux
    # questions. Ces cas éprouvent la seconde, qui n'existait pas : un défaut épinglé
    # se reproduisait, la comparaison ne trouvait aucune différence, et la porte
    # concluait GO — elle vérifiait que le produit ment de la même façon qu'hier.
    for label, kd, bloquant_attendu in [
        ("une chaîne ne dit pas sa classe : traitée comme BLOQUANTE",
         "#0 exemple", True),
        ("un faux vert bloque",
         {"issue": 209, "class": "false_green"}, True),
        ("un faux vert ne se déroge PAS, même demandé",
         {"issue": 209, "class": "false_green", "release_blocker": False}, True),
        ("un faux positif bloque par défaut",
         {"issue": 208, "class": "false_positive"}, True),
        ("un faux positif peut se déroger, explicitement",
         {"issue": 208, "class": "false_positive", "release_blocker": False}, False),
        ("une lacune de couverture ne bloque pas",
         {"issue": 210, "class": "coverage_gap"}, False),
        ("une classe inconnue est traitée comme BLOQUANTE",
         {"issue": 1, "class": "inventee"}, True),
    ]:
        d = defaut_connu({"known_defect": kd})
        ok = d is not None and d["bloquant"] == bloquant_attendu
        failures += not ok
        print(f"  {'✔' if ok else '✘'} {label}" + ("" if ok else f" — obtenu bloquant={d and d['bloquant']}"))
    for case in cases:
        label, exp, results, rc, want_ok = case[:5]
        outs = case[5] if len(case) > 5 else outputs
        v = compare(exp, results, "live", rc, outs, prefix)
        ok = v.ok == want_ok
        failures += not ok
        print(f"  {'✔' if ok else '✘'} {label}" + ("" if ok else f" — obtenu {'GO' if v.ok else 'NO-GO'} : {v.problems[:2]}"))
    # Et la falsification elle-même : sur un assessment conforme, chaque mutation doit rougir.
    ok, outcomes = falsify(expected, good, "live", 1, outputs, prefix)
    failures += not ok
    print(f"  {'✔' if ok else '✘'} la falsification rend NO-GO sur {len(outcomes)} mutations")
    if failures:
        print(f"\n{failures} cas en échec : la comparaison ne mesure pas ce qu'elle prétend", file=sys.stderr)
        return 1
    print(f"\n{len(cases) + 8} cas, tous conformes.")
    return 0


# ═══════════════════════════════════════════════════════════════════════════════

def cmd_compare(args):
    run_dir = pathlib.Path(args.run_dir).resolve()
    provider = args.provider or run_dir.name.replace("qualification-", "")
    tenant_dir = ROOT / "references" / "qualification" / provider
    tenant = yaml.safe_load((tenant_dir / "tenant.yaml").read_text())
    expected = yaml.safe_load(pathlib.Path(args.expected or tenant_dir / "expected.yaml").read_text())
    outputs, plan = {}, None
    if (run_dir / "plan.json").exists():
        plan = json.loads((run_dir / "plan.json").read_text())
        outputs = {k: o["value"] for k, o in (plan.get("planned_values", {}).get("outputs") or {}).items() if "value" in o}
    if (run_dir / "outputs.json").exists():
        outputs = json.loads((run_dir / "outputs.json").read_text())
    owned = owned_identifiers(outputs, plan)
    report = json.loads((run_dir.parent / f"qualification-{provider}.json").read_text()) if (run_dir.parent / f"qualification-{provider}.json").exists() else {}
    rcs = {s: v.get("exit_code") for s, v in report.get("comparison", {}).items()}
    problems = 0
    for source, fname in (("live", "assessment-live.json"), ("terraform", "assessment-terraform.json")):
        if not (run_dir / fname).exists():
            continue
        v = compare(expected, load_results(run_dir / fname), source, rcs.get(source), outputs, tenant["name_prefix"], owned)
        print(f"[{source}] {'GO' if v.ok else 'NO-GO'}")
        for p in v.problems:
            print(f"  ✘ {p}")
        for i in v.infos:
            print(f"  · {i}")
        problems += len(v.problems)
    print("\nverdict :", "GO" if not problems else f"NO-GO ({problems} différence(s))")
    return 0 if not problems else 1


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)
    r = sub.add_parser("run", help="applique, scanne, scelle, détruit, compare")
    r.add_argument("provider")
    r.add_argument("--plan-only", action="store_true", help="plan + scan --terraform + comparaison de la source terraform ; rien n'est appliqué")
    r.add_argument("--run-dir", help="dossier des artefacts (défaut : release-gate/qualification-<provider>)")
    c = sub.add_parser("compare", help="recompare un run existant à expected.yaml (ou à un autre fichier)")
    c.add_argument("--run-dir", required=True)
    c.add_argument("--expected")
    c.add_argument("--provider")
    sub.add_parser("selftest", help="la comparaison se prouve sur des cas synthétiques")
    args = ap.parse_args()

    if args.cmd == "selftest":
        return selftest()
    if args.cmd == "compare":
        return cmd_compare(args)
    if not args.plan_only and os.environ.get("PEPIN_GATE_LIVE") != "1":
        print("refus : appliquer un tenant de qualification sur un compte réel est opt-in — "
              "PEPIN_GATE_LIVE=1, ou --plan-only pour ne rien créer", file=sys.stderr)
        return 1
    tenant_dir = ROOT / "references" / "qualification" / args.provider
    if not (tenant_dir / "expected.yaml").exists():
        print(f"aucun tenant de qualification pour « {args.provider} » ({tenant_dir})", file=sys.stderr)
        return 2
    run_dir = pathlib.Path(args.run_dir).resolve() if args.run_dir else ROOT / "release-gate" / f"qualification-{args.provider}"
    return Run(args.provider, run_dir, args.plan_only).go()


if __name__ == "__main__":
    sys.exit(main())
