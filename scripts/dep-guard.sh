#!/usr/bin/env bash
# Garde-fou des dépendances Go — deux contrôles, statiques, sans exécuter le code :
#   1. scan OSV de go.mod (dépendances vulnérables CVE- et paquets malveillants MAL-) ;
#   2. cooldown d'âge : refuse toute version publiée il y a moins de OSV_MIN_AGE_DAYS
#      jours (par défaut 14) — la plupart des paquets malveillants sont classés MAL-
#      par OSV.dev dans les 24–72 h ; le cooldown laisse passer la fenêtre la plus risquée.
#      EXEMPTÉS : les modules de INTERNAL_MODULE_PREFIXES. Le cooldown couvre le cas
#      d'un dépôt TIERS compromis, dont on apprend la compromission par OSV. Pour nos
#      propres modules, la version vient d'un commit relu, dont la CI est verte et
#      qu'on vient de taguer soi-même : attendre 14 jours n'ajoute aucune information,
#      et bloquerait la propagation d'un correctif de sécurité (cas réel : scankit
#      v0.2.2, qui fermait une exfiltration via les politiques). Le scan OSV, lui,
#      reste appliqué à TOUS les modules, sans exception.
#
# Go n'a pas d'équivalent natif à npm `min-release-age` / uv `--exclude-newer` :
# l'âge est vérifié via le module proxy (proxy.golang.org .../@v/<ver>.info → champ Time).
#
# Usage : dep-guard.sh [module@version ...]
#   Le scan OSV tourne toujours ; l'âge n'est vérifié que pour les module@version donnés.
# Sortie : 0 si tout est vert ; 1 + message sur stderr si une garde échoue.
set -uo pipefail

MIN_AGE_DAYS="${OSV_MIN_AGE_DAYS:-14}"
# Modules publiés par le projet lui-même (préfixes, un par ligne).
INTERNAL_MODULE_PREFIXES="${INTERNAL_MODULE_PREFIXES:-github.com/stephrobert/}"
fail=0
msg=""

# 1) Scan OSV (statique, lit go.mod sans rien installer).
if command -v osv-scanner >/dev/null 2>&1 && [ -f go.mod ]; then
  out="$(osv-scanner scan -L go.mod 2>&1)"
  if [ $? -eq 1 ]; then
    fail=1
    msg+="OSV-Scanner : dépendance vulnérable ou paquet malveillant détecté.\n${out}\n"
  fi
fi

# 2) Cooldown d'âge pour chaque module ajouté (module@version).
for spec in "$@"; do
  case "$spec" in *@*) ;; *) continue ;; esac
  # Module interne : cooldown non applicable (cf. en-tête). Le scan OSV ci-dessus
  # l'a déjà couvert, lui.
  skip=0
  for pfx in $INTERNAL_MODULE_PREFIXES; do
    case "$spec" in "$pfx"*) skip=1 ;; esac
  done
  if [ "$skip" -eq 1 ]; then
    printf '  · cooldown non applicable à %s (module interne)\n' "${spec%%@*}" >&2
    continue
  fi
  age="$(python3 - "$spec" <<'PY'
import sys, json, datetime, urllib.request
spec = sys.argv[1]
mod, ver = spec.rsplit("@", 1)
if not ver or ver == "":
    print("ERR"); sys.exit()
# Encodage proxy.golang.org : chaque majuscule -> "!" + minuscule.
enc = "".join("!" + c.lower() if c.isupper() else c for c in mod)
url = f"https://proxy.golang.org/{enc}/@v/{ver}.info"
try:
    with urllib.request.urlopen(url, timeout=5) as r:
        t = json.load(r).get("Time", "")
    d = datetime.datetime.fromisoformat(t.replace("Z", "+00:00"))
    print((datetime.datetime.now(datetime.timezone.utc) - d).days)
except Exception:
    print("ERR")
PY
)"
  [ "$age" = "ERR" ] && continue            # date inconnue → le scan OSV reste le filet
  if [ "$age" -lt "$MIN_AGE_DAYS" ]; then
    fail=1
    msg+="Cooldown : ${spec} publié il y a ${age} j (< ${MIN_AGE_DAYS} j) — version trop fraîche, attendez la fenêtre de quarantaine.\n"
  fi
done

if [ "$fail" -ne 0 ]; then
  printf '%b' "$msg" >&2
  exit 1
fi
exit 0
