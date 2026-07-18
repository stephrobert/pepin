#!/usr/bin/env bash
# Ajout sûr d'une dépendance Go — on ne « fait pas confiance puis on vérifie »,
# on vérifie d'abord : résoudre (sans exécuter) → scan OSV + cooldown d'âge →
# finaliser seulement si tout est vert, sinon tout annuler.
#
# Adaptation Go du safe-add du guide OSV-Scanner. Nuance : `go get` télécharge la
# source dans le cache mais NE l'exécute PAS (pas de scripts pre/post-install à la
# npm) ; le risque d'exécution n'arrive qu'au `go build`/`test`. On scanne donc
# juste après `go get`, avant tout build, et on annule via go.mod/go.sum si KO.
#
# Usage : scripts/safe-add.sh <module[@version]>
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PKG="${1:?usage: safe-add.sh <module[@version]>}"
HERE="$(dirname "$0")"

snap="$(mktemp -d)"
cp go.mod "$snap/go.mod"
[ -f go.sum ] && cp go.sum "$snap/go.sum"
restore() {
  cp "$snap/go.mod" go.mod
  [ -f "$snap/go.sum" ] && cp "$snap/go.sum" go.sum
  rm -rf "$snap"
}

before="$(go list -m -f '{{.Path}}@{{.Version}}' all 2>/dev/null | sort)"

echo "[1/3] go get ${PKG} — résolution et téléchargement, sans exécution"
if ! go get "$PKG"; then
  echo "go get a échoué." >&2
  restore
  exit 1
fi

after="$(go list -m -f '{{.Path}}@{{.Version}}' all 2>/dev/null | sort)"
newmods="$(comm -13 <(printf '%s\n' "$before") <(printf '%s\n' "$after"))"

echo "[2/3] Vérification OSV + cooldown ${OSV_MIN_AGE_DAYS:-14} j (modules ajoutés)"
# shellcheck disable=SC2086 -- on veut le découpage en arguments des modules.
if ! bash "$HERE/dep-guard.sh" $newmods; then
  echo "[REFUSÉ] ${PKG} (ou une transitive) ne passe pas les gardes — annulation." >&2
  restore
  exit 1
fi

rm -rf "$snap"
echo "[3/3] Vert — finalisation (go mod tidy)"
go mod tidy
echo "OK : ${PKG} ajouté et vérifié."
