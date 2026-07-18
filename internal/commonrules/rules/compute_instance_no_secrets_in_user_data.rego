# compute_instance_no_secrets_in_user_data
#   Secret en clair détecté dans les données utilisateur (user-data) d'une VM :
#   un accès aux métadonnées ou un SSRF peut l'exfiltrer.
# Origine : osc-policy OSC-VM-008. SCSL : CLD-CMP-2.
# Contrat : type normalisé agnostique `compute_instance` ; attribut user_data
#   (string ; osc-sdk-go Vm.UserData renvoyé en base64 par l'API).
package pepin.rules

import rego.v1

deny contains f if {
	some vm in resources_of_type("compute_instance")
	raw := _user_data(vm.attributes)
	raw != ""
	some pattern in _secret_patterns(raw)
	id := object.get(vm.attributes, "vm_id", vm.id)
	f := {
		"code": "compute_instance_no_secrets_in_user_data",
		"severity": "high",
		"subject": id,
		"message": sprintf("VM « %s » : secret en clair dans user-data (%s).", [id, pattern]),
		"remediation": "Bannir les secrets des données utilisateur ; utiliser un coffre de secrets et l'injection au démarrage. Révoquer le secret exposé.",
		"labels": {"provider": provider_of(vm), "category": "security"},
	}
}

# _user_data — décode le base64 si nécessaire (l'API renvoie du base64 ;
# un plan Terraform peut renvoyer le texte brut).
_user_data(attrs) := decoded if {
	raw := object.get(attrs, "user_data", "")
	is_string(raw)
	_is_base64(raw)
	decoded := base64.decode(raw)
}

_user_data(attrs) := raw if {
	raw := object.get(attrs, "user_data", "")
	is_string(raw)
	not _is_base64(raw)
}

# user_data peut être un map clé→valeur (ex. Scaleway : {cloud-init: "…"}) ;
# on concatène ses valeurs textuelles pour y chercher des secrets.
_user_data(attrs) := concat("\n", vals) if {
	m := object.get(attrs, "user_data", "")
	is_object(m)
	vals := [v | some k; v := m[k]; is_string(v)]
}

_is_base64(s) if {
	s != ""
	regex.match(`^[A-Za-z0-9+/]+={0,2}$`, s)
	count(s) % 4 == 0
}

# _secret_patterns — ENSEMBLE des types de secrets détectés (un user-data peut en
# cumuler plusieurs : clé d'accès ET mot de passe…). Un set évite le conflit d'une
# fonction single-value à clauses non exclusives.
_secret_patterns(s) := patterns if {
	by_regex := {label |
		some label, re in {
			"clé d'accès (AKIA…)": `AKIA[0-9A-Z]{16}`,
			"mot de passe en clair": `(?i)password\s*[:=]\s*\S`,
		}
		regex.match(re, s)
	}
	private_key := {"bloc PRIVATE KEY" |
		contains(s, "BEGIN ")
		contains(s, "PRIVATE KEY")
	}
	patterns := by_regex | private_key
}
