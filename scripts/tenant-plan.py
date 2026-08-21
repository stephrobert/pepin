#!/usr/bin/env python3
"""Outils du tenant de référence : entrées bouchons, puis réduction du plan.

Deux sous-commandes, appelées par scripts/reference-tenant.sh :

    placeholders <dir>   écrit de quoi qu'un `terraform plan` d'une stack tierce
                         se résolve, SANS toucher une ligne de sa configuration
    reduce <in> <out>    ne garde du plan que ce que Pépin lit, et met à null
                         toute valeur que Terraform marque `sensitive`

# Pourquoi des bouchons plutôt qu'un fichier de variables écrit à la main

Une stack tierce déclare des variables sans défaut (identifiants, chemins de clé,
mots de passe). Leur donner une valeur CHOISIE reviendrait à réécrire la
configuration, donc à retomber dans la fixture auto-confirmante. Les valeurs
posées ici sont donc NEUTRES : elles ne décident d'aucune posture (pas de CIDR,
pas de booléen de sécurité, aucun réglage de chiffrement), elles ont seulement la
FORME que les providers valident. Aucune n'est un secret : ce sont celles que
l'émulateur local publie pour n'importe qui.

# Pourquoi la réduction

Pépin ne lit d'un plan que `planned_values` (ou `values`) et les `source` des
appels de modules — cf. internal/tfparse.ParsePlan. Tout le reste est ignoré par
le produit, et c'est précisément là qu'un plan pris sur un tenant réel porterait
ses identifiants. Committer un plan brut serait l'erreur que l'audit de livraison
a déjà vue passer une fois, sur un UUID d'instance oublié dans une fixture.

# La réduction est une LISTE BLANCHE, et elle descend jusqu'à l'ATTRIBUT

Découper au niveau des sections ne suffit pas. Une ressource tierce embarque des
attributs que Pépin ne projette jamais — le blob de valeurs Helm d'un
`helm_release`, le `yaml_body` d'un `kubectl_manifest`, un `kubernetes_secret`
entier — et les garder revient à republier la configuration applicative d'un
tiers pour rien. Un mot de passe n'y est pas une donnée de posture : AUCUNE règle
commune ne lit un champ de credential.

Le filtre est donc une liste blanche, et elle est DÉRIVÉE des descripteurs
(`providers/*.yaml`), jamais écrite à la main : un attribut n'est gardé que si un
`mapping_terraform` le lit réellement (`map`, `region`, `items`, plus le `name`
que internal/tfmap.Apply lit sur chaque ressource). Le sens de la question compte —
« est-ce que Pépin lit ce champ ? » a une réponse vérifiable dans le dépôt, là où
« est-ce que ce champ est un secret ? » n'en a jamais eu. Un tf_type qu'aucun
descripteur ne mappe ne garde AUCUNE valeur : internal/tfmap.Apply l'ignore, seul
son type est compté.

Corollaire assumé : `user_data` EST gardé, parce qu'une règle commune y cherche
des secrets. C'est le seul champ de texte libre de la liste blanche, et il se
relit à la main avant de committer un tenant.
"""
import json
import pathlib
import re
import sys

import yaml

VAR_RE = re.compile(r'variable\s+"([^"]+)"\s*\{', re.M)

DUMMY_KEY = "zz-pepin-dummy.pub"
PLACEHOLDER_VARS = "zz-pepin-placeholders.auto.tfvars.json"
BACKEND_OVERRIDE = "zz_pepin_backend_override.tf"


def block_of(text, start):
    depth, i = 0, start
    while i < len(text):
        if text[i] == "{":
            depth += 1
        elif text[i] == "}":
            depth -= 1
            if depth == 0:
                return text[start:i + 1]
        i += 1
    return text[start:]


def placeholder(body, name):
    """Valeur neutre pour une variable sans défaut, ou None si elle en a un."""
    if re.search(r"^\s*default\s*=", body, re.M):
        return None
    m = re.search(r"^\s*type\s*=\s*(.+)$", body, re.M)
    t = (m.group(1).strip() if m else "string")
    if t.startswith(("list", "set", "tuple")):
        return []
    if t.startswith(("map", "object")):
        return {}
    if t == "bool":
        return False
    if t == "number":
        return 1
    n = name.lower()
    if "secret_key" in n or "project_id" in n or "organization_id" in n or n.endswith("_uuid"):
        return "11111111-1111-1111-1111-111111111111"
    if "access_key" in n:
        return "SCWXXXXXXXXXXXXXXXXX"
    if "password" in n or "passwd" in n:
        return "Pl4ceholder-Passw0rd"
    if "api_secret" in n or "token" in n:
        return "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    if "api_key" in n:
        return "EXOxxxxxxxxxxxxxxxxxxxx"
    if n.endswith("zone"):
        return "fr-par-1"
    if n.endswith("region"):
        return "fr-par"
    if n.endswith("_path") or n.endswith("_file"):
        return "./" + DUMMY_KEY
    return "placeholder-" + name


