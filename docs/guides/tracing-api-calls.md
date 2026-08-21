> 🇬🇧 English · [🇫🇷 Français](tracing-api-calls.fr.md)

# Tracing a collector's real API calls

A provider descriptor **declares** endpoints. Nothing, on its own, proves the collector
**emits** them. That gap has a name in this project: it is the shape of the inline-EIM
incident, where the rule was right, the data never arrived, and no Rego test could see it.

This page is how you close it by **measuring** rather than reading.

## What a trace establishes — and what it does not

Read this table before you quote a trace as evidence. Everything below the line stays owed to
a real scan against a real tenant, which this repository never performs: it holds no cloud
credentials, by construction.

| A recording against the local emulator establishes | It does **not** establish |
|---|---|
| the endpoints the collector actually emits, against those `providers/<name>.yaml` declares | the field names and types of the provider's native contract |
| that a parent→child join fires, on an id read from the parent response | the provider's real pagination bounds |
| the pagination parameters actually put on the wire | its rate-limiting behaviour |
| the class a collection failure is filed under (`not_found`, `unavailable`, …) | that the provider answers `403` rather than `200` with an error body |
| that no control returns `pass` from an incomplete unit | anything at all about a real tenant's inventory |

**An emulator proves what Pépin does, not what the cloud answers.** Never let the two be
confused: that is precisely the false confidence this project exists to refuse.

One consequence deserves to be spelled out. The emulator **accepts every credential** and
offers no fault injection — measured: no auth header returns `200`, a junk token returns `200`,
and no fault route exists. It therefore **cannot produce a `403`**. The classification of a
refusal is exercised elsewhere, by `internal/collect/status_test.go` against a real socket that
really refuses; what remains unmeasured is whether a given provider refuses with that status.

## The procedure

```bash
mise run build
PROVIDER=scaleway mise run trace          # or: ./scripts/trace-collector.sh scaleway .trace
```

That is the whole of it. It needs [feint](https://github.com/stephrobert/feint) 0.10.0 or later
in `PATH` and `unshare` from util-linux. It needs **no cloud credentials and touches no real
API**.

### Why the chain has two proxy stages

No collection `base_url` is redirectable — they are compiled in, and only `--s3-endpoint` is an
option. What *is* true, and what makes all of this possible, is that the collection client
installs no `Transport`: it inherits `http.DefaultTransport`, **so it honours `HTTPS_PROXY`**.

```
Pépin ──HTTPS_PROXY, CONNECT──▶ upstream proxy (--forward, records)
                                     │ redials the host the client asked for
                                     ▼   (resolved to 127.0.0.1 by /etc/hosts)
                                downstream proxy (--intercept, terminates TLS)
                                     │ --upstream
                                     ▼
                                feint serve (the emulator)
```

feint 0.10.0 **refuses `--forward` and `--upstream` together**, and it is right to: `--forward`
sends each request to the host the *client* asked for, `--upstream` sends every request to the
host *you* chose. The second stage is what makes "the host the client asked for" be the
emulator rather than the real cloud.

**Nothing of yours is modified.** The whole chain runs inside a namespace (user + mount + net):
the `/etc/hosts` that is replaced is that namespace's, never yours, and the bound port 443 lives
in a private network stack that disappears with the last process. `--vm off` forbids the
emulator from starting any container with your privileges.

**No line of Pépin is modified either** — and that is the point, not a convenience. A
configurable collection endpoint was identified by the delivery audit as a way to send a
tenant's secret key to an arbitrary host. Every collection request carries that key in a header.
The chain above adds no such surface.

### Object storage: the cheapest demonstration

`--s3-endpoint` is the one endpoint that already redirects, so the S3 collector needs neither
`CONNECT` nor TLS interception:

```bash
feint proxy --addr 127.0.0.1:4601 --upstream http://127.0.0.1:4599 --record s3.jsonl
pepin scan scaleway --live --s3-endpoint http://127.0.0.1:4601
```

The emulator serves no object-storage surface, so `ListBuckets` comes back `404`. That is still
a measurement, and a useful one: it is the first time the S3 branch of `collect.Classify` — the
one that reads the AWS SDK's error through an anonymous interface rather than a typed error —
has been exercised against a real HTTP response rather than a constructed one.

## Reading a recording

The transcript is JSON Lines, one object per exchange, with the upstream operation named:

```json
{"seq":1,"method":"GET","path":"/iam/v1alpha1/api-keys","host":"api.scaleway.com",
 "status":501,"mounted":false,
 "req":{"headers":{"X-Auth-Token":"REDACTED"}},
 "res":{"body":{"type":"not_emulated"}}}
```

Look for these, in this order of value:

1. **Endpoints emitted versus declared.** An endpoint declared and never called is a control
   that measures nothing. A **child** endpoint is only reached when its parent returned at
   least one item — so a silent child is not automatically a defect, and not automatically
   fine either. It has to be read.
2. **`"mounted": false`** — the emulator has no route for it. Against a real provider this is
   where an endpoint that has moved would show.
3. **The status, and the class it produced.** Cross-read with the `collection` key of
   `--format json`: every unit says what happened to it.
4. **Repeated calls.** Two declared resources sharing one source endpoint call it twice per
   scan. Not a correctness defect; a cost and a rate-limit fact.

## Committing a recording

> **No recording enters the repository without a value-by-value re-read.**

The proxy's redaction protects **headers**, and it is a **whitelist** — which is why it also
masks `X-Content-Type-Options`, a header no blacklist would have thought to name. The
asymmetry is deliberate: a name-based check answers "does this look like a secret", never "is
this certainly not one".

**Bodies are the measurement, so they are kept in full.** Against a real tenant they carry its
resource ids, its bucket names and its IP addresses. Partial sanitisation is the trap: the
delivery audit opened on a real instance UUID left in a fixture whose IP address *had* been
sanitised.

A recording committed to `internal/genprovider/testdata/transcripts/` carries a manifest
(`<provider>.yaml`) stating what it is, what it was recorded against, the path variables it was
recorded with, and — under `non_observes` — every declared endpoint the session did **not**
exercise, with the reason. Two gates keep it honest, and they fail in both directions:

- `TestTheRecordedCollectionStillHappens` replays the recording against today's collector.
  Fewer calls than the recording saw means a datum stopped arriving; more means the collector
  reaches an endpoint no session has ever observed.
- `TestEveryDeclaredEndpointIsObservedOrDeclaredUnobserved` keeps the `non_observes` ledger
  exact. A ledger that overstates the gap is as wrong as one that understates it.

The replay uses the **recorded** responses, never responses built from the spec it is testing.
A harness that answered "what the spec expects" would be measuring its own copy of the spec: a
wrong `items:` would make it serve the wrong array and it would stay green.

## Related

- [Terraform vs live](../concepts/terraform-vs-live.md) — what each source can conclude.
- [Known limitations](../known-limitations.md) — what stays unobservable, and who can lift it.
- `CONTRIBUTING.md` — the quality gates a change has to pass.
