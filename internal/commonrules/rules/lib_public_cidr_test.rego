# Ce qu'une source « non restreinte » recouvre, et ce qu'elle ne doit pas recouvrir.
#
# La règle exigeait un préfixe <= 1. Mesuré sur un tenant Outscale réel : quatre plages
# `/2` couvrent tout l'espace IPv4, et RDP ouvert à tout Internet ne produisait AUCUN
# finding — pendant que `tcp 22 [0.0.0.0/1, 128.0.0.0/1]` du même groupe était attrapé.
#
# Les deux sens comptent autant l'un que l'autre : élargir la détection sans garder les
# contre-exemples muets échangerait un faux négatif contre une pluie de faux positifs,
# et un outil bruyant finit désactivé — ce qui coûte plus cher que le défaut corrigé.
package pepin.rules

import rego.v1

# ── Les formes MESURÉES sur le tenant, qui passaient toutes ────────────────────

test_quarter_of_the_internet_is_public if {
	is_public_cidr("0.0.0.0/2")
	is_public_cidr("64.0.0.0/2")
	is_public_cidr("128.0.0.0/2")
	is_public_cidr("192.0.0.0/2")
}

test_an_eighth_of_the_internet_is_public if is_public_cidr("0.0.0.0/3")

test_a_non_private_slash_eight_is_public if is_public_cidr("1.0.0.0/8")

# ── Ce qui était déjà attrapé, et doit le rester ───────────────────────────────

test_the_whole_internet_is_public if {
	is_public_cidr("0.0.0.0/0")
	is_public_cidr("0.0.0.0/1")
	is_public_cidr("128.0.0.0/1")
}

test_maskless_literals_stay_public if {
	is_public_cidr("0.0.0.0")
	is_public_cidr("*")
}

# ── LES CONTRE-EXEMPLES : ce qui doit rester muet ──────────────────────────────

# Le contre-exemple que ce contrôle ne doit jamais perdre. Un /8 privé est un /8, et
# c'est le garde-fou qui empêche l'élargissement de devenir un faux positif.
test_the_private_slash_eight_stays_silent if not is_public_cidr("10.0.0.0/8")

test_loopback_stays_silent if not is_public_cidr("127.0.0.0/8")

# « Ce réseau » : silencieux au /8, mais `0.0.0.0/2` en est un quart d'Internet — la
# contenance, pas l'octet de tête, est ce qui les sépare.
test_this_network_stays_silent_only_at_slash_eight if {
	not is_public_cidr("0.0.0.0/8")
	is_public_cidr("0.0.0.0/2")
}

# Les autres espaces privés sont plus étroits qu'un /8 : le seuil de largeur suffit à
# les taire, sans avoir à les énumérer.
test_narrower_private_ranges_stay_silent if {
	not is_public_cidr("172.16.0.0/12")
	not is_public_cidr("192.168.0.0/16")
	not is_public_cidr("100.64.0.0/10")
	not is_public_cidr("169.254.0.0/16")
}

# Le réseau d'un partenaire : EXTERNE, et parfaitement légitime. « Ouvert à Internet »
# et « ouvert à quelqu'un d'autre que moi » sont deux affirmations différentes.
test_a_partner_network_stays_silent if {
	not is_public_cidr("203.0.113.0/24")
	not is_public_cidr("198.51.100.0/24")
}

# Le réseau d'administration du contre-exemple de véracité.
test_an_administration_network_stays_silent if not is_public_cidr("10.42.0.0/16")

# ── L'UNION : ce qu'aucune plage prise isolément ne montre ─────────────────────

# Le cas MESURÉ sur le tenant, vu comme la règle le voit : une liste, pas une plage.
test_the_union_of_four_quarters_is_unrestricted if {
	unrestricted_source(["0.0.0.0/2", "64.0.0.0/2", "128.0.0.0/2", "192.0.0.0/2"])
}

# Le cas que j'avais d'abord déclaré hors de portée : 512 plages plus étroites que le
# seuil, dont l'union couvre tout. `net.cidr_merge` les collapse en `0.0.0.0/0`.
test_the_union_of_narrower_ranges_is_unrestricted if {
	# Un TABLEAU, comme le modèle normalisé le porte : `cidr_list` ne convertit qu'un
	# tableau ou un scalaire, et lui passer un ensemble rendait ce test vide — donc
	# vert pour la mauvaise raison, ce qu'il a fait au premier essai.
	moities := [c |
		some a in numbers.range(0, 255)
		some suffixe in ["0.0.0", "128.0.0"]
		c := sprintf("%d.%s/9", [a, suffixe])
	]
	count(moities) == 512
	unrestricted_source(moities)
}

