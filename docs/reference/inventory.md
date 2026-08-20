> 🇬🇧 English · [🇫🇷 Français](inventory.fr.md)

# The normalized inventory — an internal contract

Everything passes through the normalized inventory. Collectors project into it, rules evaluate
on it, the assessment derives from it, the evidence bundle seals it. Anything that consumes
Pépin beyond its CLI consumes this shape.

As long as it stayed implicit, every new use froze it a little more **by accident**, and a
change to the model would break a consumer in silence. So it is named, versioned and frozen,
with the same regard as the CLI surface.

<!-- pepin:gen inventory-format -->
```text
pepin-inventory/v2
```
<!-- /pepin:gen inventory-format -->

The version travels with every evidence bundle, in `manifest.inventory_schema`. A consumer
that meets a version it does not know must stop rather than guess.

## The envelope

```json
{
  "provider": "scaleway",
  "evaluated_at": "2026-08-19T10:11:12Z",
  "resources": [ … ],
  "collection": { "units": [ … ], "unmapped": [ … ] }
}
```

- `provider` — the identifier of the scanned provider.
- `resources` — the array of normalized resources, possibly empty.
- `evaluated_at` — the single instant of evaluation, RFC3339 UTC, added by `scan`. Time-sensitive
  rules anchor on it rather than on the clock, so replaying a sealed `input.json` yields the
  same verdict. It is **never** overwritten on a replay.
- `collection` — the state of what the collection could read, present when Pépin **measured**
  the inventory (live collection, Terraform plan) and absent when it **received** it (a
  third-party export). See the next section.

## The collection state

```json
{
  "units": [
    { "unit": "compute_instance", "types": ["compute_instance"], "attempted": true, "complete": true },
    { "unit": "iam_policy_inline", "types": ["iam_policy"], "attempted": true, "complete": false,
      "error": "permission_denied", "detail": "HTTP 403 · POST https://api…/ReadUserPolicies · AccessDenied" }
  ],
  "unmapped": [ { "type": "outscale_public_ip", "count": 2 } ]
}
```

A **unit** is one endpoint, or one chain of endpoints, feeding one or more normalized resource
types. `complete` is true when the unit returned everything the API had to return: a unit that
returned zero resources without error is complete — "there is nothing" is a measurement — while
a unit that returned a hundred resources out of a thousand before a `403` is not.

`error` is a stable class, not a message: `permission_denied`, `not_found`, `rate_limited`,
`timeout`, `truncated`, `unreadable`, `unavailable`. A pipeline must be able to tell "the
scanning account cannot see this" (fix the account policy) from "the service did not answer"
(retry). `detail` carries the provider's own response, untranslated — it is data, not Pépin
prose.

Every control that reads a type fed by an incomplete unit becomes `not-evaluated`, with that
unit named as the reason, and the scan does not return `0`. That decision belongs to the
assessment, never to a rule: a per-rule guard is a guard someone forgets to write on the
fiftieth rule.

`unmapped` lists resource types the source carried that no spec projects. It is **not**
incompleteness and it does not gate: no control reads those types, so no verdict depends on
them. It is there so that "Pépin saw ten resources and can read six of them" is never silent.

## The resource

```json
{
  "provider": "scaleway",
  "type": "security_group_rule",
  "id": "sg-bastion",
  "name": "sg-bastion",
  "region": "fr-par",
  "attributes": { "protocol": "tcp", "port_from": 22 },
  "provenance": { "protocol": { "origin": "api", "source": "GET https://api.scaleway.com/…", "path": "protocol", "observed": true } }
}
```

- `provider`, `type`, `id`, `name`, `attributes` are always present.
- `region` and `provenance` are present when they carry something.
- `attributes` is a **flat** map from attribute name to JSON value. This is what the Rego rules
  read, and it is why one rule can be common to every cloud.
- `provenance` is indexed by the **same** attribute names, never nested inside a value. See
  [the assessment model](../concepts/assessment-model.md) for what a `derived` origin means.

## What is guaranteed

- An attribute name is snake_case and **provider-agnostic**. The native vocabulary lives in the
  descriptor's projection, never in the model.
- A resource type is snake_case, singular, and named after its service family
  (`compute_instance`, `security_group_rule`, `object_storage_bucket`, …).
- **An absent attribute is never forced to a value.** "not collected" and "collected as false"
  do not get confused. Every trust guarantee in Pépin rests on that one invariant.
