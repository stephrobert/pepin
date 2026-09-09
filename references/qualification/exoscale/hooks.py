#!/usr/bin/env python3
"""Crochets Exoscale du tenant de qualification — PLAN SEUL.

Aucun compte Exoscale n'est disponible (tenant.yaml `live: unavailable`). Le runner
ne demande donc à ces crochets que `variables`, pour donner au provider Terraform
les identifiants FACTICES qu'un plan sans source de données n'emploie jamais (les
mêmes que scripts/tenant-plan.py). Les autres commandes REFUSENT, explicitement :
une identité qu'on ne peut pas confronter et un inventaire qu'on ne peut pas relire
ne se simulent pas.

Quand un compte existera : `identity` devra demander l'organisation à l'API v2
(GET /organization, signature EXO2-HMAC-SHA256 — celle que internal/collectkit
implémente), `inventory` lister les familles (instances, groupes, réseaux privés,
volumes et snapshots block, clusters SKS, rôles et clés IAM, buckets SOS par zone),
`cleanup` supprimer par le préfixe du tenant, et `extra` rester vide si le provider
sait tout créer. Les issues de destruction de exoscale/terraform-provider-exoscale
sont à lire d'abord.
"""
import json
import sys


def variables(mode, identity_path):
    if mode != "--plan-only":
        sys.exit("aucun compte Exoscale : ce tenant est plan seul (tenant.yaml `live: unavailable`)")
    return {
        "tf_vars": {},
        # Forme attendue par le provider ; aucune valeur réelle. Un plan n'appelle pas l'API.
        "env": {"EXOSCALE_API_KEY": "EXOxxxxxxxxxxxxxxxxxxxx",
                "EXOSCALE_API_SECRET": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
        "wait_until": None,
        "note": "plan seul : identifiants factices pour le provider, rien à attendre",
    }


def main(argv):
    if len(argv) < 2:
        sys.exit(__doc__)
    if argv[1] == "variables":
        print(json.dumps(variables(argv[2], argv[3])))
        return 0
    if argv[1] == "extra":
        print(json.dumps({"created": [], "deleted": [], "left": []}))
        return 0
    sys.exit(f"{argv[1]} : aucun compte Exoscale — ce tenant est plan seul (tenant.yaml `live: unavailable`)")


if __name__ == "__main__":
    sys.exit(main(sys.argv))
