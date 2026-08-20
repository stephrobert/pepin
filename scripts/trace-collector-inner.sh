#!/usr/bin/env bash
#
# Second étage de scripts/trace-collector.sh : exécuté DANS l'espace de noms
# (user, mount et net). Ne s'appelle pas seul : il suppose une pile réseau privée
# et un /etc/hosts qu'il a le droit de remplacer.
set -uo pipefail

OUT="${PEPIN_TRACE_OUT:?}"
PROVIDER="${PEPIN_TRACE_PROVIDER:?}"
HOSTS=$(paste -sd, "$OUT/hosts.txt")
PEPIN="${PEPIN_BIN:-./pepin}"

# Un proxy d'entreprise hérité de l'environnement se glisserait dans la mesure et
# la rendrait fausse. Il n'y a pas de mesure sans maîtrise de ce qui la traverse.
unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy ALL_PROXY all_proxy
unset GRPC_PROXY grpc_proxy FTP_PROXY ftp_proxy NO_PROXY no_proxy

ip link set lo up
mount --bind "$OUT/hosts" /etc/hosts

cleanup() {
  # CLAUDE.md §1.1 : rien de ce qui a été lancé ne survit à la mesure. Le SIGTERM
  # laisse aux proxys le temps de vider leur file d'écriture ; un SIGKILL perdrait
  # les derniers échanges, c'est-à-dire précisément ceux de la fin de collecte.
  for pid in "${UP:-}" "${DOWN:-}" "${SERVE:-}"; do
    [ -n "$pid" ] && kill -TERM "$pid" 2>/dev/null && wait "$pid" 2>/dev/null
  done
}
trap cleanup EXIT

# 1. l'émulateur. --vm off : aucun conteneur ne démarre avec vos privilèges.
feint serve --addr 127.0.0.1:4599 --vm off --state "$OUT/feint-state.json" \
  > "$OUT/serve.log" 2>&1 &
SERVE=$!
feint wait --addr 127.0.0.1:4599 --timeout 30s || { cat "$OUT/serve.log"; exit 1; }

waitCA() { # rend le chemin de l'autorité éphémère qu'un proxy vient de frapper
  local log=$1
  for _ in $(seq 1 100); do
    grep -q "CA written to" "$log" 2>/dev/null && break
    sleep 0.2
  done
  sed -n 's/.*CA written to \([^ ]*\).*/\1/p' "$log" | head -1
}

# 2. étage AVAL : sert le TLS des hôtes figés, renvoie à l'émulateur.
feint proxy --addr 127.0.0.1:443 --intercept "$HOSTS" \
  --upstream http://127.0.0.1:4599 --record "$OUT/aval-$PROVIDER.jsonl" \
  > "$OUT/aval-$PROVIDER.log" 2>&1 &
DOWN=$!
CA_DOWN=$(waitCA "$OUT/aval-$PROVIDER.log")
[ -s "$CA_DOWN" ] || { echo "étage aval : pas d'autorité"; cat "$OUT/aval-$PROVIDER.log"; exit 1; }

# 3. étage AMONT : accepte le CONNECT, déchiffre, ENREGISTRE. Sa transcription
#    est celle qui fait foi, car elle voit exactement ce que Pépin a émis.
SSL_CERT_FILE="$CA_DOWN" feint proxy --addr 127.0.0.1:4600 --forward "$HOSTS" \
  --record "$OUT/$PROVIDER.jsonl" > "$OUT/amont-$PROVIDER.log" 2>&1 &
UP=$!
CA_UP=$(waitCA "$OUT/amont-$PROVIDER.log")
[ -s "$CA_UP" ] || { echo "étage amont : pas d'autorité"; cat "$OUT/amont-$PROVIDER.log"; exit 1; }

# 4. Pépin, INCHANGÉ. Les identifiants sont ceux de l'émulateur, qui les accepte
#    tous : aucun secret réel ne traverse quoi que ce soit.
eval "$(feint env "$PROVIDER" 2>/dev/null)"
# L'émulateur publie SCW_API_URL / OSC_ENDPOINT_API / EXOSCALE_API_ENDPOINT.
# Pépin ne les lit PAS : ses base_url sont figées, et c'est la raison d'être de
# toute cette plomberie. On les retire pour que rien ne puisse le prétendre.
unset SCW_API_URL OSC_ENDPOINT_API OSC_PROTOCOL EXOSCALE_API_ENDPOINT SCW_INSECURE
export HTTPS_PROXY=http://127.0.0.1:4600
export SSL_CERT_FILE="$CA_UP"

"$PEPIN" scan "$PROVIDER" --live --format json \
  > "$OUT/scan-$PROVIDER.json" 2> "$OUT/scan-$PROVIDER.err"
echo "pepin scan $PROVIDER --live → code $?"

cleanup; trap - EXIT
echo
echo "transcription : $OUT/$PROVIDER.jsonl"
echo "état de collecte : $OUT/scan-$PROVIDER.json (clé \"collection\")"
echo
echo "AVANT de committer : relire la transcription VALEUR PAR VALEUR."
echo "Les corps sont conservés : contre un vrai tenant ils portent son inventaire."