def cmd_placeholders(d):
    dd = pathlib.Path(d)
    out = {}
    for f in sorted(dd.glob("*.tf")):
        text = f.read_text(errors="replace")
        for m in VAR_RE.finditer(text):
            v = placeholder(block_of(text, m.end() - 1), m.group(1))
            if v is not None:
                out[m.group(1)] = v
    # Une clé publique factice, pour les `file(var.…_path)` d'une stack tierce.
    (dd / DUMMY_KEY).write_text(
        "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
        " pepin@placeholder\n")
    # Un backend LOCAL : le plan ne doit joindre aucun stockage d'état distant.
    # Les fichiers `*_override.tf` sont le mécanisme prévu par Terraform ; la
    # configuration amont n'est pas modifiée.
    (dd / BACKEND_OVERRIDE).write_text('terraform {\n  backend "local" {}\n}\n')
    if out:
        (dd / PLACEHOLDER_VARS).write_text(json.dumps(out, indent=2, sort_keys=True) + "\n")
    print(f"{len(out)} variable(s) bouchonnée(s)")


# Les noms d'attributs qui portent un secret par vocation. Ils ne servent qu'en
# SECOND rideau, sous la liste blanche : un champ qui passe la liste blanche est
# déjà un champ qu'une règle lit, et aucun de ceux-là n'en est un. Le filet couvre
# le cas d'un attribut mappé qui NICHE une valeur de credential.
SECRET_KEY_RE = re.compile(
    r"pass(word|wd)?$|secret|token|credential|private_key|api_key|access_key", re.I)


def root_field(path):
    """Premier segment d'un chemin de projection : `audit.0.endpoint` -> `audit`."""
    path = path.strip()
    if path.startswith("_parent."):
        path = path[len("_parent."):]
    return path.split(".", 1)[0].split("[", 1)[0]


def mapped_fields(root):
    """Champs Terraform réellement lus, par tf_type, DÉRIVÉS des descripteurs.

    Ce que internal/tfmap.Apply lit d'une ressource du plan, et rien d'autre :
    les racines des chemins de `map` (`||` = repli, `_parent.` = la ressource
    porteuse), le champ `region`, le bloc répété de `items`, et le `name` qu'Apply
    lit sur chaque ressource pour nommer la ressource normalisée.
    """
    allow = {}
    for f in sorted((pathlib.Path(root) / "providers").glob("*.yaml")):
        desc = yaml.safe_load(f.read_text()) or {}
        for r in (desc.get("mapping_terraform") or {}).get("resources") or []:
            tf = r.get("tf_type")
            if not tf:
                continue
            keep = allow.setdefault(tf, {"name"})
            if r.get("region"):
                keep.add(r["region"])
            if r.get("items"):
                keep.add(root_field(r["items"]))
            for path in (r.get("map") or {}).values():
                for alt in str(path).split("||"):
                    if field := root_field(alt):
                        keep.add(field)
    return allow


def scrub(v):
    """Annule, en profondeur, les clés dont le NOM dit qu'elles portent un secret."""
    if isinstance(v, dict):
        return {k: (None if SECRET_KEY_RE.search(k) else scrub(x)) for k, x in v.items()}
    if isinstance(v, list):
        return [scrub(x) for x in v]
    return v


def reduce_module(m, allow):
    out = {}
    res = []
    for r in m.get("resources") or []:
        rr = {k: r[k] for k in ("address", "type", "name") if k in r}
        vals = r.get("values") or {}
        sens = r.get("sensitive_values") or {}
        # La LISTE BLANCHE : un tf_type qu'aucun descripteur ne mappe ne garde
        # aucune valeur, et un attribut qu'aucun `map` ne lit non plus.
        keep = allow.get(r.get("type")) or set()
        kept = {}
        for k in sorted(vals):
            if k not in keep:
                continue
            kept[k] = None if (sens.get(k) is True or SECRET_KEY_RE.search(k)) else scrub(vals[k])
        rr["values"] = kept
        res.append(rr)
    if res:
        out["resources"] = res
    ch = [c for c in (reduce_module(x, allow) for x in (m.get("child_modules") or [])) if c]
    if ch:
        out["child_modules"] = ch
    if "address" in m:
        out["address"] = m["address"]
    return out


def reduce_config(m):
    calls = {}
    for name, call in (m.get("module_calls") or {}).items():
        c = {}
        if "source" in call:
            c["source"] = call["source"]
        sub = reduce_config(call.get("module") or {})
        if sub:
            c["module"] = sub
        calls[name] = c
    return {"module_calls": calls} if calls else {}


def cmd_reduce(src, dst):
    doc = json.loads(pathlib.Path(src).read_text())
    allow = mapped_fields(pathlib.Path(__file__).resolve().parent.parent)
    if not allow:
        print("aucun mapping_terraform lu : la liste blanche ne mesure rien", file=sys.stderr)
        return 1
    out = {k: doc[k] for k in ("format_version", "terraform_version") if k in doc}
    pv = doc.get("planned_values") or doc.get("values")
    if pv:
        out["planned_values"] = {"root_module": reduce_module(pv.get("root_module") or {}, allow)}
    cfg = reduce_config((doc.get("configuration") or {}).get("root_module") or {})
    if cfg:
        out["configuration"] = {"root_module": cfg}
    pathlib.Path(dst).write_text(json.dumps(out, separators=(",", ":"), sort_keys=True) + "\n")
    print(f"{pathlib.Path(dst).stat().st_size} octets")


def main(argv):
    if len(argv) >= 3 and argv[1] == "placeholders":
        return cmd_placeholders(argv[2])
    if len(argv) >= 4 and argv[1] == "reduce":
        return cmd_reduce(argv[2], argv[3])
    print(__doc__, file=sys.stderr)
    return 2


if __name__ == "__main__":
    sys.exit(main(sys.argv) or 0)
