> 🇬🇧 English · [🇫🇷 Français](SECURITY.fr.md)

# Security policy

Pépin is a security tool: its reliability is a requirement, not an option.

## Reporting a vulnerability

Please do **not** open a public issue for a vulnerability.

Report it privately here:
**https://github.com/stephrobert/pepin/security/advisories/new**
(GitHub → **Security** → **Report a vulnerability**). The advisory stays private
between you and the maintainer until a fix ships.

If that form is unavailable to you, contact the maintainer through any private
channel rather than a public issue.

Please include:

- a description of the vulnerability and its impact;
- reproduction steps (version/commit, command, output);
- a proposed fix, if you have one.

We acknowledge receipt within a few business days and keep you informed of the
fix and its release.

## Scope

Of particular concern:

- **result integrity**: a control that would come out `pass` while not actually
  evaluated, or an evidence/bundle that could be tampered with undetected;
- **credential handling**: any path where a secret (key, token) could leak into a
  command-line argument, a log, or an artifact;
- **execution**: injection via a provider descriptor, a Terraform plan, or an
  inventory supplied as input.

## Security design

The trust boundaries are provider credentials and API responses, Terraform
plans, inventories, third-party Rego policies, and evidence bundles. A cloud
resource being audited is not trusted simply because its provider returned it.

| Principle | Application and evidence |
|---|---|
| Economy of mechanism | Common rules use typed observations; provider-specific collection stays separate from assessment. See [ADR-0001](docs/adr/0001-regles-communes-providers-collecteurs.md). |
| Fail-safe defaults | Missing observations do not establish compliance. Exceptions produce `exempted`, not `pass`. See [assessment](internal/assess/) and [exceptions](internal/exempt/). |
| Complete mediation | Required observations are checked before asserting a control's `pass`; bundle verification rechecks digests and consistency rather than trusting a stored verdict. See [ADR-0018](docs/adr/0018-ce-quun-bundle-prouve-de-lui-meme.md). |
| Open design | Rules, provider contracts, assessment logic and verification are public; the design does not rely on hidden algorithms. |
| Separation of privilege | A bundle's internal consistency and the signer's identity are distinct checks. A digest inside the bundle is not an external trust anchor. |
| Least privilege | Operators should use the provider's documented minimum permissions and a restricted kubeconfig. Rights come from the supplied identity; Pépin does not reduce them automatically. Some provider paths have documented permission limitations. |
| Least common mechanism | Provider credential resolution is separate from normalized inventory projection. Regression tests exercise credential-file authentication without exporting the credentials into inventory. See [credential-projection tests](internal/genprovider/genprovider_test.go). |
| Psychological acceptability | Typed results distinguish failure, unsupported measurement and exceptions; errors explain why evaluation or verification was refused. See the [CLI reference](docs/reference/cli.md). |

### Common errors and mitigations

| Risk | Mitigation and boundary |
|---|---|
| Credential disclosure | Native provider configuration or environment variables, declared-field projection, and secret-scanning CI. Reports can still contain sensitive tenant data; protect them separately. |
| Policy-driven exfiltration | User Rego policies are untrusted code. A local HTTP witness tests that they cannot reach the network, including remote JSON-schema references. See [ADR-0021](docs/adr/0021-une-regle-chargee-a-lexecution-est-du-code-non-fiable.md) and its [regression test](cmd/policy_network_test.go). |
| Malformed input or fabricated evidence | Parsers and policy loaders validate input; absent observations cannot be invented. Parser fuzzing and verification-tampering tests exercise rejection paths. See [campaigns](.github/workflows/campaigns.yml) and [verification tests](internal/assess/bundle_test.go). |
| Excessive input surface | Provider descriptors admit known authentication types and validate their shape rather than treating arbitrary input as a valid configuration. See [descriptor validation](internal/genprovider/validate.go). |
| Untrusted supply chain | Pinned actions and tools, dependency analysis, signed releases and provenance. See [ADR-0016](docs/adr/0016-chaine-dapprovisionnement-verifiable.md). |
| Excessive cloud permissions | Use the minimum permissions in the provider documentation; never assume that a scanner makes a privileged credential read-only. CI holds no cloud credentials. See [ADR-0012](docs/adr/0012-aucun-identifiant-en-ci.md). |

This describes the safeguards and their limits, not an attestation of a
developer's knowledge. The OpenSSF `know_secure_design` and `know_common_errors`
answers require a primary developer's own confirmation.

### Cryptography

Pépin uses cryptography; it is not appropriate to mark every cryptographic
criterion N/A merely because this is not a cryptographic library.

- SHA-256 digests use Go's `crypto/sha256`. EXO2 request authentication calls
  `crypto/hmac` with SHA-256; SigV4 calls the AWS SDK signer. Pépin implements
  the request format, not a new hash or MAC algorithm. See [authentication](internal/collect/auth.go).
- The [Kubernetes client](internal/kubeauth/kubeauth.go) verifies its configured
  CA and requires TLS 1.2 or later. Its TLS key agreement and randomness are
  provided by Go; Pépin does not supply a custom random generator.
- Bundle signatures are verified separately from their SHA-256 digests.
  Unsigned integrity checking is not proof of authorship; use the signature
  requirement when the caller needs authenticated evidence.
- Pépin does not authenticate external users with a local password database.
  Provider credentials are outbound credentials, not password hashes.
- **Key-strength limitation:** Pépin does not currently enforce a minimum
  bit length on operator-supplied TLS client keys. Operators should provide RSA
  keys of at least 2048 bits or elliptic-curve keys of at least 224 bits and
  follow their provider's credential policy. This recommendation is not an
  implemented key-strength check, and does not by itself establish the
  OpenSSF `crypto_keylength` criterion.

## Usage good practices

- Credentials pass only through the environment or the provider's native
  configuration, never as a command-line argument.
- Check an evidence bundle's integrity with `pepin verify`, and its **signature**
  with `pepin verify --pubkey <key>` (cosign sealing is the operator's identity).
- Pin versions; run only binaries whose provenance you verify.