# Un littéral SANS masque n'est pas un CIDR valide : la fusion le perdrait, le chemin
# brut le garde. Les deux chemins sont nécessaires.
test_a_maskless_literal_survives_the_merge if {
	unrestricted_source(["0.0.0.0"])
	unrestricted_source(["*"])
}

# Une entrée MALFORMÉE — donnée d'un tiers — ne doit pas rendre la règle muette :
# `net.cidr_merge` y devient indéfini, et le filtrage par validité est ce qui l'évite.
test_a_malformed_entry_does_not_silence_the_rule if {
	unrestricted_source(["pas-un-cidr", "0.0.0.0/0"])
}

# ── Les contre-exemples tiennent aussi pour l'union ────────────────────────────

# Deux moitiés d'un réseau privé fusionnent en ce réseau privé : toujours silencieux.
test_the_union_of_two_private_halves_stays_silent if {
	not unrestricted_source(["10.0.0.0/9", "10.128.0.0/9"])
}

test_a_list_of_partner_networks_stays_silent if {
	not unrestricted_source(["203.0.113.0/24", "198.51.100.0/24", "10.42.0.0/16"])
}

# ── LE DAMIER : ce que la fusion seule ne voit pas ─────────────────────────────

# 256 plages `/9` une sur deux : la MOITIÉ d'Internet, et rien d'adjacent à fusionner.
# `net.cidr_merge` rend les 256 blocs inchangés ; seul le décompte le voit.
test_a_checkerboard_of_half_the_internet_is_unrestricted if {
	damier := [c |
		some a in numbers.range(0, 255)
		c := sprintf("%d.0.0.0/9", [a])
	]
	count(damier) == 256
	unrestricted_source(damier)
}

# Un /8 d'adresses publiques en deux morceaux NON adjacents.
test_two_disjoint_halves_of_a_slash_eight_are_unrestricted if {
	unrestricted_source(["8.0.0.0/9", "9.0.0.0/9"])
}

# ── Le décompte ne doit pas devenir un faux positif ────────────────────────────

# Deux moitiés d'un réseau PRIVÉ : le décompte les voit, et les compte à zéro.
test_two_private_halves_count_as_zero if {
	public_coverage(["10.0.0.0/9", "10.128.0.0/9"]) == 0
	not unrestricted_source(["10.0.0.0/9", "10.128.0.0/9"])
}

# Une liste de réseaux partenaires reste très en dessous du seuil.
test_a_list_of_partner_networks_stays_below_the_threshold if {
	public_coverage(["203.0.113.0/24", "198.51.100.0/24", "192.0.2.0/24"]) < 16777216
	not unrestricted_source(["203.0.113.0/24", "198.51.100.0/24", "192.0.2.0/24"])
}

# Une plage SEULE ne passe jamais par le décompte : elle relève de la largeur, et la
# compter ici produirait un second finding pour un seul fait.
test_a_single_range_never_goes_through_the_count if {
	public_coverage(["0.0.0.0/0"]) == 0
	public_coverage(["1.0.0.0/8"]) == 0
}

# ── UNE étiquette, pas une par plage ───────────────────────────────────────────

# La régression que la fermeture de l'union avait introduite : quatre `/2` produisaient
# CINQ findings — les quatre plages, plus leur fusion — pour un seul fait.
test_one_label_for_one_fact if {
	unrestricted_label(["0.0.0.0/2", "64.0.0.0/2", "128.0.0.0/2", "192.0.0.0/2"]) == "0.0.0.0/0"
	unrestricted_label(["10.42.0.0/16"]) == ""
}

# La forme fusionnée est PRÉFÉRÉE : elle dit d'un coup ce qu'aucune plage ne montre.
test_the_label_prefers_the_merged_form if {
	unrestricted_label(["0.0.0.0/1", "128.0.0.0/1"]) == "0.0.0.0/0"
}

# Un littéral sans masque n'est pas un CIDR valide : la voie brute le porte.
test_the_label_keeps_maskless_literals if {
	unrestricted_label(["*"]) == "*"
	unrestricted_label(["0.0.0.0"]) == "0.0.0.0"
}
