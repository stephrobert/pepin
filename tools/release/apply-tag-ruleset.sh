#!/usr/bin/env bash
# Pose le ruleset qui manque à la chaîne de release : restreindre la CRÉATION des
# tags `v*` aux administrateurs du dépôt.
#
# Pourquoi ce script existe. `mise run release-check` (preflight.sh) est un
# contrôle LOCAL : rien ne relie un tag au fait de l'avoir exécuté. `git tag
# v1.0.0 && git push origin v1.0.0` déclenche release.yml et publie une release
# complète, signée et attestée, sans qu'une ligne du preflight ait tourné.
# release.yml rejoue depuis le job `gate` les portes vérifiables hors ligne, ce
# qui ferme la moitié du trou ; l'autre moitié — QUI a le droit de poser un tag —
# est de la configuration de dépôt et ne peut pas vivre dans le dépôt.
#
# Le modèle est le ruleset « main protection » déjà en place sur stephrobert/scankit
# (actor_id 5 = rôle admin), transposé de la branche vers les tags.
#
# Usage :
#     tools/release/apply-tag-ruleset.sh [owner/repo]
# Sans argument, le dépôt est déduit du remote `origin`.
#
# Idempotent : si un ruleset du même nom existe déjà, il est mis à jour (PUT)
# plutôt que dupliqué.
set -euo pipefail

NAME="tag protection (v*)"

command -v gh >/dev/null 2>&1 || {
  echo "gh est requis : https://cli.github.com" >&2
  exit 1
}

repo="${1:-}"
if [ -z "$repo" ]; then
  repo="$(gh repo view --json nameWithOwner --jq .nameWithOwner 2>/dev/null || true)"
fi
if [ -z "$repo" ]; then
  echo "dépôt introuvable : passer owner/repo en argument, ou configurer un remote origin" >&2
  exit 1
fi
echo "dépôt : $repo"

payload="$(cat <<'JSON'
{
  "name": "tag protection (v*)",
  "target": "tag",
  "enforcement": "active",
  "bypass_actors": [
    { "actor_id": 5, "actor_type": "RepositoryRole", "bypass_mode": "always" }
  ],
  "conditions": {
    "ref_name": { "include": ["refs/tags/v*"], "exclude": [] }
  },
  "rules": [
    { "type": "creation" },
    { "type": "deletion" },
    { "type": "update" }
  ]
}
JSON
)"

# Un ruleset du même nom existe-t-il déjà ?
existing="$(gh api "repos/${repo}/rulesets" --jq \
  ".[] | select(.name == \"${NAME}\") | .id" 2>/dev/null | head -1 || true)"

if [ -n "$existing" ]; then
  echo "mise à jour du ruleset #${existing}"
  printf '%s' "$payload" | gh api --method PUT "repos/${repo}/rulesets/${existing}" --input - >/dev/null
else
  echo "création du ruleset"
  printf '%s' "$payload" | gh api --method POST "repos/${repo}/rulesets" --input - >/dev/null
fi

echo "règles actives sur les tags :"
gh api "repos/${repo}/rulesets" --jq \
  ".[] | select(.target == \"tag\") | \"  \(.name) [\(.enforcement)]\""

cat <<'EOF'

Effet : seul un administrateur peut créer, supprimer ou déplacer un tag `v*`.
Un tag posé par quelqu'un d'autre est refusé AVANT que release.yml ne démarre,
donc avant toute publication.

À vérifier une fois posé — le contrôle porte sur le comportement :
    git tag v0.0.0-ruleset-test && git push origin v0.0.0-ruleset-test
avec un compte non-administrateur : le push doit être refusé.
Nettoyer ensuite : git tag -d v0.0.0-ruleset-test
EOF
