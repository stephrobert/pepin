# compute_instance_no_secrets_in_user_data
#   Secret en clair détecté dans les données utilisateur (user-data) d'une VM :
#   un accès aux métadonnées ou un SSRF peut l'exfiltrer.
#
# # NIVEAUX DE CONFIANCE (issue #48)
#
# Chaque motif porte son niveau, et le finding le publie (`labels.confidence`) :
#
#   high   — secret CONFIRMÉ par sa forme : un bloc PEM `-----BEGIN … PRIVATE KEY-----`
#            n'est pas autre chose qu'une clé privée ;
#   medium — secret PROBABLE : préfixe reconnu ET format attendu (AKIA…, SCW…,
#            EXO…, ghp_…, glpat-…, JWT à trois segments) ;
#   low    — SUSPECT : heuristique générique (`password=…`, `api_key=…`), qui ne
#            distingue pas `password=changeme123456` d'un vrai secret.
#
# Pourquoi ça compte : donner à `password=…` le même poids qu'une clé privée rend
# l'outil irritant en CI, et un outil irritant finit désactivé — ce qui coûte plus
# cher que le faux positif qu'on voulait éviter. Le seuil est donc RÉGLABLE
# (`secrets.min_confidence`, défaut `low` = tout signaler, le comportement
# d'avant). Le monter est un ASSOUPLISSEMENT : le référentiel adosse la
# correspondance CLD-CMP-9 à `secrets.min_confidence: au_plus_le_defaut`, et un
# seuil plus haut la fait tomber, visiblement.
#
# Le niveau voyage dans `labels.confidence`, PAS dans le message. Deux raisons :
# le message d'un contrôle est une surface que des captures, des tickets et des
# annotations de forge recopient — le déplacer d'un octet pour une information
# structurée serait le payer partout —, et un niveau est fait pour être FILTRÉ
# (`jq '.findings[] | select(.labels.confidence == "low")'`), pas lu au milieu
# d'une phrase. Le rendu par défaut n'a donc pas bougé d'un caractère.
#
# # LA VALEUR DÉTECTÉE NE SORT JAMAIS
#
# Le message n'interpole que le LIBELLÉ du motif, jamais ce qui a matché — quel
# que soit le niveau. Un détecteur de secrets qui recopie le secret dans son
# rapport transforme le rapport en fuite, et ce rapport voyage en SARIF, dans les
# artefacts de CI et dans un bundle scellé. La propriété est TESTÉE
# (`test_secret_value_never_leaks_*`), pas seulement écrite ici.
#
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
	meets_confidence(pattern.confidence)
	id := object.get(vm.attributes, "vm_id", vm.id)
	f := {
		"code": "compute_instance_no_secrets_in_user_data",
		"severity": "high",
		"subject": id,
		"message": sprintf("VM « %s » : secret en clair dans user-data (%s).", [id, pattern.fr]),
		"remediation": "Bannir les secrets des données utilisateur ; utiliser un coffre de secrets et l'injection au démarrage. Révoquer le secret exposé.",
		"labels": {
			"provider": provider_of(vm),
			"category": "security",
			"confidence": pattern.confidence,
			"message_en": sprintf("VM \"%s\": cleartext secret in user-data (%s).", [id, pattern.en]),
			"remediation_en": "Ban secrets from user data; use a secrets vault and inject them at boot. Revoke the exposed secret.",
		},
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

# _secret_regexes : motifs de détection, chacun avec son LIBELLÉ dans les deux
# langues et son NIVEAU DE CONFIANCE. Le libellé est interpolé dans le message du
# finding : le garder monolingue produirait une phrase anglaise à moitié
# française, exactement ce que l'internationalisation corrige. La clé de la carte
# est le libellé français (langue de référence) ; `en` en est la traduction, `re`
# le motif, `confidence` ce que la détection vaut.
_secret_regexes := {
	# Clés d'accès cloud, y compris SOUVERAINES (préfixes documentés). PRÉFIXE
	# RECONNU + FORMAT ATTENDU : probable, pas confirmé — une documentation peut
	# citer un exemple de clé.
	"clé d'accès Outscale (format AKIA…)": {
		"en": "Outscale access key (AKIA… format)",
		"re": `AKIA[0-9A-Z]{16}`,
		"confidence": "medium",
	},
	"clé d'accès Scaleway (SCW…)": {
		"en": "Scaleway access key (SCW…)",
		"re": `SCW[A-Z0-9]{17,}`,
		"confidence": "medium",
	},
	"clé d'accès Exoscale (EXO…)": {
		"en": "Exoscale access key (EXO…)",
		"re": `EXO[A-Za-z0-9]{16,}`,
		"confidence": "medium",
	},
	# Jetons de forge / d'identité fréquemment collés dans le user-data.
	"jeton GitHub": {
		"en": "GitHub token",
		"re": `(gh[pousr]_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{22,})`,
		"confidence": "medium",
	},
	"jeton GitLab (glpat-)": {
		"en": "GitLab token (glpat-)",
		"re": `glpat-[A-Za-z0-9_-]{20}`,
		"confidence": "medium",
	},
	"jeton JWT": {
		"en": "JWT token",
		"re": `eyJ[A-Za-z0-9_-]{8,}\.eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`,
		"confidence": "medium",
	},
	"en-tête Authorization Bearer": {
		"en": "Authorization Bearer header",
		"re": `(?i)authorization\s*:\s*bearer\s+[A-Za-z0-9._-]{12,}`,
		"confidence": "medium",
	},
	# Affectation de mot de passe EN CLAIR. La valeur doit suivre le séparateur SUR LA MÊME
	# LIGNE (>=4 caractères non blancs) : évite le faux positif sur les blocs cloud-init
	# `chpasswd:` / `password:` suivis d'un bloc YAML (la valeur passe alors à la ligne).
	# HEURISTIQUE GÉNÉRIQUE : elle ne distingue pas un vrai mot de passe d'un
	# `password=changeme` de gabarit → `low`.
	"mot de passe en clair": {
		"en": "cleartext password",
		"re": `(?i)(password|passwd|pwd)[ \t]*[:=][ \t]*['"]?[^\s'"]{4,}`,
		"confidence": "low",
	},
	# Bloc cloud-init `chpasswd` : la façon la PLUS courante de poser un mot de passe.
	# La valeur est sur une ligne indentée `utilisateur:motdepasse` — sans espace après
	# le deux-points, ce qui la distingue d'une clé YAML (`expire: true`).
	"mot de passe cloud-init (chpasswd)": {
		"en": "cloud-init password (chpasswd)",
		"re": `(?i)chpasswd:[\s\S]{0,300}?\n[ \t]+[A-Za-z0-9_.-]+:[^\s]{6,}`,
		"confidence": "low",
	},
	"clé/API générique affectée": {
		"en": "generic key/API assignment",
		"re": `(?i)(api[_-]?key|secret[_-]?key|access[_-]?key|secret[_-]?access[_-]?key|auth[_-]?token|api[_-]?token)\s*[:=]\s*['"]?[A-Za-z0-9/+._-]{16,}`,
		"confidence": "low",
	},
}

# _secret_patterns — ENSEMBLE des types de secrets détectés (un user-data peut en
# cumuler plusieurs : clé d'accès ET mot de passe…). Un set évite le conflit d'une
# fonction single-value à clauses non exclusives. Chaque élément porte le libellé
# dans les deux langues (`fr`, `en`) et son niveau (`confidence`) — jamais la
# valeur qui a matché.
_secret_patterns(s) := patterns if {
	by_regex := {{"fr": label, "en": spec.en, "confidence": spec.confidence} |
		some label, spec in _secret_regexes
		regex.match(spec.re, s)
	}

	# Un bloc PEM est un secret CONFIRMÉ : sa forme ne laisse pas de place au doute.
	private_key := {{"fr": "bloc PRIVATE KEY", "en": "PRIVATE KEY block", "confidence": "high"} |
		contains(s, "BEGIN ")
		contains(s, "PRIVATE KEY")
	}
	patterns := by_regex | private_key
}
