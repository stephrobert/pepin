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
"""
import json
import pathlib
import re
import sys

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


def reduce_module(m):
    out = {}
    res = []
    for r in m.get("resources") or []:
        rr = {k: r[k] for k in ("address", "type", "name") if k in r}
        vals = dict(r.get("values") or {})
        sens = r.get("sensitive_values") or {}
        for k in list(vals):
            if sens.get(k) is True:
                vals[k] = None
        rr["values"] = vals
        res.append(rr)
    if res:
        out["resources"] = res
    ch = [c for c in (reduce_module(x) for x in (m.get("child_modules") or [])) if c]
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
    out = {k: doc[k] for k in ("format_version", "terraform_version") if k in doc}
    pv = doc.get("planned_values") or doc.get("values")
    if pv:
        out["planned_values"] = {"root_module": reduce_module(pv.get("root_module") or {})}
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
