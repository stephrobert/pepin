#!/usr/bin/env python3
"""Détecte la dérive de FORME entre les ADR et le dépôt.

Ce contrôle est délibérément partiel. Il attrape ce qu'une machine peut voir —
un chemin disparu, une garde qui n'existe plus, un statut incohérent — et rien
de ce qui demande de lire. Un rapport vide ne prouve pas qu'un ADR dit encore la
vérité : il prouve seulement qu'il ne cite rien de manifestement mort.

Code de sortie : 0 rien à signaler, 1 dérive détectée, 2 erreur technique.
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
ADR = ROOT / "docs" / "adr"

# Un chemin cité : `internal/assess/assess.go`, `cmd/scan.go`, `docs/coverage.md`.
PATH = re.compile(r"`((?:internal|cmd|providers|referentiel|docs|tools|scripts)/[\w./-]+)`")
# Une garde citée : `TestQuelqueChose` ou `test_quelque_chose` (Rego).
GUARD = re.compile(r"`(Test[A-Z]\w+|test_[a-z0-9_]+)`")
STATUS = re.compile(r"^status:\s*(.+)$", re.M)
SUPERSEDES = re.compile(r"ADR-(\d{4})\s+supersedes\s+ADR-(\d{4})", re.I)


def go_symbols() -> set[str]:
    """Noms de tests Go et Rego présents dans le dépôt."""
    out: set[str] = set()
    for p in ROOT.rglob("*_test.go"):
        if ".git" in p.parts:
            continue
        out.update(re.findall(r"^func (Test\w+)", p.read_text(encoding="utf-8", errors="replace"), re.M))
    for p in ROOT.rglob("*_test.rego"):
        if ".git" in p.parts:
            continue
        out.update(re.findall(r"^(test_[a-z0-9_]+)", p.read_text(encoding="utf-8", errors="replace"), re.M))
    return out


def main() -> int:
    if not ADR.is_dir():
        print(f"pas de registre ADR en {ADR}", file=sys.stderr)
        return 2

    adrs = sorted(p for p in ADR.glob("0*.md"))
    if not adrs:
        print("registre vide : rien à contrôler", file=sys.stderr)
        return 2

    known = go_symbols()
    findings: list[str] = []
    superseded_by: dict[str, str] = {}

    for p in adrs:
        text = p.read_text(encoding="utf-8")
        num = p.name[:4]

        st = STATUS.search(text)
        status = st.group(1).strip() if st else ""
        if not status:
            findings.append(f"{p.name} : aucun `status:`")

        for m in SUPERSEDES.finditer(text):
            superseded_by[f"{int(m.group(2)):04d}"] = f"{int(m.group(1)):04d}"

        for cited in set(PATH.findall(text)):
            # Un chemin peut désigner un répertoire ou un fichier.
            if not (ROOT / cited).exists():
                findings.append(f"{p.name} : chemin cité inexistant — {cited}")

        guards = set(GUARD.findall(text))
        for g in guards:
            if g not in known:
                findings.append(f"{p.name} : garde citée introuvable — {g}")

        if status.startswith("Accepted") and "## Invariants" in text and not guards:
            if "automatisable" not in text and "automatable" not in text:
                findings.append(
                    f"{p.name} : Accepted, des invariants, aucune garde citée "
                    "et aucune raison écrite de ne pas en avoir"
                )

    for old, new in superseded_by.items():
        match = [p for p in adrs if p.name.startswith(old)]
        if not match:
            findings.append(f"ADR-{new} remplace ADR-{old}, qui n'existe pas")
            continue
        text = match[0].read_text(encoding="utf-8")
        if f"Superseded by ADR-{new}" not in text:
            findings.append(
                f"{match[0].name} : remplacé par ADR-{new} mais son statut ne le dit pas"
            )

    print(f"adr-drift : {len(adrs)} ADR contrôlés")
    if not findings:
        print("  rien à signaler — de forme ; la substance se relit")
        return 0
    for f in findings:
        print(f"  ✘ {f}")
    print(f"\n{len(findings)} dérive(s). Ce contrôle ne voit que la forme.")
    return 1


if __name__ == "__main__":
    sys.exit(main())
