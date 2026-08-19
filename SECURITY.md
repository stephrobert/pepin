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

## Usage good practices

- Credentials pass only through the environment or the provider's native
  configuration, never as a command-line argument.
- Check an evidence bundle's integrity with `pepin verify`, and its **signature**
  with `pepin verify --pubkey <key>` (cosign sealing is the operator's identity).
- Pin versions; run only binaries whose provenance you verify.
