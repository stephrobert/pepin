> 🇬🇧 English · [🇫🇷 Français](install.fr.md)

# Installing Pépin

One Go binary, no daemon, no dependency. Four ways in, each verified before
anything runs: the released binary, the container image, the GitHub action,
the GitLab template. `0.1.0` below names a released version — a mutable
`latest` installs whatever is newest, which is a binary nobody can name
afterwards.

## The released binary

Needs `cosign` (or `gh`, below) and nothing else. Every file is fetched to
disk and verified before anything runs it.

```bash
base=https://github.com/stephrobert/pepin/releases/download/v0.1.0

case "$(uname -s)-$(uname -m)" in
  Linux-x86_64)  asset=pepin-linux-amd64 ;;
  Linux-aarch64) asset=pepin-linux-arm64 ;;
  Darwin-x86_64) asset=pepin-darwin-amd64 ;;
  Darwin-arm64)  asset=pepin-darwin-arm64 ;;
  *) echo "no published binary for $(uname -s)-$(uname -m)" >&2; exit 1 ;;
esac

curl -fsSLO "$base/$asset"
curl -fsSLO "$base/checksums.txt"
curl -fsSLO "$base/checksums.txt.cosign.bundle"

# Who produced the checksum list, before trusting a single hash inside it.
# The identity names the release workflow and the tag ref, not the repository:
# a repository-wide pattern would accept any workflow that ever gets
# id-token: write.
cosign verify-blob --bundle checksums.txt.cosign.bundle \
  --certificate-identity-regexp '^https://github\.com/stephrobert/pepin/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt

sha256sum -c checksums.txt --ignore-missing

install -m 0755 "$asset" ~/.local/bin/pepin
```

With `gh`, which checks the build provenance instead — it proves which
workflow and which commit produced the binary:

```bash
gh release download v0.1.0 --repo stephrobert/pepin --pattern 'pepin-linux-amd64'
gh attestation verify pepin-linux-amd64 --repo stephrobert/pepin \
  --signer-workflow stephrobert/pepin/.github/workflows/release.yml
```

Building from source needs a Go toolchain and gives up every check above,
since nothing is signed until it is released:

```bash
go install github.com/stephrobert/pepin@latest
```

## The container image

For the CI that consumes an image rather than a binary. The binary inside is
the released binary — nothing is compiled in the Dockerfile, so the release's
checksums, SBOM and provenance describe the image's content too. The base is
distroless (CA roots for `--live`'s TLS, user 65532, no shell, no package
manager), and the release workflow refuses to push an image whose
`pepin version` is not the tag or whose exit codes have moved.

One tag per release, no `latest`. Verify, then run:

```bash
cosign verify ghcr.io/stephrobert/pepin:v0.1.0 \
  --certificate-identity-regexp '^https://github\.com/stephrobert/pepin/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com

# audit a Terraform plan: no credential, nothing provisioned
docker run --rm -v "$PWD:/work" ghcr.io/stephrobert/pepin:v0.1.0 \
  scan scaleway --terraform /work/plan.json
```

Three practical points, all measured:

- **Credentials never enter the image.** For `--live`, pass the provider's
  own variables at run time and nothing else:
  `docker run --rm -e OSC_ACCESS_KEY -e OSC_SECRET_KEY -e OSC_REGION ghcr.io/stephrobert/pepin:v0.1.0 scan outscale --live`
  (naming the variables without `=` forwards them from your environment
  without putting their values in the command line or your shell history).
- **Sealing a bundle into a mounted volume needs your uid**, because the
  image runs as 65532 and the volume belongs to you:
  `docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/work" ... scan ... --seal /work/bundle --redact`.
- **`pepin verify --pubkey` does not work inside the image**: it executes the
  `cosign` binary, and a distroless image deliberately has none. Verify
  signatures on the host; `pepin verify` without `--pubkey` (integrity only)
  works anywhere.

## GitHub Actions

The composite action downloads the released binary, **verifies its SHA-256
against the release's checksum list before anything runs it** (a CI job in
this repository corrupts one byte of that download and requires the refusal),
scans, and turns the exit codes into a gate:

```yaml
- uses: stephrobert/pepin/.github/actions/pepin-scan@v0.1.0
  with:
    version: 0.1.0
    provider: scaleway
    terraform-plan: plan.json
```

`fail-on-nonconformity: 'false'` reports the posture (exit 1 becomes a
warning) instead of gating on it; a technical error (exit 2) fails the job
whatever that input says, because a swallowed error is a posture nobody
measured. Credentials are never inputs: for `live: true`, put the provider's
own variables in `env:` from repository secrets. Full example:
[`examples/github-actions/pepin.yml`](../examples/github-actions/pepin.yml).

## GitLab CI

Same doctrine, as an includable template — binary verified in
`before_script`, exit codes as the contract, report-only via
`allow_failure: exit_codes: [1, 3]` (never 2):

```yaml
include:
  - remote: 'https://raw.githubusercontent.com/stephrobert/pepin/v0.1.0/examples/gitlab-ci/pepin.gitlab-ci.yml'
```

Full example: [`examples/gitlab-ci/`](../examples/gitlab-ci/).

## Credentials, in every mode

Pépin reads each provider's **native** environment variables (or its native
config file), so a credential that already works for the provider's own CLI
works here, and nothing is renamed on the way:

| provider | environment |
|---|---|
| Scaleway | `SCW_ACCESS_KEY`, `SCW_SECRET_KEY`, `SCW_DEFAULT_ORGANIZATION_ID`, `SCW_DEFAULT_REGION` |
| Outscale | `OSC_ACCESS_KEY`, `OSC_SECRET_KEY`, `OSC_REGION` |
| Exoscale | `EXOSCALE_API_KEY`, `EXOSCALE_API_SECRET`, `EXOSCALE_ZONE` |

Only `--live` needs any of it. The Terraform-plan mode and the inventory mode
run with no account at all, which is why every CI example gates on the plan
first. Pépin never prints a secret; what deserves your attention instead is
the **evidence bundle** (`--seal`): it embeds the evaluated inventory of a
real tenant, and user-data or policy documents in that inventory can carry
the very secrets the rules detect. In CI, seal with `--redact` (the GitHub
action's default), keep artifact retention short, and remember that artifacts
of a public project are downloadable. Drop `--redact` only when a bundle must
support `pepin verify --re-derive`, and then treat the bundle as a secret.
