#!/usr/bin/env bash
# Download a released pepin binary, verify its SHA-256 against the published
# checksum list, and install it. Nothing runs before the bytes are checked:
# an action that pipes a download into a shell is the supply-chain shape this
# repository spends a release workflow proving it does not have.
#
# Usage: install.sh <version-without-v> <dest-dir> [base-url]
#
# The third argument exists for one caller only: the CI job that proves this
# verification both accepts a genuine binary and refuses a corrupted one, by
# serving locally built artefacts over a loopback HTTP server. The composite
# action never passes it, so every consumer downloads from the GitHub release.
set -euo pipefail

VERSION="${1:?usage: install.sh <version> <dest-dir> [base-url]}"
DEST="${2:?usage: install.sh <version> <dest-dir> [base-url]}"
BASE="${3:-https://github.com/stephrobert/pepin/releases/download/v${VERSION}}"

case "$(uname -s)-$(uname -m)" in
  Linux-x86_64)  asset="pepin-linux-amd64" ;;
  Linux-aarch64) asset="pepin-linux-arm64" ;;
  Darwin-x86_64) asset="pepin-darwin-amd64" ;;
  Darwin-arm64)  asset="pepin-darwin-arm64" ;;
  *) echo "no published binary for $(uname -s)-$(uname -m)" >&2; exit 1 ;;
esac

workdir="$(mktemp -d)"
trap 'rm -rf "${workdir}"' EXIT

curl --retry 3 --retry-connrefused -fsSL -o "${workdir}/${asset}" "${BASE}/${asset}"
curl --retry 3 --retry-connrefused -fsSL -o "${workdir}/checksums.txt" "${BASE}/checksums.txt"

# Authenticity BEFORE integrity. Comparing the binary against checksums.txt
# proves only that the two files agree, and both come from the same origin:
# whoever can replace the release assets replaces both, and the check passes.
# The release publishes build provenance attestations precisely to settle this,
# so the installer verifies them instead of pointing at the docs.
#
# `gh` ships on every GitHub-hosted runner. The skip is an EXPLICIT opt-out, not
# a side effect of passing a base URL: the CI job that serves locally built
# artefacts has no attestation to show, but every other caller must be covered
# -- and a test can then prove the check actually refuses an unattested binary.
if [ "${PEPIN_SKIP_ATTESTATION:-0}" != "1" ]; then
  if command -v gh >/dev/null 2>&1; then
    gh attestation verify "${workdir}/${asset}" \
      --repo stephrobert/pepin \
      --signer-workflow stephrobert/pepin/.github/workflows/release.yml \
      || { echo "provenance non vérifiée pour ${asset} : installation refusée" >&2; exit 1; }
  else
    echo "AVERTISSEMENT : gh absent, provenance non vérifiée — empreinte seule." >&2
    echo "  Vérification manuelle : docs/install.md (cosign verify-blob)." >&2
  fi
fi

# The bytes are checked before anything runs them.
(cd "${workdir}" && grep " ${asset}$" checksums.txt | sha256sum -c -)

install -m 0755 "${workdir}/${asset}" "${DEST}/pepin"
echo "pepin ${VERSION} installed in ${DEST} (checksum verified)"
