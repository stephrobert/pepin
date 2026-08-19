#!/usr/bin/env python3
"""Dérive de l'index SCSL depuis la baseline versionnée.

Le référentiel de Pépin mappe chaque contrôle sur une exigence CLD-* de l'index
gelé du framework SCSL (framework-scsl/api/v1/exigences.json). Cet index vit
HORS du dépôt et bouge sous lui : une exigence retirée, reformulée ou ajoutée
sans que personne ne retrie les mappings est le même défaut qu'une opération
amont non triée : le rapport cite alors une référence normative qui n'existe
plus, ou plus sous ce texte.

La baseline (referentiel/scsl-baseline.json) est versionnée dans le dépôt : elle
enregistre, pour chaque exigence CLD, l'empreinte de son contenu normatif
(texte, devoir, niveau, essentielle, famille) et les codes Pépin que son
outillage cite. Ce script compare l'index vivant à la baseline et NOMME chaque
écart. Le triage est humain : relire l'exigence, ajuster controles.yaml ou le
catalogue, puis `mise run scsl-drift-update`.

Codes de sortie (convention des outils de release, distincte de celle de
`pepin scan` : ici 2 signifie dérive, pas erreur technique) :
    0  l'index vivant est celui que la baseline décrit
    1  erreur (index ou baseline illisible)
    2  dérive détectée : exigence ajoutée, retirée ou modifiée depuis la baseline

Usage :
    python3 scripts/scsl-drift.py [--index CHEMIN] [--update]
"""
from __future__ import annotations

import argparse
import hashlib
import json
import pathlib
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
BASELINE = ROOT / "referentiel" / "scsl-baseline.json"
BASELINE_FORMAT = "pepin-scsl-baseline/v1"
DEFAULT_INDEX = ROOT.parent / "framework-scsl" / "api" / "v1" / "exigences.json"


def exigences_cld(index_path: pathlib.Path) -> dict[str, dict]:
    """Lit l'index et retourne {id CLD-* -> {empreinte, pepin}}.

    Reflète referentiel.ParseCLDExigences (Go) : domaine CLD, famille non vide,
    id coupé à partir de « CLD- » quel que soit le préfixe amont (SCSL-, SOCLE-).
    """
    try:
        items = json.loads(index_path.read_text(encoding="utf-8"))
    except OSError as e:
        sys.exit(f"erreur : index SCSL illisible ({e}) ; cloner framework-scsl ou passer --index")
    except json.JSONDecodeError as e:
        sys.exit(f"erreur : index SCSL invalide ({e})")
    out: dict[str, dict] = {}
    for e in items:
        if e.get("domaine") != "CLD" or not e.get("famille"):
            continue
        ident = e.get("id", "")
        cut = ident.find("CLD-")
        if cut >= 0:
            ident = ident[cut:]
        # Le contenu normatif seulement : ce que Pépin cite dans ses rapports.
        # Les champs d'illustration (exemples, menaces) peuvent bouger sans que
        # la référence change de sens ; les geler ferait crier la garde pour rien.
        normatif = {
            "texte": e.get("texte", ""),
            "devoir": e.get("devoir", ""),
            "niveau": e.get("niveau", ""),
            "essentielle": bool(e.get("essentielle", False)),
            "famille": e.get("famille", ""),
        }
        canon = json.dumps(normatif, sort_keys=True, ensure_ascii=False)
        # L'outillage cite Pépin soit nu (« Pépin »), soit avec un code
        # (« Pépin: <code> ») : on retient les deux formes, car perdre la
        # citation amont est un signal de retriage au même titre qu'un code.
        pepin = sorted(
            (o.split("Pépin:", 1)[1].strip() if "Pépin:" in o else "Pépin")
            for o in e.get("outillage", [])
            if "Pépin" in o
        )
        out[ident] = {
            "empreinte": "sha256:" + hashlib.sha256(canon.encode("utf-8")).hexdigest(),
            "pepin": pepin,
        }
    if not out:
        sys.exit("erreur : aucune exigence CLD dans l'index (format inattendu)")
    return out


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--index", type=pathlib.Path, default=DEFAULT_INDEX,
                    help="chemin de exigences.json (défaut : ../framework-scsl/api/v1/exigences.json)")
    ap.add_argument("--update", action="store_true",
                    help="réécrire la baseline depuis l'index vivant (après triage humain)")
    args = ap.parse_args()

    live = exigences_cld(args.index)

    if args.update:
        doc = {"format": BASELINE_FORMAT, "exigences": dict(sorted(live.items()))}
        BASELINE.write_text(json.dumps(doc, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
        print(f"baseline réécrite : {BASELINE.relative_to(ROOT)} ({len(live)} exigences CLD)")
        return 0

    try:
        doc = json.loads(BASELINE.read_text(encoding="utf-8"))
    except OSError:
        sys.exit(f"erreur : baseline absente ({BASELINE.relative_to(ROOT)}) ; la créer : mise run scsl-drift-update")
    except json.JSONDecodeError as e:
        sys.exit(f"erreur : baseline invalide ({e})")
    if doc.get("format") != BASELINE_FORMAT:
        sys.exit(f"erreur : format de baseline inconnu {doc.get('format')!r} (attendu {BASELINE_FORMAT})")
    base = doc.get("exigences", {})

    drifts: list[str] = []
    for ident in sorted(set(base) - set(live)):
        drifts.append(f"retirée   : {ident} : les contrôles qui la mappent citent une exigence disparue")
    for ident in sorted(set(live) - set(base)):
        drifts.append(f"ajoutée   : {ident} : à trier (mapper un contrôle, ou laisser au catalogue en le disant)")
    for ident in sorted(set(base) & set(live)):
        if base[ident].get("empreinte") != live[ident]["empreinte"]:
            drifts.append(f"modifiée  : {ident} : le contenu normatif a changé, relire le mapping")
        elif base[ident].get("pepin") != live[ident]["pepin"]:
            drifts.append(f"outillage : {ident} : les codes Pépin cités amont ont changé "
                          f"({base[ident].get('pepin')} -> {live[ident]['pepin']})")

    if drifts:
        print(f"dérive SCSL : {len(drifts)} écart(s) depuis la baseline", file=sys.stderr)
        for d in drifts:
            print(f"  {d}", file=sys.stderr)
        print("après triage : mise run scsl-drift-update, puis committer la baseline", file=sys.stderr)
        return 2

    print(f"index SCSL conforme à la baseline ({len(live)} exigences CLD)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
