> 🇬🇧 English · [🇫🇷 Français](terraform-vs-live.fr.md)

# Terraform plan vs live scan

Pépin evaluates one normalized model, and that model can be fed from three sources: a Terraform
plan, a live collection through the provider API, or a normalized inventory export. The rules
never know which one they are reading — [that is the architecture](../coverage.md). What
changes is **what the source can see**, and the difference is not cosmetic: the same control,
on the same resource, can conclude on one side and refuse to conclude on the other.

Neither source is strictly better. A plan audits what does not exist yet; a live scan audits
what is actually running, including everything nobody wrote in Terraform.

## The two sources, side by side

| | Terraform plan | Live scan |
|---|---|---|
| Credentials | none | yes, the provider's own environment variables |
| What is audited | the state the code declares | the effective configuration |
| Resources created outside the code | invisible | seen |
| Drift between code and reality | invisible by construction | this is what it measures |
| Before deployment | yes, on a pull request | no |
| Runtime data (attributes computed on apply) | partly unknown | available |
| Cost, blast radius | nothing provisioned, nothing billed | reads a real tenant |
| Typical use | the gate on a merge request | the scheduled posture check |

```bash
pepin scan scaleway --terraform plan.json     # no account, nothing created
pepin scan scaleway --live --region fr-par    # reads the real tenant
```

