#!/usr/bin/env bash
# Pose la protection de la branche par défaut, pendant de apply-tag-ruleset.sh.
#
# Pourquoi. Sans elle, `CODEOWNERS` ne sert à rien : le fichier n'a d'effet que
# si une règle exige la revue d'un CODEOWNER. Et `main` accepte un push direct,
# `--force` compris, alors que release.yml publie depuis cette branche.
#
# Le modèle est le ruleset « main protection » de stephrobert/scankit, qui a
# refusé un push direct pendant l'audit de livraison — le comportement recherché.
# Les status checks exigés sont ceux qui gardent la livraison, et ce sont les
# mêmes que le filtre `ci_jobs` de preflight.sh : les deux doivent rester alignés.
#
# Usage :
#     tools/release/apply-branch-ruleset.sh [owner/repo]
# Idempotent : met à jour le ruleset du même nom plutôt que d'en créer un second.
set -euo pipefail

NAME="main protection"

command -v gh >/dev/null 2>&1 || { echo "gh est requis : https://cli.github.com" >&2; exit 1; }

repo="${1:-}"
[ -z "$repo" ] && repo="$(gh repo view --json nameWithOwner --jq .nameWithOwner 2>/dev/null || true)"
[ -z "$repo" ] && { echo "dépôt introuvable : passer owner/repo, ou configurer un remote origin" >&2; exit 1; }
echo "dépôt : $repo"

payload="$(cat <<'JSON'
{
  "name": "main protection",
  "target": "branch",
  "enforcement": "active",
  "bypass_actors": [
    { "actor_id": 5, "actor_type": "RepositoryRole", "bypass_mode": "pull_request" }
  ],
  "conditions": {
    "ref_name": { "include": ["~DEFAULT_BRANCH"], "exclude": [] }
  },
  "rules": [
    { "type": "deletion" },
    { "type": "non_fast_forward" },
    { "type": "required_linear_history" },
    {
      "type": "pull_request",
      "parameters": {
        "required_approving_review_count": 1,
        "require_code_owner_review": true,
        "dismiss_stale_reviews_on_push": true,
        "require_last_push_approval": false,
        "required_review_thread_resolution": false,
        "allowed_merge_methods": ["squash", "rebase"]
      }
    },
    {
      "type": "required_status_checks",
      "parameters": {
        "strict_required_status_checks_policy": true,
        "required_status_checks": [
          { "context": "Build + Test" },
          { "context": "golangci-lint" },
          { "context": "Audit (gosec + govulncheck)" },
          { "context": "Analyze (go)" },
          { "context": "Dependency Review" },
          { "context": "Secret scanning (pattern-based)" },
          { "context": "The image scans for real" },
          { "context": "The action installs only what it verified" }
        ]
      }
    }
  ]
}
JSON
)"

existing="$(gh api "repos/${repo}/rulesets" --jq ".[] | select(.name == \"${NAME}\") | .id" 2>/dev/null | head -1 || true)"

if [ -n "$existing" ]; then
  echo "mise à jour du ruleset #${existing}"
  printf '%s' "$payload" | gh api --method PUT "repos/${repo}/rulesets/${existing}" --input - >/dev/null
else
  echo "création du ruleset"
  printf '%s' "$payload" | gh api --method POST "repos/${repo}/rulesets" --input - >/dev/null
fi

echo "règles actives sur la branche par défaut :"
gh api "repos/${repo}/rules/branches/main" --jq '.[].type' 2>/dev/null | sort -u | sed 's/^/  /'

cat <<'EOF'

Effet : `main` n'accepte plus de push direct ni de force-push. Tout changement
passe par une pull request approuvée, revue par un CODEOWNER, avec les huit
status checks au vert.

Conséquence à connaître : sur un dépôt à un seul mainteneur, l'approbation vient
du bypass administrateur (`bypass_mode: pull_request`). La protection s'oppose
donc d'abord à l'erreur et aux tiers, pas au propriétaire.
EOF
