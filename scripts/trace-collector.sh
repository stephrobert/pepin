#!/usr/bin/env bash
#
# Enregistre les appels HTTP RÉELS d'un collecteur Pépin, contre un ÉMULATEUR
# LOCAL, sans aucun identifiant cloud et sans modifier une ligne de Pépin.
#
#   ./scripts/trace-collector.sh scaleway [répertoire de sortie]
#
# ─── CE QUE CETTE PROCÉDURE ÉTABLIT, ET CE QU'ELLE N'ÉTABLIT PAS ─────────────
#
# Elle établit ce que Pépin FAIT : les endpoints réellement émis face à ceux que
# providers/<nom>.yaml déclare, les jointures parent→enfant qui tirent, les
# paramètres de pagination émis, et la classe attribuée à un échec de collecte.
#
# Elle n'établit RIEN de ce que le fournisseur RÉPOND : ni les noms et types
# exacts des champs de son contrat natif, ni ses bornes réelles de pagination, ni
# son comportement de limitation de débit, ni son refus par défaut de droits.
# Un émulateur qui accepte tout identifiant ne peut, par construction, produire
# aucun 403. Cela reste dû à un scan réel, contre un tenant réel.
#
# ─── POURQUOI CETTE CHAÎNE-LÀ ────────────────────────────────────────────────
#
# Aucune base_url de collecte n'est redirigeable (seul --s3-endpoint l'est). Le
# client de collecte, lui, n'installe pas de Transport : il hérite de
# http.DefaultTransport, DONC il honore HTTPS_PROXY. Un seul étage suffit alors,
# et il ne demande aucune modification de Pépin, donc aucune surface
# d'exfiltration : un endpoint de collecte surchargeable serait un moyen
# d'envoyer la clé secrète d'un tenant vers un hôte arbitraire (ADR-0012).
#
#   Pépin ──HTTPS_PROXY, CONNECT──▶ feint proxy --forward api.x=http://…:PORT
#                                        │ enregistre, puis redial
#                                        ▼
#                                   feint serve (l'émulateur)
#
# Cette procédure exigeait naguère un espace de noms utilisateur, un /etc/hosts
# de remplacement et le port 443 privilégié, parce que feint 0.10.0 refusait
# --forward et --upstream ensemble. feint ≥ 0.12 accepte `host=target` : l'hôte
# demandé reste dans la transcription, seule la socket va ailleurs. Les trois
# contraintes tombent, et la procédure tourne désormais là où
# apparmor_restrict_unprivileged_userns=1 interdit `unshare` (issue #92).
#
# Rien n'écoute hors de la boucle locale. --vm off interdit à l'émulateur de
# démarrer le moindre conteneur avec vos privilèges, donc rien n'est provisionné
# et rien n'est à détruire (CLAUDE.md §1.1).
#
# ─── LA PRÉCAUTION QUI NE SE NÉGOCIE PAS ─────────────────────────────────────
#
# La rédaction du proxy protège les EN-TÊTES (liste blanche). Les CORPS sont la
# mesure, donc conservés : contre un vrai tenant ils portent les identifiants de
# ses ressources, ses noms de buckets et ses adresses IP. AUCUN enregistrement
# n'entre au dépôt sans relecture VALEUR PAR VALEUR. L'assainissement partiel est
# le piège : l'audit de livraison s'est ouvert sur un UUID d'instance réelle
# oublié dans une fixture dont l'adresse IP avait pourtant été assainie.
set -uo pipefail

PROVIDER="${1:?usage: trace-collector.sh <scaleway|outscale|exoscale> [outdir]}"
OUT="${2:-$(mktemp -d)}"
mkdir -p "$OUT"

command -v feint >/dev/null || { echo "feint absent du PATH (https://github.com/…/feint)"; exit 2; }

# Les hôtes que les descripteurs figent. Un hôte non nommé ici voit son CONNECT
# REFUSÉ par le proxy amont, qui le signale à l'arrêt. C'est ainsi que l'API
# Kubernetes managé d'Outscale, servie sur un hôte à part, s'est fait remarquer.
cat > "$OUT/hosts.txt" <<'EOF'
api.scaleway.com
s3.fr-par.scw.cloud
api.eu-west-2.outscale.com
oos.eu-west-2.outscale.com
api.eu-west-2.oks.outscale.com
api-ch-dk-2.exoscale.com
api-ch-gva-2.exoscale.com
sos-ch-dk-2.exo.io
sos-ch-gva-2.exo.io
EOF

exec "$(dirname "$0")/trace-collector-inner.sh" "$PROVIDER" "$OUT"
