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

# The bytes are checked before anything runs them. The cosign signature over
# the checksum list is the stronger half and needs cosign, which a CI job may
# not have; docs/install.md carries that command for anyone who wants it.
(cd "${workdir}" && grep " ${asset}$" checksums.txt | sha256sum -c -)

install -m 0755 "${workdir}/${asset}" "${DEST}/pepin"
echo "pepin ${VERSION} installed in ${DEST} (checksum verified)"
