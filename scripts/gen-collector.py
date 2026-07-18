#!/usr/bin/env python3
"""Génère le SQUELETTE de collecteur Go d'un provider à partir de son contrat
d'API (referentiel/contrats/<provider>.yaml).

Pour chaque type `etat: verifie`, émet une fonction de projection
`map<Type>(...) model.Resource` avec les clés d'attributs communs pré-remplies et
le champ natif (du contrat) en commentaire. Les appels API/auth/pagination et les
transforms non triviaux restent des TODO (signalés). Les types `a_verifier` sont
listés en TODO (contrat à compléter avant de coder).

Sortie : providers/<provider>/collector_scaffold.go.txt (NON compilé : à compléter
puis renommer en .go). Usage : python3 scripts/gen-collector.py <provider>
"""
import sys, os, yaml

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def gofunc(typ):  # security_group_rule -> SecurityGroupRule
    return "".join(p.capitalize() for p in typ.split("_"))


def main():
    if len(sys.argv) != 2:
        sys.exit("usage: gen-collector.py <provider>")
    prov = sys.argv[1]
    contrat = os.path.join(REPO, "referentiel", "contrats", f"{prov}.yaml")
    with open(contrat) as f:
        c = yaml.safe_load(f)
    types = c.get("types", {})
    verifie = {t: d for t, d in types.items() if d.get("etat") == "verifie"}
    a_verifier = [t for t, d in types.items() if d.get("etat") != "verifie"]

    out = [f"// SQUELETTE généré par scripts/gen-collector.py depuis",
           f"// referentiel/contrats/{prov}.yaml — À COMPLÉTER puis renommer en .go.",
           f"// Règle d'archi : ce collecteur NORMALISE vers le modèle commun ; aucune règle ici.",
           f"package {prov}", "",
           'import "github.com/stephrobert/pepin/internal/model"', ""]
    for typ, d in verifie.items():
        src = d.get("source", "?")
        out.append(f"// map{gofunc(typ)} projette {src} → type normalisé « {typ} ».")
        out.append(f"// TODO: signature réelle (struct SDK) + appel List dans collectLive.")
        out.append(f"func map{gofunc(typ)}(src any) model.Resource {{")
        out.append(f'\treturn model.Resource{{Provider: "{prov}", Type: "{typ}",')
        out.append("\t\tAttributes: map[string]any{")
        for attr, native in (d.get("mapping") or {}).items():
            out.append(f'\t\t\t"{attr}": nil, // {native}')
        out.append("\t\t},")
        out.append("\t}")
        out.append("}")
        out.append("")
    out.append("// collectLive : TODO — auth (contrat: sdk), puis pour chaque type vérifié :")
    out.append("//   lister via l'API (pagination), appeler map<Type> sur chaque élément.")
    out.append("// Types VÉRIFIÉS à collecter : " + ", ".join(verifie) + ".")
    if a_verifier:
        out.append("// Types A_VERIFIER (compléter le contrat avant de coder) : " + ", ".join(a_verifier) + ".")

    dest = os.path.join(REPO, "providers", prov, "collector_scaffold.go.txt")
    os.makedirs(os.path.dirname(dest), exist_ok=True)
    with open(dest, "w") as f:
        f.write("\n".join(out) + "\n")
    print(f"{len(verifie)} mappers générés ({len(a_verifier)} types à vérifier) → {dest}")


if __name__ == "__main__":
    main()
