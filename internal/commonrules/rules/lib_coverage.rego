# La COUVERTURE d'une liste de sources : combien d'adresses publiques elle ouvre.
#
# # Pourquoi la fusion ne suffit pas
#
# `net.cidr_merge` ne rapproche que ce qui est CONTIGU. Deux formes lui échappent, et
# elles sont toutes deux mesurées :
#
#	256 plages `/9` une sur deux   -> 256 blocs inchangés, la MOITIÉ d'Internet, muet
#	["8.0.0.0/9", "9.0.0.0/9"]     -> 2 blocs, un /8 d'adresses publiques, muet
#
# Un damier n'a rien d'exotique : c'est ce que produit une liste d'allocations
# régionales, ou un pare-feu qui autorise « tout sauf quelques trous ».
#
# # Le décompte est EXACT, pas approché
#
# Deux propriétés le permettent sans arithmétique d'intervalles :
#
#   - `net.cidr_merge` rend des blocs DISJOINTS ;
#   - deux CIDR sont soit disjoints, soit emboîtés — jamais en chevauchement partiel.
#
# Donc, pour un bloc fusionné : zéro s'il est contenu dans un espace non public, sinon
# sa taille moins celle des espaces non publics qu'il contient. Les entiers d'OPA sont
# exacts jusqu'à 2^32, ce que ce calcul ne dépasse jamais.
#
# # Ce que ce fichier NE fait PAS
#
# L'IPv6 n'est pas décompté : 2^128 dépasse l'exactitude des entiers d'OPA, et un
# décompte faux serait pire qu'aucun. L'IPv6 reste jugé par sa LARGEUR seule
# (`::/0`, `2000::/3`, v4-mappé), ce qui couvre les formes réellement rencontrées.
package pepin.rules

import rego.v1

# _public_threshold — 2^24 adresses, soit un /8. Le MÊME seuil que celui de la largeur :
# deux seuils différents feraient dire deux choses à une seule politique.
_public_threshold := 16777216

# _pow2 — 2^k pour k de 0 à 32. Calculé une fois par évaluation, exact.
_pow2 := {k: n |
	some k in numbers.range(0, 32)
	n := product([2 |
		some i in numbers.range(1, 32)
		i <= k
	])
}

# _v4_nonpublic — les blocs IPv4 à usage spécial d'au moins un /16, DEUX À DEUX
# DISJOINTS. La disjonction est ce qui rend la soustraction exacte : deux blocs qui se
# recouvriraient seraient comptés deux fois.
#
# Les blocs plus fins (TEST-NET, 192.0.0.0/24) sont comptés comme publics : trois fois
# 256 adresses ne déplacent pas un seuil de 16,7 millions, et les énumérer coûterait
# plus de lignes que d'exactitude.
_v4_nonpublic := {
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"224.0.0.0/4",
	"240.0.0.0/4",
}

# _v4_size — le nombre d'adresses d'un CIDR IPv4. `net.cidr_is_valid` garantit un
# préfixe de 0 à 32 en amont, donc l'indexation ne peut pas manquer.
_v4_size(c) := _pow2[32 - to_number(split(c, "/")[1])]

# _public_size — les adresses PUBLIQUES d'un bloc fusionné.
_public_size(m) := 0 if _in_nonpublic(m)

_public_size(m) := _v4_size(m) - sum([_v4_size(p) |
	some p in _v4_nonpublic
	net.cidr_contains(m, p)
]) if {
	not _in_nonpublic(m)
}

_in_nonpublic(m) if {
	some p in _v4_nonpublic
	net.cidr_contains(p, m)
}

# public_coverage — les adresses publiques que cette liste de sources ouvre.
#
# La borne `count(v4) >= 2` n'est pas une optimisation : une plage SEULE relève déjà de
# `is_public_cidr`, et la compter ici produirait un second finding pour un seul fait.
public_coverage(cidrs) := n if {
	v4 := [c |
		some c in cidr_list(cidrs)
		net.cidr_is_valid(c)
		not contains(c, ":")
	]
	count(v4) >= 2

	# GARDE DE COÛT, et elle est exacte dans le bon sens : la somme brute des tailles
	# MAJORE toujours la couverture fusionnée (le recouvrement ne peut que la gonfler).
	# Sous le seuil, ni la fusion ni les inclusions ne s'exécutent — ce qui est le cas
	# de la quasi-totalité des règles réelles.
	sum([_v4_size(c) | some c in v4]) >= _public_threshold

	n := sum([_public_size(m) | some m in net.cidr_merge(v4)])
}

public_coverage(cidrs) := 0 if {
	v4 := [c |
		some c in cidr_list(cidrs)
		net.cidr_is_valid(c)
		not contains(c, ":")
	]
	count(v4) < 2
}

public_coverage(cidrs) := 0 if {
	v4 := [c |
		some c in cidr_list(cidrs)
		net.cidr_is_valid(c)
		not contains(c, ":")
	]
	count(v4) >= 2
	sum([_v4_size(c) | some c in v4]) < _public_threshold
}
