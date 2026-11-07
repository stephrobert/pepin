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
- [The shared TLS policy](internal/tlspolicy/tlspolicy.go) applies to Kubernetes,
  REST collectors (including EIM and OKS), and S3. It requires TLS 1.2 or later,
  TLS 1.2 AEAD/ECDHE suites, RSA keys of at least 2048 bits, elliptic-curve keys
  of at least 224 bits, or Ed25519. Go supplies TLS 1.3, key agreement and randomness.
  Server names, expiry and trust are still verified by Go. Every certificate in
  at least one verified server chain must meet the key policy, including its CA.
  Every certificate in an operator-supplied client chain is checked at load time.
  Weak keys fail explicitly; collection failures remain recorded, not successful
  empty inventories. Private clusters using RSA 1024 must rotate their keys.
  S3 keeps SDK custom trust roots (`AWS_CA_BUNDLE` or a profile's `ca_bundle`)
  and proxy configuration; custom CAs do not bypass the key-strength policy.
- Bundle signatures are verified separately from their SHA-256 digests.
  Unsigned integrity checking is not proof of authorship; use the signature
  requirement when the caller needs authenticated evidence.
- The [shared key policy](internal/cryptopolicy/keys.go) also rejects EXO2,
  SigV4 and S3 HMAC secrets shorter than 14 actual UTF-8 bytes (112 bits of
  material). Provider setup and direct signing/SDK paths enforce the guard,
  without an override. This is a key-size check, **not an entropy estimate**:
  use provider-issued secrets or cryptographically generated material.
- `verify --pubkey` requires a local PEM public key (PKIX or RSA PKCS#1).
  It applies the same RSA/EC/Ed25519 thresholds before invoking Cosign.
  Cosign receives a private-permission temporary snapshot of the validated key,
  not the original path that could be replaced after validation. Export KMS
  public keys to PEM first; remote key references are not accepted unchecked.
  Cancellation and deadlines remain context errors, including after Cosign
  starts; an interrupted verification is not reported as an invalid signature.
- Pépin does not authenticate external users with a local password database.
  Provider credentials are outbound credentials, not password hashes.
- From v0.5.1, release signatures use Rekor v2. Cosign authenticates the signing
  configuration through Sigstore TUF; the shared helper removes v1 fallback.
  The workflow verifies the checksum signature and image digest against its
  exact GitHub workflow identity and issuer before creating the GitHub Release.
  No transparency or trust check is disabled. Cosign 3.1.3 also verifies older
  v1 release signatures using its standard TUF trust.

#### Cryptographic scope and enforced sizes

| Mechanism | Parameters and ownership | Enforcement / limit |
|---|---|---|
| Inventory, policy and bundle digests | SHA-256, chosen by Pépin | 256-bit digest, standard library |
| EXO2, SigV4 and S3 request MACs | HMAC-SHA256; API secret issued by the provider or local S3 operator | At least 14 input bytes, checked before signing; SHA-256 digest |
| Kubernetes, REST, EIM, OKS and S3 TLS | Go TLS with the shared policy above | Client keys and verified server chains checked; RSA 1024 refused |
| Bundle signatures | Operator PEM public key and Cosign | Same RSA/EC/Ed25519 key-size guard, validated snapshot, then actual signature verification |
| Release signatures | Pinned Cosign v3.1.3, keyless default | Ephemeral ECDSA P-256/SHA-256; CI guards against key/algorithm substitution |
| Bearer tokens and header credentials | Outbound authentication identifiers, not keys selected for a cryptographic primitive | Not generated by Pépin; their length is not used as an entropy estimate |
| HTTP endpoints, including local MinIO | Existing non-TLS transport | Still accepted for compatibility; no TLS confidentiality or key check |

[Tests](internal/tlspolicy/collectors_test.go) exercise real local TLS connections
for REST and S3, alongside [Kubernetes tests](internal/kubeauth/kubeauth_test.go):
RSA 1024 is refused, RSA 2048 and P-256 remain accepted, and a refused REST
connection is recorded as incomplete before any authenticated HTTP request arrives.
HMAC boundary tests refuse 13 bytes and accept 14. Signature tests reject RSA
1024 before Cosign, and exercise the actual binary with RSA 2048/P-256 signatures
and tampered blobs. The native test is opt-in:

```bash
PEPIN_COSIGN_INTEGRATION=1 mise exec -- go test ./cmd -run '^TestNativeCosignKeyStrengthAndSignature$' -v -count=1
```

It uses generated test keys, never cloud credentials; Cosign may contact its
public transparency service. Default suites test the guards without network access.
The pinned release default is verified against
[Cosign's key generation](https://github.com/sigstore/cosign/blob/v3.1.3/pkg/cosign/keys.go).
These checks establish key sizes, not a provider's random-generation quality.
The OpenSSF `crypto_keylength` proposal is now **Met**; publishing and saving the
questionnaire are distinct steps.

## Usage good practices

- Credentials pass only through the environment or the provider's native
  configuration, never as a command-line argument.
- Check an evidence bundle's integrity with `pepin verify`, and its **signature**
  with `pepin verify --pubkey <key>` (cosign sealing is the operator's identity).
- Pin versions; run only binaries whose provenance you verify.