The two flags are mutually exclusive, and Pépin refuses both rather than picking one
([CLI reference](../reference/cli.md#one-source-and-exactly-one)).

## Divergence 1 — an attribute that is unknown at plan time

This is not a hypothetical. It is the false positive fixed in **v0.1.1**, and it is reproducible
from the plan committed in this repository.

The plan declares an instance and a security group. The instance's group is created by the same
plan, so its identifier does not exist yet: Terraform marks it `unknown after apply` and
`planned_values` simply has no such key.

<!-- pepin:gen drift-plan-unknown -->
```json
{
  "address": "scaleway_instance_server.web",
  "change": {
    "after": {},
    "after_unknown": {
      "id": true,
      "public_ips": true,
      "security_group_id": true
    }
  },
  "type": "scaleway_instance_server"
}
```
<!-- /pepin:gen drift-plan-unknown -->

Three keys, all unknown: the identifier, the public addresses, and the security group. The
control that reads instance filtering therefore cannot decide, and says so rather than guessing:

<!-- pepin:gen drift-plan-status -->
```json
{
  "control": "compute_instance_has_security_group",
  "evidence": {
    "attribute": "security_group_ids",
    "observed": "attribute \"security_group_ids\" not collected on the resources of type \"compute_instance\" (capability guard)",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "security_group_ids=terraform-plan:scaleway_instance_server observed=0/1"
  },
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-CMP-1"
    },
    {
      "framework": "cis-v8",
      "id": "4.4"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.20"
    },
    {
      "framework": "iso-27017",
      "id": "CLD.9.5.2"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "13.2"
    }
  ],
  "severity": "critical",
  "status": "not-evaluated",
  "subject": "scaleway",
  "title": "Instance without network filtering"
}
```
<!-- /pepin:gen drift-plan-status -->

Before v0.1.1, a collection transform fabricated an empty list for that absent key, the rule's
capability guard was satisfied by that empty list, and the scan reported a `CRITICAL` "VM
without a security group" on a VM that was, in fact, properly attached. The transform now runs
only when the source key exists: **absent means the source does not expose the information;
present-and-empty is information.**

On a source where the attribute is collected, the same control concludes. Here is the same
instance in an inventory shaped as the live collector normalizes it — `vm_id` from `Server.ID`,
`security_group_ids` from the server's security group, per `providers/scaleway.yaml`:

<!-- pepin:gen drift-live-fixture -->
```json
{
  "provider": "scaleway",
  "resources": [
    {
      "provider": "scaleway",
      "type": "compute_instance",
      "id": "3f2b1c00-0000-4a00-9000-000000000001",
      "name": "web",
      "region": "fr-par",
      "attributes": {
        "vm_id": "3f2b1c00-0000-4a00-9000-000000000001",
        "state": "running",
        "security_group_ids": ["b1a2c3d4-0000-4a00-9000-000000000002"],
        "tags": [{"key": "env", "value": "prod"}]
      }
    }
  ]
}
```
<!-- /pepin:gen drift-live-fixture -->

<!-- pepin:gen drift-live-status -->
```json
{
  "control": "compute_instance_has_security_group",
  "evidence": {
    "observed": "no deviation detected on the collected resources of type \"compute_instance\" (contract verified)",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "export"
  },
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-CMP-1"
    },
    {
      "framework": "cis-v8",
      "id": "4.4"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.20"
    },
    {
      "framework": "iso-27017",
      "id": "CLD.9.5.2"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "13.2"
    }
  ],
  "severity": "critical",
  "status": "pass",
  "subject": "scaleway",
  "title": "Instance without network filtering"
}
```
<!-- /pepin:gen drift-live-status -->

> **No live account was used to produce this page.** The inventory above is a fixture written
> in the shape the collector produces, and the scan of it is real. A live scan against a
> Scaleway tenant would exercise the same code path from the same attribute; nothing here
> claims to have run it.

`not-evaluated` → `pass` on the same control, same tenant, same day. The verdict did not change
because the configuration changed; it changed because the source could see one more attribute.
That is the whole point of this page.

## Divergence 2 — a boolean the plan renders as a string

The second divergence is quieter, because it does not change a status: it changes a **type**.

Outscale's launch permission for a machine image is a boolean in the API
(`PermissionsToLaunch.GlobalPermission`), and a *string* in a Terraform plan
(`permission_additions[].global_permission`). Read the committed plan:

<!-- pepin:gen drift-bool-plan -->
```json
{
  "address": "outscale_image_launch_permission.public_omi",
  "change": {
    "after": {
      "image_id": "ami-12345678",
      "permission_additions": [
        {
          "account_ids": null,
          "global_permission": "true"
        }
      ]
    },
    "after_unknown": {
      "permission_additions": [
        {}
      ]
    }
  },
  "type": "outscale_image_launch_permission"
}
```
<!-- /pepin:gen drift-bool-plan -->

`"true"` — with quotes. A rule written as `attributes.public == true` would silently miss every
publicly shared image in a plan, while catching them live. The common rules therefore go through
a shared helper (`truthy` in `internal/commonrules/rules/lib.rego`) that accepts the boolean
`true` and the string `"true"`, case-insensitively. The deviation is found:

<!-- pepin:gen drift-bool-finding -->
```json
{
  "code": "CLD-STO-2",
  "labels": {
    "category": "security",
    "check": "compute_image_not_public",
    "provider": "outscale",
    "tf_file": "main.tf",
    "tf_line": "32"
  },
  "message": "Machine image \"ami-12345678\" shared publicly (anyone is allowed to launch it).",
  "remediation": "Remove the image's public sharing; reserve it for legitimate accounts.",
  "severity": "high",
  "subject": "ami-12345678",
  "title": "Machine image shared publicly"
}
```
<!-- /pepin:gen drift-bool-finding -->

The lesson generalizes: a plan is a *rendering* of the configuration by a Terraform provider
schema, not the API's own payload. Types can differ, names can differ, and computed attributes
can be missing entirely. That is why normalization lives in the collector and the mapper, per
provider and per source, and never in the rules.

## Controls that only one source can observe

This table is computed from the provider descriptors and the reference: for these pairs, one
source produces the resource type and its deciding attribute, and the other does not.

<!-- pepin:gen single-source -->
| Control | Provider | Observable only through | Reason |
|---|---|---|---|
| `blockstorage_snapshot_not_public` | outscale | live | this source produces no resource of type "blockstorage_snapshot" |
| `blockstorage_volume_snapshots_exist` | outscale | live | this source produces no resource of type "blockstorage_volume" |
| `compute_instance_deletion_protection` | outscale | live | deciding attribute "deletion_protection" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `compute_instance_no_secrets_in_user_data` | scaleway | terraform | deciding attribute "user_data" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `compute_instance_public_ip_with_open_securitygroup` | scaleway | live | deciding attribute "public_ip" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `database_backup_enabled` | scaleway | terraform | this source produces no resource of type "managed_database" |
| `database_encryption_at_rest_enabled` | scaleway | terraform | this source produces no resource of type "managed_database" |
| `database_service_not_open_to_internet` | scaleway | terraform | this source produces no resource of type "managed_database" |
| `governance_resource_region_in_eu` | outscale | live | deciding attribute "region" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `iam_accesskey_expiration_set` | outscale | live | this source produces no resource of type "access_key" |
| `iam_account_mfa_enforced` | outscale | live | this source produces no resource of type "api_access_policy" |
| `iam_apiaccesspolicy_max_key_expiration` | outscale | live | this source produces no resource of type "api_access_policy" |
| `iam_apiaccessrule_defined` | outscale | live | this source produces no resource of type "api_access_summary" |
| `iam_apiaccessrule_no_public_cidr` | outscale | live | this source produces no resource of type "api_access_rule" |
| `iam_no_root_access_key` | outscale | live | this source produces no resource of type "access_key" |
| `iam_policy_no_privilege_escalation` | scaleway | terraform | this source produces no resource of type "iam_policy" |
| `iam_user_mfa_enabled` | exoscale | live | this source produces no resource of type "iam_user" |
| `iam_user_mfa_enabled` | scaleway | live | this source produces no resource of type "iam_user" |
| `kubernetes_cluster_auto_upgrade_enabled` | outscale | live | this source produces no resource of type "kubernetes_cluster" |
| `kubernetes_cluster_control_plane_highly_available` | outscale | live | this source produces no resource of type "kubernetes_cluster" |
| `kubernetes_cluster_deletion_protection` | outscale | live | this source produces no resource of type "kubernetes_cluster" |
| `kubernetes_cluster_not_publicly_accessible` | outscale | live | this source produces no resource of type "kubernetes_cluster" |
| `loadbalancer_logging_enabled` | outscale | live | this source produces no resource of type "load_balancer" |
| `loadbalancer_ssl_listeners` | outscale | live | this source produces no resource of type "load_balancer" |
| `network_peering_cross_organization` | outscale | live | this source produces no resource of type "network_peering" |
| `network_securitygroup_default_deny` | scaleway | terraform | this source produces no resource of type "security_group" |
| `network_securitygroup_default_restrict_traffic` | outscale | live | deciding attribute "security_group_name" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `objectstorage_bucket_default_encryption` | outscale | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_kms_encryption` | scaleway | live | deciding attribute "sse_kms_enabled" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `objectstorage_bucket_object_lock_enabled` | exoscale | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_object_lock_enabled` | outscale | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_public_access` | exoscale | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_public_access` | outscale | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_versioning_enabled` | exoscale | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_versioning_enabled` | outscale | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_versioning_enabled` | scaleway | live | deciding attribute "versioning" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
<!-- /pepin:gen single-source -->

Per-provider totals, and the reason for every non-✅ cell, are in the
[coverage matrix](../coverage.md).

## Comparing two scans honestly

Two reports of the same tenant, one from a plan and one from live, are **not** interchangeable.
Before reading a difference as drift, check the three things that can explain it without any
configuration having changed:

**1. The source.** Every assessment carries it, and it is sealed in the evidence bundle. The
two documents behind this page declare it like this:

<!-- pepin:gen drift-source-provenance -->
```json
{"run": {"source": "terraform-plan"}}
{"run": {"source": "export"}}
```
<!-- /pepin:gen drift-source-provenance -->

**2. Coverage.** A control that is `not-evaluated` on one side and `pass` on the other may be a
coverage difference, not a posture difference. The evidence field says which: it names the
attribute or the resource type it lacked.

**3. The language.** Codes, statuses and severities are stable across languages; titles,
messages and evidence are not. A pipeline that diffs report *text* must pin `PEPIN_LANG`
([Output formats](../reference/output-formats.md#formats-and-language)).

## Where a finding comes from

A finding from a Terraform plan carries the **origin** of the resource that produced it: the
module, and — when the `.tf` sources sit beside the plan — the file and the line of the
`resource` block. `--format json` puts it in `labels.tf_file`, `labels.tf_line` and
`labels.tf_module`; SARIF puts it in the result's `physicalLocation`, which is what makes a
forge annotate the guilty line rather than the plan file.

```json
"labels": {
  "check": "objectstorage_bucket_public_access",
  "provider": "scaleway",
  "tf_file": "main.tf",
  "tf_line": "81",
  "tf_module": "module.storage"
}
```

Three things this deliberately does **not** do:

- **It never invents.** `terraform show -json` carries no file and no line — verified in
  Terraform's own source, where the configuration representation of a resource holds `address`,
  `type`, `name`, `expressions` and nothing about the document. The file and the line are read
  from the `.tf` files next to the plan; when they are not there, or when the same
  `resource "type" "name"` header appears twice in the searched directory, the origin is simply
  absent. A wrong line sends someone to fix the wrong place, and it is believed.
- **It does not apply to a live collection.** There, the information does not exist. No label is
  set, nothing fails, and the parsable formats simply do not carry the fields.
- **It does not follow a remote module.** A module whose `source` is a registry or a git
  repository has no directory in the working tree: its resources keep their `module`, and have
  neither file nor line. A partial origin is rendered partial.

## The workflow that uses both

- **On a pull request — the plan.** No credential, nothing provisioned, and the feedback lands
  before the resource exists. Gate on exit code 1
  ([GitHub Actions](../guides/github-actions.md), [GitLab CI](../guides/gitlab-ci.md)).
- **On a schedule — live.** Once a day or once a week, against the real tenant, with a
  read-only key. This is what catches what nobody wrote in Terraform, what a console click
  changed, and what an expired credential hides — the last one showing up as exit code 3
  rather than as a false green ([Exit codes](../reference/exit-codes.md)).
- **Seal the live run.** A live scan is the one worth `--seal`: it observed a real tenant at a
  real instant ([Evidence bundles](../guides/evidence-bundles.md)).

A plan-only pipeline is blind to drift. A live-only pipeline finds problems after they are
deployed. The two answer different questions, and a serious posture programme asks both.

## Where to go next

- [The assessment model](assessment-model.md) — why `not-evaluated` is a first-class result.
- [Coverage matrix](../coverage.md) — per control, per provider, per source.
- [Known limitations](../known-limitations.md) — what neither source can see.
- [Scope and non-goals](scope.md) — what a Pépin report is not.

## How this page stays true

The plan excerpts are read from the fixtures committed in `examples/`, and the statuses are the
output of real scans of those fixtures, captured by `internal/docgen`. If the guard on that
control were removed, or if the plan stopped marking the attribute unknown, this page would
stop matching what the binary produces and `TestGeneratedDocsAreUpToDate` would fail.
