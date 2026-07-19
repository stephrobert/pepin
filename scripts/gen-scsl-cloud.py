#!/usr/bin/env python3
"""Transpose les contrôles de Pépin (catalogue + controles.yaml) en exigences
SCSL FINES (une exigence par contrôle) pour le module posture cloud du framework.

Source : referentiel/catalogue.yaml (liste cible complète) ∪ controles.yaml (29
actifs, champs FR curatés). Aucune invention : pour les actifs, texte = titre,
correspondance = frameworks ; pour les `a_trier`, texte = code + redaction:a_faire.
Sortie : framework-scsl/source/cloud-controls.generated.yaml (régénérable, en cours
de gel dans SCSL). Usage : python3 scripts/gen-scsl-cloud.py [chemin_framework_scsl]
"""
import sys, os, yaml

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CONTROLES = os.path.join(REPO, "referentiel", "controles.yaml")
CATALOGUE = os.path.join(REPO, "referentiel", "catalogue.yaml")
FW = sys.argv[1] if len(sys.argv) > 1 else os.path.join(REPO, "..", "framework-scsl")
OUT = os.path.join(FW, "source", "cloud-controls.generated.yaml")

# famille SCSL déduite du préfixe de service du code agnostique.
PREFAM = {"network": "NET", "compute": "CMP", "objectstorage": "STO", "blockstorage": "STO",
          "iam": "IAM", "kubernetes": "CMP", "loadbalancer": "CHF", "governance": "GOV"}
FAM = {"iam": "IAM", "reseau": "NET", "compute": "CMP", "stockage": "STO",
       "chiffrement": "CHF", "journalisation": "LOG", "gouvernance": "GOV"}
# sévérité → (niveau, devoir)
LVL = {"critical": ("R1", "DOIT"), "high": ("R1", "DOIT"),
       "medium": ("R2", "DEVRAIT"), "low": ("R3", "DEVRAIT")}
FWLABEL = {"secnumcloud_3_2": "SecNumCloud {}",
           "cis_controls_v8": "CIS Controls v8 {}", "iso_27001_2022": "ISO 27001 ({})",
           "iso_27017": "ISO 27017 ({})"}


def correspondances(frameworks):
    out = []
    for key, items in (frameworks or {}).items():
        tmpl = FWLABEL.get(key, key + " {}")
        out += [tmpl.format(i) for i in items]
    return out


def main():
    with open(CONTROLES) as f:
        controles = {c["code"]: c for c in yaml.safe_load(f)["controles"]}
    with open(CATALOGUE) as f:
        catalogue = {e["code"]: e for e in yaml.safe_load(f)["catalogue"]}

    codes = sorted(set(catalogue) | set(controles))
    exigences = []
    for code in codes:
        ctl = controles.get(code)          # données riches (FR) si contrôle actif
        cat = catalogue.get(code, {})
        severite = (ctl or cat).get("severite", "medium")
        niveau, devoir = LVL.get(severite, ("R2", "DEVRAIT"))
        famille = FAM.get(ctl["famille"]) if ctl else PREFAM.get(code.split("_")[0], "CLD")
        ex = {
            "id": code, "ref": code, "domaine": "CLD", "couche": "module",
            "module": "SCSL-C", "famille": famille, "niveau": niveau, "devoir": devoir,
            "texte": ctl.get("titre") if ctl else code,
            "scsl_parent": (ctl.get("scsl") or [None])[0] if ctl else None,
            "correspondance": correspondances(ctl.get("frameworks")) if ctl else [],
            "outillage": ["Pépin: " + code],
            "statut": "actif" if ctl else "a_trier",
            "fournisseurs": ctl.get("fournisseurs", []) if ctl else [],
        }
        if not ctl:
            ex["redaction"] = "a_faire"  # titre FR + correspondances à rédiger
        exigences.append(ex)

    header = ("# Exigences SCSL FINES du module posture cloud — GÉNÉRÉ (en cours de gel).\n"
              "# Source : referentiel/{catalogue,controles}.yaml de Pépin. Régénérer via\n"
              "# `make gen-scsl`. Une exigence par contrôle (ref = code agnostique) ;\n"
              "# scsl_parent = famille CLD-* de regroupement ; statut actif|a_trier.\n")
    os.makedirs(os.path.dirname(OUT), exist_ok=True)
    with open(OUT, "w") as f:
        f.write(header)
        yaml.safe_dump({"exigences_cloud_detail": exigences}, f,
                       allow_unicode=True, sort_keys=False, width=1000)
    print(f"{len(exigences)} exigences SCSL générées → {OUT}")


if __name__ == "__main__":
    main()
