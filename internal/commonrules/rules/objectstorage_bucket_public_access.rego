# objectstorage_bucket_public_access — COMMUN à tous les providers.
#   Bucket de stockage objet accessible publiquement : ACL « canned » publique,
#   grant ACL au groupe AllUsers, ou bucket policy avec Principal « * ».
# SCSL : CLD-STO-1. Le stockage objet souverain est S3-compatible ; le collecteur
# partagé internal/objectstorage produit `object_storage_bucket` pour chaque
# provider. Attributs : acl (canned), acl_grants[], public_via_acl, policy_public.
package pepin.rules

import rego.v1

# `authenticated-read` accorde la lecture à TOUT utilisateur authentifié de la
# plateforme, donc hors du tenant : c'est une exposition inter-tenant, pas une
# nuance de configuration. Confirmé par la doc Outscale (« private | public-read |
# public-read-write | authenticated-read ») et par le mapping Scaleway.
_public_canned_acls := {"public-read", "public-read-write", "authenticated-read"}

# Même raisonnement pour les grants : AuthenticatedUsers est un groupe global,
# au même titre qu'AllUsers.
_public_group_uris := {
	"http://acs.amazonaws.com/groups/global/AllUsers",
	"http://acs.amazonaws.com/groups/global/AuthenticatedUsers",
}

deny contains f if {
	some r in resources_of_type("object_storage_bucket")
	lower(object.get(r.attributes, "acl", "")) in _public_canned_acls
	f := _bucket_public_finding(r, "ACL publique")
}

deny contains f if {
	some r in resources_of_type("object_storage_bucket")
	_acl_grant_public(r.attributes)
	f := _bucket_public_finding(r, "ACL accordant l'accès à un groupe global (AllUsers / AuthenticatedUsers)")
}

deny contains f if {
	some r in resources_of_type("object_storage_bucket")
	truthy(object.get(r.attributes, "policy_public", false))
	f := _bucket_public_finding(r, "bucket policy avec Principal « * »")
}

_acl_grant_public(attrs) if truthy(object.get(attrs, "public_via_acl", false))

_acl_grant_public(attrs) if {
	some g in object.get(attrs, "acl_grants", [])
	object.get(object.get(g, "grantee", {}), "uri", "") in _public_group_uris
}

_bucket_public_finding(r, cause) := {
	"code": "objectstorage_bucket_public_access",
	"severity": "critical",
	"subject": object.get(r.attributes, "name", r.id),
	"message": sprintf("Bucket « %s » accessible publiquement (%s).", [object.get(r.attributes, "name", r.id), cause]),
	"remediation": "Rendre le bucket privé (ACL private, retrait du grant AllUsers, suppression de la policy publique) ; servir via des URLs pré-signées si nécessaire.",
	"labels": {"provider": provider_of(r), "category": "security"},
}
