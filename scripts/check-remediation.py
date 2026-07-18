#!/usr/bin/env python3
"""Suivi de complétude des preuves de remédiation (cf. references/remediation/).

Pour chaque contrôle actif (referentiel/controles.yaml) et chaque provider qui
l'implémente (`fournisseurs`), vérifie qu'il existe une preuve de remédiation
references/remediation/<provider>/<code>.tf (ou .md). Rapporte les manquantes par
provider. Pendant de `make validate` côté documentation.

Usage :
    python3 scripts/check-remediation.py [provider ...]   # tous, ou ceux listés
Sortie : exit 0 si tout est couvert, 1 sinon (utilisable en garde CI).
"""
from __future__ import annotations

import pathlib
import sys

import yaml

ROOT = pathlib.Path(__file__).resolve().parent.parent
CONTROLES = ROOT / "referentiel" / "controles.yaml"
REMEDIATION = ROOT / "references" / "remediation"


def has_proof(provider: str, code: str) -> bool:
    base = REMEDIATION / provider / code
    if base.with_suffix(".md").exists():  # note documentaire
        return True
    return base.is_dir() and any(base.glob("*.tf"))  # module Terraform autonome


def main(argv: list[str]) -> int:
    doc = yaml.safe_load(CONTROLES.read_text(encoding="utf-8")) or {}
    controles = doc.get("controles", []) if isinstance(doc, dict) else doc
    covered: dict[str, list[str]] = {}
    missing: dict[str, list[str]] = {}
    for c in controles:
        if not isinstance(c, dict):
            continue
        code = c.get("code")
        for provider in c.get("fournisseurs") or []:
            if argv and provider not in argv:
                continue
            bucket = covered if has_proof(provider, code) else missing
            bucket.setdefault(provider, []).append(code)
    total_missing = 0
    for provider in sorted(set(covered) | set(missing)):
        cov, miss = covered.get(provider, []), missing.get(provider, [])
        print(f"\n# {provider} : {len(cov)}/{len(cov) + len(miss)} remédiations")
        for code in sorted(miss):
            print(f"  ✗ manquante : {code}")
        total_missing += len(miss)
    print(f"\n{total_missing} preuve(s) de remédiation manquante(s).")
    return 0 if total_missing == 0 else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
