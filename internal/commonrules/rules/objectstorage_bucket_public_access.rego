# objectstorage_bucket_public_access — COMMUN à tous les providers.
#   Bucket de stockage objet accessible publiquement : ACL « canned » publique,
#   grant ACL au groupe AllUsers, ou bucket policy avec Principal « * ».
# SCSL : CLD-STO-1. Le stockage objet souverain est S3-compatible ; le collecteur
# partagé internal/objectstorage produit `object_storage_bucket` pour chaque
# provider. Attributs : acl (canned), acl_grants[], public_via_acl, policy_public.
package pepin.rules

import rego.v1

_public_canned_acls := {"public-read", "public-read-write"}
_all_users_uri := "http://acs.amazonaws.com/groups/global/AllUsers"

deny contains f if {
	some r in resources_of_type("object_storage_bucket")
	object.get(r.attributes, "acl", "") in _public_canned_acls
	f := _bucket_public_finding(r, "ACL publique")
}

deny contains f if {
	some r in resources_of_type("object_storage_bucket")
	_acl_grant_public(r.attributes)
	f := _bucket_public_finding(r, "ACL accordant l'accès au groupe public AllUsers")
}

deny contains f if {
	some r in resources_of_type("object_storage_bucket")
	object.get(r.attributes, "policy_public", false) == true
	f := _bucket_public_finding(r, "bucket policy avec Principal « * »")
}

_acl_grant_public(attrs) if object.get(attrs, "public_via_acl", false) == true

_acl_grant_public(attrs) if {
	some g in object.get(attrs, "acl_grants", [])
	object.get(object.get(g, "grantee", {}), "uri", "") == _all_users_uri
}

_bucket_public_finding(r, cause) := {
	"code": "objectstorage_bucket_public_access",
	"severity": "critical",
	"subject": object.get(r.attributes, "name", r.id),
	"message": sprintf("Bucket « %s » accessible publiquement (%s).", [object.get(r.attributes, "name", r.id), cause]),
	"remediation": "Rendre le bucket privé (ACL private, retrait du grant AllUsers, suppression de la policy publique) ; servir via des URLs pré-signées si nécessaire.",
	"labels": {"provider": provider_of(r), "category": "security"},
}
