> [🇬🇧 English](inventory.md) · 🇫🇷 Français

# L'inventaire normalisé : un contrat interne

Tout passe par l'inventaire normalisé. Les collecteurs y projettent, les règles s'y évaluent,
l'assessment en dérive, le bundle de preuve le scelle. Tout ce qui consomme Pépin au-delà de sa
CLI consomme cette forme.

Tant qu'elle restait implicite, chaque nouvel usage la figeait un peu plus **par accident**, et
une évolution du modèle cassait un consommateur en silence. Elle est donc nommée, versionnée et
gelée, avec les mêmes égards que la surface CLI.

<!-- pepin:gen inventory-format -->
```text
pepin-inventory/v4
```
<!-- /pepin:gen inventory-format -->

La version voyage avec chaque bundle de preuve, dans `manifest.inventory_schema`. Un
consommateur qui rencontre une version qu'il ne connaît pas doit s'arrêter plutôt que deviner.

## L'enveloppe

```json
{
  "provider": "scaleway",
  "evaluated_at": "2026-08-19T10:11:12Z",
  "resources": [ … ],
  "collection": { "units": [ … ], "unmapped": [ … ] }
}
```

- `provider` : l'identifiant du fournisseur scanné.
- `resources` : le tableau des ressources normalisées, éventuellement vide.
- `evaluated_at` : l'instant d'évaluation unique, RFC3339 UTC, posé par `scan`. Les règles
  sensibles au temps s'y ancrent plutôt qu'à l'horloge, si bien que rejouer un `input.json`
  scellé rend le même verdict. Il n'est **jamais** écrasé lors d'un rejeu.
- `collection` : l'état de ce que la collecte a pu lire. Présent quand Pépin a **mesuré**
  l'inventaire (collecte live, plan Terraform), absent quand il l'a **reçu** (export d'un
  tiers). Voir la section suivante.

## L'état de collecte

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

Une **unité** est un endpoint, ou une chaîne d'endpoints, qui alimente un ou plusieurs types de
ressources normalisés. `complete` est vrai quand l'unité a rendu tout ce que l'API avait à
rendre : une unité qui a rendu zéro ressource sans erreur est complète — « il n'y a rien » est
une mesure — alors qu'une unité qui a rendu cent ressources sur mille avant un `403` ne l'est
pas.

`error` est une classe stable, pas un message : `permission_denied`, `not_found`,
`rate_limited`, `timeout`, `truncated`, `unreadable`, `unavailable`. Un pipeline doit pouvoir
distinguer « le compte de scan ne voit pas cette surface » (à corriger sur la politique du
compte) de « le service n'a pas répondu » (à réessayer). `detail` porte la réponse du
fournisseur telle quelle, non traduite : c'est une donnée, pas de la prose de Pépin.

**Jusqu'où chaque classe est mesurée.** La *correspondance* l'est contre de vraies sockets :
`internal/collect/status_test.go` pilote un serveur qui refuse vraiment, expire vraiment et
tronque vraiment une page, et un `403` y ressort bien en `permission_denied`. Une session
enregistrée contre l'émulateur (`internal/genprovider/testdata/transcripts/`) y ajoute
`not_found` et `unavailable` observés sur le réseau, y compris à travers l'interface d'erreur du
SDK AWS sur le chemin du stockage objet. Ce qu'aucun contrôle de ce dépôt n'établit, c'est
quelle classe un **fournisseur donné** déclenchera : si Outscale répond `403` plutôt que `200`
accompagné d'un corps `Errors`, si un appel Scaleway limité rend `429` plutôt que `503`.
L'émulateur ne peut pas trancher non plus, puisqu'il accepte n'importe quel identifiant et ne
sait donc jamais refuser. Cela reste dû à un scan réel ; voir
[Limites connues](../known-limitations.fr.md) et
[Tracer les appels réels](../guides/tracing-api-calls.fr.md).

Tout contrôle qui lit un type alimenté par une unité incomplète devient `not-evaluated`, avec
cette unité nommée comme motif, et le scan ne rend pas `0`. Cette décision appartient à
l'assessment, jamais à une règle : une garde par règle est une garde qu'on oublie d'écrire à la
cinquantième.

`unmapped` liste les types de ressources que la source portait et qu'aucune spec ne projette.
Ce n'est **pas** une incomplétude et cela ne bloque aucune porte : aucun contrôle ne lit ces
types, donc aucun verdict n'en dépend. Cette liste existe pour que « Pépin a vu dix ressources
et sait en lire six » ne soit jamais silencieux.

## La ressource

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