- A `provenance` key may exist without the matching attribute being in `attributes`: it means
  the field was looked for at that path and the source did not carry it.

## What is not guaranteed

- **The order** of resources or of attributes. Nothing fixes it, and relying on it would break
  at the first pagination change.
- **The presence** of a given attribute on a given resource. It depends on the provider, on the
  rights of the token, and on the source — a Terraform plan knows nothing of the effective
  state. That question is exactly what `provenance` answers, per attribute.
- **The completeness** of the inventory. A scan measures what its credentials reach, never
  "the whole tenant".
- **The values** themselves. They mirror the provider's native contract, which can change
  without Pépin deciding anything.

## The vocabulary

Every resource type and the common attributes it can carry, derived from the loaded provider
descriptors and from the Go collectors — never a hand-kept list beside the code.

<!-- pepin:gen inventory-types -->
| Resource type read | Common attributes |
|---|---|
| `access_key` | `access_key_id` `expiration_date` `owner_user` `scope` `state` |
| `api_access_policy` | `id` `max_access_key_expiration_seconds` `require_trusted_env` |
| `api_access_rule` | `api_access_rule_id` `ca_ids` `cns` `ip_ranges` |
| `api_access_summary` | `id` `rule_count` |
| `blockstorage_snapshot` | `creation_date` `global_permission` `snapshot_id` `volume_id` |
| `blockstorage_volume` | `encrypted` `state` `tags` `volume_id` |
| `compute_image` | `image_id` `public` `state` `tags` |
| `compute_instance` | `deletion_protection` `nic_public_ips` `public_ip` `security_group_ids` `state` `tags` `user_data` `vm_id` |
| `governance_provider` | `capital_control` `eu_established` `extraterritorial_exposure` `jurisdiction` `secnumcloud` |
| `iam_policy` | `manages_iam` `owner_group` `owner_user` `policy_id` `policy_name` `scope` `statements` |
| `iam_role` | `admin_privileges` `editable` `manages_iam` `max_session_ttl` `name` `policy_has_expiration` `role_id` `source_ip_restricted` |
| `iam_user` | `mfa_enabled` `user_id` `username` |
| `k8s_cluster_role_binding` | `name` `role_ref` `subjects` |
| `k8s_crd` | `name` |
| `k8s_namespace` | `labels` `name` |
| `k8s_network_policy` | `name` `namespace` |
| `kubernetes_cluster` | `admin_whitelist` `audit_enabled` `auto_upgrade` `control_plane_multi_az` `deletion_protection` `name` `version` |
| `load_balancer` | `access_log` `listeners` `load_balancer_name` `load_balancer_type` `tags` |
| `managed_database` | `database_id` `disable_backup` `encryption_at_rest` `ip_filter` |
| `network` | `cidr` `description` `name` `network_id` `state` `tags` |
| `network_peering` | `accepter_account` `peering_id` `source_account` `state` |
| `object_storage_bucket` | `acl` `acl_grants` `default_encryption_enabled` `kms_key_id` `name` `object_lock_enabled` `policy_public` `public_via_acl` `sse_kms_enabled` `tags` `versioning` |
| `security_group` | `inbound_default_policy` `security_group_id` |
| `security_group_rule` | `action` `cidrs` `description` `direction` `port_from` `port_to` `protocol` `security_group_id` `security_group_name` |
| `subnet` | `map_public_ip_on_launch` `network_id` `state` `subnet_id` `tags` |
<!-- /pepin:gen inventory-types -->

An entry here means "this attribute exists in the model, on this type". It does **not** mean it
is present on a given resource of a given scan: that question belongs to the provenance.

## How it moves

The shape and this vocabulary are frozen in `cmd/testdata/frozen/inventory.json`. Adding an
attribute to a descriptor turns the freeze red, and the change then costs three things: run
`mise run frozen-update`, bump `model.InventoryFormat`, write the CHANGELOG line. That cost is
deliberate — it is what "the inventory is no longer an implementation detail" means in practice.

## Where to go next

- [Output formats](output-formats.md) — the parsable documents derived from this inventory.
- [Evidence bundles](../guides/evidence-bundles.md) — where the inventory is sealed, with its
  schema version.
- [The assessment model](../concepts/assessment-model.md) — how a status is derived from it.

## How this page stays true

The format string, the vocabulary and the shape come from the code through the generated
regions above. Regenerate with `mise run gen-docs`; `TestGeneratedDocsAreUpToDate` fails if the
committed page and the code disagree.
