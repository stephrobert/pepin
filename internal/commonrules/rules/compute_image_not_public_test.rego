package pepin.rules

import rego.v1

_img(attrs) := {"resources": [{"provider": "outscale", "type": "compute_image", "id": "ami-1", "attributes": attrs}]}

# ✗ Image publique → finding.
test_image_public_denied if {
	some f in deny with input as _img({"image_id": "ami-1", "public": true})
	f.code == "compute_image_not_public"
}

# ✓ Image privée → aucun finding.
test_image_private_ok if {
	count({f | some f in deny; f.code == "compute_image_not_public"}) == 0 with input as _img({"image_id": "ami-1", "public": false})
}

# ✗ Forme chaîne « true » (plan Terraform : global_permission est une string) → finding.
test_image_public_string_denied if {
	some f in deny with input as _img({"image_id": "ami-1", "public": "true"})
	f.code == "compute_image_not_public"
}

# ✓ Chaîne « false » → aucun finding.
test_image_private_string_ok if {
	count({f | some f in deny; f.code == "compute_image_not_public"}) == 0 with input as _img({"image_id": "ami-1", "public": "false"})
}

# ✓ Attribut non collecté (garde de capacité implicite : défaut false) → pas de faux positif.
test_image_uncollected_ok if {
	count({f | some f in deny; f.code == "compute_image_not_public"}) == 0 with input as _img({"image_id": "ami-1"})
}