- `provider`, `type`, `id`, `name`, `attributes` sont toujours présents.
- `region`, `provenance` et `source` sont présents quand ils portent quelque chose.
- `source` est l'**origine dans le code d'infrastructure** : `file`, `line` et `module`. Elle
  n'existe que pour la source Terraform, et seulement dans la mesure où elle se mesure — le
  module se lit dans l'adresse de la ressource, le fichier et la ligne se trouvent dans les
  sources `.tf` posées à côté du plan. Une collecte live n'en porte rien, et rien n'y est
  inventé. Voir
  [Plan Terraform et collecte live](../concepts/terraform-vs-live.fr.md#doù-vient-un-finding).
- `attributes` est une carte **plate** de nom d'attribut vers valeur JSON. C'est ce que lisent
  les règles Rego, et c'est pourquoi une même règle vaut pour tous les clouds.
- `provenance` est indexée par les **mêmes** noms d'attributs, jamais imbriquée dans une
  valeur. Voir [le modèle d'assessment](../concepts/assessment-model.fr.md) pour ce que
  signifie une origine `derived`.

## Ce qui est garanti

- Un nom d'attribut est en snake_case et **agnostique du fournisseur**. Le vocabulaire natif
  vit dans la projection du descripteur, jamais dans le modèle.
- Un type de ressource est en snake_case, au singulier, nommé d'après sa famille de service
  (`compute_instance`, `security_group_rule`, `object_storage_bucket`…).
- **Un attribut absent n'est jamais forcé à une valeur.** « Non collecté » et « collecté à
  faux » ne se confondent pas. Toutes les garanties de confiance de Pépin reposent sur cet
  invariant.
- Une clé de `provenance` peut exister sans que l'attribut correspondant soit dans
  `attributes` : cela signifie que le champ a été cherché à ce chemin et que la source ne le
  portait pas.

## Ce qui n'est pas garanti

- **L'ordre** des ressources ni celui des attributs. Rien ne le fixe, et s'en servir casserait
  au premier changement de pagination.
- **La présence** d'un attribut donné sur une ressource donnée. Elle dépend du fournisseur, des
  droits du jeton et de la source : un plan Terraform ignore tout de l'état effectif. C'est
  exactement la question à laquelle `provenance` répond, attribut par attribut.
- **L'exhaustivité** de l'inventaire. Un scan mesure ce à quoi ses identifiants donnent accès,
  jamais « tout le tenant ».
- **Les valeurs** elles-mêmes. Elles reflètent le contrat natif du fournisseur, qui peut
  changer sans que Pépin en décide.

## Le vocabulaire

Chaque type de ressource et les attributs communs qu'il peut porter, dérivés des descripteurs
de fournisseurs chargés et des collecteurs Go, jamais d'une liste tenue à la main à côté du
code.

<!-- pepin:gen inventory-types -->
| Type de ressource lu | Attributs communs |
|---|---|
| `access_key` | `access_key_id` `expiration_date` `owner_user` `scope` `state` |
| `api_access_policy` | `id` `max_access_key_expiration_seconds` `require_trusted_env` |
| `api_access_rule` | `api_access_rule_id` `ca_ids` `cns` `ip_ranges` |
| `api_access_summary` | `id` `rule_count` |
| `blockstorage_snapshot` | `creation_date` `global_permission` `snapshot_id` `state` `volume_id` |
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

Une entrée signifie ici « cet attribut existe dans le modèle, sur ce type ». Elle ne signifie
**pas** qu'il est présent sur une ressource donnée d'un scan donné : cette question-là est
celle de la provenance.

## Comment il bouge

La forme et ce vocabulaire sont gelés dans `cmd/testdata/frozen/inventory.json`. Ajouter un
attribut à un descripteur fait rougir le gel, et le changement coûte alors trois choses :
lancer `mise run frozen-update`, incrémenter `model.InventoryFormat`, écrire la ligne de
CHANGELOG. Ce coût est délibéré : c'est ce que veut dire, concrètement, « l'inventaire cesse
d'être un détail d'implémentation ».

## Pour aller plus loin

- [Formats de sortie](output-formats.fr.md) : les documents analysables dérivés de cet inventaire.
- [Bundles de preuve](../guides/evidence-bundles.fr.md) : là où l'inventaire est scellé, avec
  sa version de schéma.
- [Le modèle d'assessment](../concepts/assessment-model.fr.md) : comment un statut en découle.

## Comment cette page reste vraie

La chaîne de format, le vocabulaire et la forme viennent du code, par les régions générées
ci-dessus. On régénère avec `mise run gen-docs` ; `TestGeneratedDocsAreUpToDate` échoue si la
page committée et le code divergent.
