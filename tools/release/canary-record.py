#!/usr/bin/env python3
"""Écrit le relevé de canari d'un fournisseur depuis la sortie `--format json`.

    canary-record.py <fournisseur> <scan.json> <version> <sortie.yaml>

Le relevé ne porte que des FAITS d'endpoint : hôte, méthode, chemin, statut HTTP,
et la classe que Pépin a attribuée au refus. Ni identifiant, ni nom de ressource,
ni rien d'un tenant — le canari n'en produit aucun, puisqu'il n'est jamais
authentifié, et le relevé est écrit de façon à ne pas pouvoir en porter.

Trois verdicts, et c'est tout le vocabulaire :

    answered     l'endpoint a répondu un statut HTTP. Il existe, il se résout,
                 il parle. C'est ce que le canari vient chercher.
    moved        404 : le chemin déclaré n'est plus là. C'est la régression
                 qu'un descripteur ne peut pas voir venir tout seul.
    unreachable  aucune réponse HTTP (DNS, TCP, TLS, proxy). Non concluant :
                 cela parle du réseau du mainteneur, pas du fournisseur.
"""
import datetime
import json
import re
import sys

# `HTTP 401 · GET https://host/chemin · {corps}` — la forme que
# internal/collect compose pour le détail d'une unité incomplète.
STATUS_RE = re.compile(r"HTTP (\d{3})")
URL_RE = re.compile(r"\b(GET|POST|PUT|HEAD|DELETE)\s+(https?://[^\s·]+)")


def verdict_of(status):
    if status is None:
        return "unreachable"
    if status == 404:
        return "moved"
    return "answered"


def unit_row(u):
    detail = u.get("detail") or ""
    m = STATUS_RE.search(detail)
    status = int(m.group(1)) if m else None
    m = URL_RE.search(detail)
    method, url = (m.group(1), m.group(2)) if m else ("", "")
    # L'URL est scindée : l'hôte est un fait du descripteur, le chemin aussi.
    # Aucune query string n'est conservée — c'est le seul endroit où une valeur
    # de tenant pourrait se glisser.
    host, path = "", ""
    if url:
        rest = url.split("://", 1)[1]
        host, _, path = rest.partition("/")
        path = "/" + path.split("?", 1)[0]
    return {
        "unit": u.get("unit", ""),
        "attempted": bool(u.get("attempted")),
        "complete": bool(u.get("complete")),
        "method": method,
        "host": host,
        "path": path,
        "status": status,
        "classified": u.get("error", ""),
        "verdict": verdict_of(status),
    }


def yaml_escape(s):
    return '"' + str(s).replace("\\", "\\\\").replace('"', '\\"') + '"'


def render(provider, version, rows):
    counts = {"answered": 0, "moved": 0, "unreachable": 0}
    for r in rows:
        counts[r["verdict"]] += 1
    today = datetime.date.today().isoformat()
    out = []
    out.append("# Relevé de canari — GÉNÉRÉ par tools/release/canary.sh.")
    out.append("#")
    out.append("# Une requête NON AUTHENTIFIÉE par endpoint déclaré, contre le vrai plan de")
    out.append("# contrôle du fournisseur. Ce qui se mesure est le REFUS : un endpoint qui")
    out.append("# répond existe et se résout ; un 404 dit qu'il a bougé.")
    out.append("#")
    out.append("# Ce relevé ne porte aucun identifiant : le canari n'en a jamais eu.")
    out.append(f"provider: {provider}")
    out.append(f"recorded: {today}")
    out.append(f"pepin_version: {yaml_escape(version)}")
    out.append("authenticated: false")
    out.append("summary:")
    for k in ("answered", "moved", "unreachable"):
        out.append(f"  {k}: {counts[k]}")
    out.append("units:")
    for r in sorted(rows, key=lambda x: x["unit"]):
        out.append(f"  - unit: {r['unit']}")
        out.append(f"    verdict: {r['verdict']}")
        if r["status"] is not None:
            out.append(f"    status: {r['status']}")
        if r["method"]:
            out.append(f"    method: {r['method']}")
        if r["host"]:
            out.append(f"    host: {r['host']}")
        if r["path"]:
            out.append(f"    path: {yaml_escape(r['path'])}")
        out.append(f"    classified: {r['classified'] or '\"\"'}")
    return "\n".join(out) + "\n"


def main(argv):
    if len(argv) != 5:
        print(__doc__, file=sys.stderr)
        return 2
    provider, scan, version, dest = argv[1:]
    try:
        doc = json.load(open(scan))
    except Exception as e:                                  # noqa: BLE001
        print(f"sortie de scan illisible : {e}", file=sys.stderr)
        return 1
    units = (doc.get("collection") or {}).get("units") or []
    if not units:
        print("aucune unité de collecte : le canari ne mesure rien", file=sys.stderr)
        return 1
    with open(dest, "w") as f:
        f.write(render(provider, version, [unit_row(u) for u in units]))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
