#!/usr/bin/env bash
#
# Étage unique. Appelé par trace-collector.sh, qui a préparé $OUT/hosts.txt.
#
# ─── POURQUOI CE SCRIPT A MAIGRI ─────────────────────────────────────────────
#
# Il tenait 89 lignes et exigeait un espace de noms utilisateur, un /etc/hosts de
# remplacement et le port 443 privilégié. La raison était écrite dans son propre
# en-tête : feint 0.10.0 REFUSAIT --forward et --upstream ensemble, parce que le
# premier envoie chaque requête à l'hôte que le CLIENT a demandé, le second à
# l'hôte qu'on a CHOISI. Il fallait donc un second étage pour que « l'hôte
# demandé » devienne l'émulateur, et un /etc/hosts privé pour l'y résoudre.
#
# feint ≥ 0.12 accepte `host=target` dans --forward : l'hôte demandé est
# conservé dans la transcription, seule la socket va ailleurs. Le second étage,
# l'espace de noms et le port privilégié disparaissent tous les trois.
#
# Ce que cela débloque concrètement : la procédure tourne désormais sur un hôte
# portant apparmor_restrict_unprivileged_userns=1, où `unshare` échoue — c'est
# le cas de la machine de développement, et c'était l'objet de l'issue #92.
#
#   Pépin ──HTTPS_PROXY, CONNECT──▶ feint proxy --forward api.x=http://…:PORT
#                                        │ enregistre, puis redial
#                                        ▼
#                                   feint serve (l'émulateur)
#
# Aucune ligne de Pépin n'est modifiée, donc aucune surface d'exfiltration n'est
# créée : un endpoint de collecte surchargeable serait un moyen d'envoyer la clé
# secrète d'un tenant vers un hôte arbitraire (ADR-0012).
set -euo pipefail

PROVIDER=${1:?provider}
OUT=${2:?répertoire de sortie}
HOSTS=$(paste -sd, "$OUT/hosts.txt")

# Les PORTS sont configurables, et ils doivent l'être : feint tourne souvent en
# parallèle pour d'autres projets sur la même machine, et un port figé transforme
# une collision en échec incompréhensible.
FEINT_PORT="${FEINT_PORT:-4599}"
PROXY_PORT="${PROXY_PORT:-4600}"

# Refuser tôt : un port déjà pris donne sinon un timeout de 30 s sans raison lisible.
for p in "$FEINT_PORT" "$PROXY_PORT"; do
  if ss -ltn 2>/dev/null | grep -qE "[:.]${p}\b"; then
    echo "port $p déjà occupé — feint tourne peut-être pour un autre projet." >&2
    echo "Relancer avec FEINT_PORT / PROXY_PORT sur des ports libres." >&2
    exit 2
  fi
done

# `feint stop` ne connaît que les instances lancées par `feint start`, qui les
# enregistre ; un `feint serve &` lui est invisible — il répond « nothing recorded »
# et laisse le processus vivant. On garde donc les PID et on les tue. Vérifié en
# exécution : sans cela, l'émulateur survivait au script et gardait son port.
cleanup() {
  for pid in "${PROXY_PID:-}" "${SERVE_PID:-}"; do
    [ -n "$pid" ] && kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

# 1. l'émulateur. --vm off : aucun conteneur n'est démarré avec les privilèges de
#    l'utilisateur, et il n'y a donc rien à détruire ensuite (CLAUDE.md §1.1).
feint serve --addr "127.0.0.1:$FEINT_PORT" --vm off --state "$OUT/feint-state.json" \
  >"$OUT/serve.log" 2>&1 &
SERVE_PID=$!
feint wait --addr "127.0.0.1:$FEINT_PORT" --timeout 30s || { cat "$OUT/serve.log"; exit 1; }

# 2. le proxy direct, qui termine le TLS et renvoie chaque hôte vers l'émulateur.
FORWARD=$(tr ',' '\n' <<<"$HOSTS" | sed "s|\$|=http://127.0.0.1:$FEINT_PORT|" | paste -sd,)
feint proxy --addr "127.0.0.1:$PROXY_PORT" --forward "$FORWARD" \
  --record "$OUT/$PROVIDER.jsonl" >"$OUT/proxy-$PROVIDER.log" 2>&1 &
PROXY_PID=$!

waitCA() { # rend le chemin de l'autorité éphémère que le proxy vient de frapper
  for _ in $(seq 1 100); do
    grep -q "CA written to" "$OUT/proxy-$PROVIDER.log" 2>/dev/null && break
    sleep 0.2
  done
  sed -n 's/.*CA written to \([^ ]*\).*/\1/p' "$OUT/proxy-$PROVIDER.log" | head -1
}
CA=$(waitCA)
[ -s "$CA" ] || { echo "le proxy n'a frappé aucune autorité :" >&2; cat "$OUT/proxy-$PROVIDER.log" >&2; exit 1; }

# 3. le scan. Les identifiants sont SYNTHÉTIQUES : l'émulateur accepte tout, et
#    aucun secret réel n'entre donc dans la procédure ni dans l'enregistrement.
SSL_CERT_FILE="$CA" \
HTTPS_PROXY="http://127.0.0.1:$PROXY_PORT" \
HTTP_PROXY="http://127.0.0.1:$PROXY_PORT" \
SCW_ACCESS_KEY=SCWXXXXXXXXXXXXXXXXX \
SCW_SECRET_KEY=11111111-1111-1111-1111-111111111111 \
SCW_DEFAULT_ORGANIZATION_ID=11111111-1111-1111-1111-111111111111 \
SCW_DEFAULT_PROJECT_ID=11111111-1111-1111-1111-111111111111 \
SCW_DEFAULT_REGION=fr-par \
OSC_ACCESS_KEY=SCWXXXXXXXXXXXXXXXXX \
OSC_SECRET_KEY=11111111-1111-1111-1111-111111111111 \
OSC_REGION=eu-west-2 \
EXOSCALE_API_KEY=EXOxxxxxxxxxxxxxxxxxxxx \
EXOSCALE_API_SECRET=11111111-1111-1111-1111-111111111111 \
  ./pepin scan "$PROVIDER" --live --format json >"$OUT/$PROVIDER-scan.json" 2>"$OUT/$PROVIDER-scan.log" || true

echo "enregistrement : $OUT/$PROVIDER.jsonl ($(wc -l <"$OUT/$PROVIDER.jsonl" 2>/dev/null || echo 0) échanges)"
