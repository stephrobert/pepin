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
