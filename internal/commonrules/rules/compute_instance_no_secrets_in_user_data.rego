# compute_instance_no_secrets_in_user_data
#   Secret en clair détecté dans les données utilisateur (user-data) d'une VM :
#   un accès aux métadonnées ou un SSRF peut l'exfiltrer.
# SCSL : CLD-CMP-9 (aucun secret en clair dans les user-data).
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

# _is_base64 — chaîne réellement base64 : alphabet valide, longueur multiple de 4, ET le
# décodage produit du TEXTE imprimable (le user-data encodé est du cloud-init, du texte). Sans
# le test d'imprimabilité, un user-data EN CLAIR qui matche par accident l'alphabet base64
# serait décodé en binaire et la détection de secret perdue (faux négatif).
_is_base64(s) if {
	s != ""
	regex.match(`^[A-Za-z0-9+/]+={0,2}$`, s)
	count(s) % 4 == 0
	regex.match(`^[[:print:][:space:]]*$`, base64.decode(s))
}

# _secret_patterns — ENSEMBLE des types de secrets détectés (un user-data peut en
# cumuler plusieurs : clé d'accès ET mot de passe…). Un set évite le conflit d'une
# fonction single-value à clauses non exclusives.
_secret_patterns(s) := patterns if {
	by_regex := {label |
		some label, re in {
			# Clés d'accès cloud, y compris SOUVERAINES (préfixes documentés).
			"clé d'accès Outscale (format AKIA…)": `AKIA[0-9A-Z]{16}`,
			"clé d'accès Scaleway (SCW…)": `SCW[A-Z0-9]{17,}`,
			"clé d'accès Exoscale (EXO…)": `EXO[A-Za-z0-9]{16,}`,
			# Jetons de forge / d'identité fréquemment collés dans le user-data.
			"jeton GitHub": `(gh[pousr]_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{22,})`,
			"jeton GitLab (glpat-)": `glpat-[A-Za-z0-9_-]{20}`,
			"jeton JWT": `eyJ[A-Za-z0-9_-]{8,}\.eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`,
			"en-tête Authorization Bearer": `(?i)authorization\s*:\s*bearer\s+[A-Za-z0-9._-]{12,}`,
			# Affectation de mot de passe EN CLAIR. La valeur doit suivre le séparateur SUR LA MÊME
			# LIGNE (>=4 caractères non blancs) : évite le faux positif sur les blocs cloud-init
			# `chpasswd:` / `password:` suivis d'un bloc YAML (la valeur passe alors à la ligne).
			"mot de passe en clair": `(?i)(password|passwd|pwd)[ \t]*[:=][ \t]*['"]?[^\s'"]{4,}`,
			# Bloc cloud-init `chpasswd` : la façon la PLUS courante de poser un mot de passe.
			# La valeur est sur une ligne indentée `utilisateur:motdepasse` — sans espace après
			# le deux-points, ce qui la distingue d'une clé YAML (`expire: true`).
			"mot de passe cloud-init (chpasswd)": `(?i)chpasswd:[\s\S]{0,300}?\n[ \t]+[A-Za-z0-9_.-]+:[^\s]{6,}`,
			"clé/API générique affectée": `(?i)(api[_-]?key|secret[_-]?key|access[_-]?key|secret[_-]?access[_-]?key|auth[_-]?token|api[_-]?token)\s*[:=]\s*['"]?[A-Za-z0-9/+._-]{16,}`,
		}
		regex.match(re, s)
	}
	private_key := {"bloc PRIVATE KEY" |
		contains(s, "BEGIN ")
		contains(s, "PRIVATE KEY")
	}
	patterns := by_regex | private_key
}
