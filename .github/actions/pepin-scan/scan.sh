#!/usr/bin/env bash
# Run `pepin scan` and translate its exit code into a CI verdict without ever
# blurring the one distinction a compliance gate lives on: "the tenant is not
# compliant" (exit 1, and exit 3 under --strict) is a verdict, "the scan could
# not conclude" (exit 2) is a failure of the measurement itself.
#
# FAIL_ON_NONCONFORMITY=false downgrades the verdict codes (1 and 3) to a
# warning so a pipeline can report posture without gating on it. It NEVER
# touches exit 2: a pipeline that swallows a technical error is reporting a
# posture nobody measured.
#
# Everything arrives through the environment (set by action.yml from the
# action's inputs) so that no input value is ever interpolated into a script.
set -uo pipefail

PEPIN="${PEPIN_BIN:?PEPIN_BIN must point at the pepin binary}"

# Exactly one source. `pepin scan` would also refuse, but refusing here names
# the action's inputs rather than the CLI's flags.
sources=0
[ -n "${INVENTORY:-}" ] && sources=$((sources + 1))
[ -n "${TERRAFORM_PLAN:-}" ] && sources=$((sources + 1))
[ "${LIVE:-false}" = "true" ] && sources=$((sources + 1))
if [ "${sources}" -ne 1 ]; then
  echo "::error::exactly one of 'inventory', 'terraform-plan' or 'live: true' must be set (got ${sources})" >&2
  exit 2
fi

args=(scan "${PROVIDER:?the provider input is required}")
if [ -n "${INVENTORY:-}" ]; then
  args+=("${INVENTORY}")
elif [ -n "${TERRAFORM_PLAN:-}" ]; then
  args+=(--terraform "${TERRAFORM_PLAN}")
else
  args+=(--live)
fi
[ -n "${REGION:-}" ] && args+=(--region "${REGION}")
[ -n "${FORMAT:-}" ] && args+=(--format "${FORMAT}")
[ -n "${POLICY_DIR:-}" ] && args+=(--policy-dir "${POLICY_DIR}")
[ "${STRICT:-false}" = "true" ] && args+=(--strict)
if [ -n "${SEAL:-}" ]; then
  args+=(--seal "${SEAL}")
  # Redaction is the default when sealing in CI: the bundle embeds the
  # evaluated inventory, and user-data or policy documents can carry the very
  # secrets the rules detect. Opting out is for bundles that must support
  # `pepin verify --re-derive`, and it is an explicit choice.
  [ "${REDACT:-true}" != "false" ] && args+=(--redact)
fi

rc=0
if [ -n "${OUTPUT_FILE:-}" ]; then
  "${PEPIN}" "${args[@]}" > "${OUTPUT_FILE}" || rc=$?
else
  "${PEPIN}" "${args[@]}" || rc=$?
fi

echo "exit-code=${rc}" >> "${GITHUB_OUTPUT}"
case "${rc}" in
  0)
    echo "verdict=compliant" >> "${GITHUB_OUTPUT}"
    exit 0
    ;;
  1|3)
    echo "verdict=non-compliant" >> "${GITHUB_OUTPUT}"
    if [ "${FAIL_ON_NONCONFORMITY:-true}" = "false" ]; then
      echo "::warning::pepin found non-conformities (exit ${rc}); the job continues because fail-on-nonconformity is false"
      exit 0
    fi
    echo "::error::pepin found non-conformities (exit ${rc})"
    exit "${rc}"
    ;;
  2)
    echo "verdict=error" >> "${GITHUB_OUTPUT}"
    echo "::error::pepin could not conclude (exit 2, technical error). This is not a posture verdict, and fail-on-nonconformity never downgrades it."
    exit 2
    ;;
  *)
    echo "verdict=error" >> "${GITHUB_OUTPUT}"
    echo "::error::pepin exited with unexpected code ${rc}"
    exit "${rc}"
    ;;
esac
