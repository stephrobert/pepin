# Helpers COMMUNS à tous les providers (package pepin.rules : OPA fusionne les
# fichiers d'un même package). Chargés à chaque scan en plus des règles du
# provider. S'appliquent au modèle normalisé de Pépin (input.resources[]).
package pepin.rules

import rego.v1

# resources_of_type — ressources de l'inventaire normalisé du type donné.
# Les types sont AGNOSTIQUES (vocabulaire commun : security_group,
# security_group_rule, compute_instance, object_storage_bucket, iam_policy,
# access_key, kubernetes_cluster, load_balancer…) ; les attributs restent natifs.
# Le collecteur live et le parseur Terraform projettent tous deux vers ces types.
resources_of_type(t) := [r | some r in input.resources; r.type == t]

# is_public_cidr — le CIDR couvre l'Internet public (IPv4 ou IPv6). On PARSE la longueur de
# préfixe au lieu de comparer des littéraux : un préfixe /0 couvre tout l'espace d'adressage
# quelle que soit l'écriture (`0.0.0.0/0`, `::/0`, `::0/0`, `0000::/0`). Un /1 (IPv4
# `0.0.0.0/1`|`128.0.0.0/1`, ou IPv6 `::/1`) couvre la MOITIÉ de l'espace d'adressage :
# contournement CSPM connu, jamais légitime comme source « restreinte » → également public.
is_public_cidr(cidr) if _cidr_prefix(trim_space(cidr)) <= 1

# Littéraux « toute origine » SANS masque (certaines API/UI : une source vide/`0.0.0.0`/`*`).
is_public_cidr(cidr) if trim_space(cidr) in {"0.0.0.0", "::", "::0", "*"}

# IPv6 mappé-IPv4 couvrant TOUT l'espace IPv4 (`::ffff:0.0.0.0/<=96` ou sa forme compressée
# `::ffff:0:0/<=96`) : contournement connu.
is_public_cidr(cidr) if {
	t := lower(trim_space(cidr))
	_ipv4_mapped_all(t)
	_cidr_prefix(t) <= 96
}

_ipv4_mapped_all(t) if startswith(t, "::ffff:0.0.0.0")

_ipv4_mapped_all(t) if startswith(t, "::ffff:0:0")

# `2000::/<=3` couvre tout l'IPv6 global-unicast (= l'Internet IPv6 alloué) : source non
# restreinte, jamais légitime → public.
is_public_cidr(cidr) if {
	t := lower(trim_space(cidr))
	startswith(t, "2000::")
	_cidr_prefix(t) <= 3
}

_cidr_prefix(cidr) := n if {
	parts := split(cidr, "/")
	count(parts) == 2
	regex.match(`^[0-9]+$`, parts[1])
	n := to_number(parts[1])
}

# _eval_now_ns — instant d'ÉVALUATION en nanosecondes. Si le scan a injecté
# `input.evaluated_at` (RFC3339, horodatage du run porté par le bundle), on l'utilise : une
# règle sensible au temps (fenêtres de fraîcheur) redonne alors le MÊME verdict au rejeu du
# bundle (reproductibilité/opposabilité). À défaut (opa test direct), on retombe sur l'horloge.
_eval_now_ns := time.parse_rfc3339_ns(ts) if {
	ts := object.get(input, "evaluated_at", "")
	ts != ""
}

_eval_now_ns := time.now_ns() if object.get(input, "evaluated_at", "") == ""

# truthy — valeur booléenne vraie, que la source la donne en bool (API live, ex. OAPI
# *bool) OU en chaîne « true » (plan Terraform : certains attributs de schéma sont des
# strings). Insensible à la casse. false / "false" / absent → faux.
truthy(v) if lower(sprintf("%v", [v])) == "true"

# has_tag — true si la liste de tags ({key,value}) porte la clé avec une valeur
# non vide. Contrat natif : Tag{Key, Value} (snake_case normalisé).
has_tag(tags, key) if {
	some t in tags
	t.key == key
	t.value != ""
}

# get_tag — valeur du tag pour la clé, ou "" si absent.
get_tag(tags, key) := v if {
	some t in tags
	t.key == key
	v := t.value
} else := ""

# volume_in_use — le volume block storage est rattaché à une machine (en usage),
# donc à protéger par une sauvegarde. Valeurs d'état NATIVES normalisées :
# Outscale `in-use`, Exoscale `attached`, Scaleway `in_use` (api/block/v1, underscore).
# Un état inconnu n'est pas considéré en usage (pas de faux positif).
volume_in_use(v) if object.get(v.attributes, "state", "") in {"in-use", "in_use", "attached"}

# covers_port — la règle de filtrage couvre le port p (schéma SG commun : port_from/port_to).
# Deux cas :
#   1. au moins une borne numérique définie ⇒ plage [from, to] (to = from si absent : port unique) ;
#   2. AUCUNE borne (règle « tous les ports », p. ex. le collecteur live projette port_from/to en
#      chaîne vide) ⇒ couvre TOUT port. Sans ce cas, une règle tcp sans ports ouverte à Internet
#      (= any/any TCP) échappait à tous les checks (faux négatif).
covers_port(rule, p) if {
	from := _port_bound(rule, "port_from")
	to := _to_bound(rule, from)
	from <= p
	p <= to
}

covers_port(rule, _) if {
	not _port_bound(rule, "port_from")
	not _port_bound(rule, "port_to")
}

# 3. sentinelle « tous les ports » : plusieurs API (ex. Outscale OAPI) encodent une plage
#    complète par FromPortRange/ToPortRange = -1. Sans ce cas, [-1,-1] ne couvrirait AUCUN
#    port (faux négatif : une règle -1/-1 ouverte à Internet = any/any échapperait aux checks).
covers_port(rule, _) if _port_bound(rule, "port_from") == -1

covers_port(rule, _) if _port_bound(rule, "port_to") == -1

# _port_bound — borne de port numérique, défensif face au typage : accepte un nombre ou une
# chaîne purement numérique ; undefined si absente/vide/non numérique (le « » du moteur live).
_port_bound(rule, key) := v if {
	v := object.get(rule, key, "")
	is_number(v)
}

_port_bound(rule, key) := to_number(v) if {
	v := object.get(rule, key, "")
	is_string(v)
	regex.match(`^[0-9]+$`, v)
}

_to_bound(rule, from) := max([from, _port_bound(rule, "port_to")])

_to_bound(rule, from) := from if not _port_bound(rule, "port_to")

# proto_covers — la règle couvre le protocole `want` (ou « all »). Schéma SG
# commun : protocol ∈ tcp|udp|icmp|all.
proto_covers(rule, want) if lower(object.get(rule, "protocol", "")) == want

# « tout protocole » se dit de plusieurs facons selon la source : « all », « any »,
# le -1 de l'API Outscale (que les specs traduisent, mais pas toujours), ou un
# champ absent/vide. Ne reconnaitre que « all » laissait passer une regle
# any/any depuis Internet sans le moindre finding.
proto_covers(rule, _) if {
	p := lower(trim_space(sprintf("%v", [object.get(rule, "protocol", "")])))
	p in {"all", "any", "-1", ""}
}

# sg_accepting — la règle AUTORISE le trafic. Ensemble EXPLICITE d'actions acceptantes
# (accept/allow, ou absente ⇒ modèle allow-list type Outscale) : toute autre valeur
# (drop, deny, reject…) est bloquante. On ne présume PAS acceptant « tout ce qui n'est pas
# drop » — un `deny`/`reject` d'un futur provider serait alors compté à tort comme ouvert.
sg_accepting(rule) if lower(object.get(rule, "action", "accept")) in {"accept", "allow", ""}

# sg_inbound_from_internet — règle entrante acceptante dont la source couvre
# Internet ; renvoie true si au moins un CIDR public est présent.
sg_inbound_from_internet(rule) if {
	lower(object.get(rule, "direction", "")) == "inbound"
	sg_accepting(rule)
	some cidr in cidr_list(object.get(rule, "cidrs", []))
	is_public_cidr(cidr)
}

# cidr_list — accepte la liste du modele normalise AUSSI BIEN qu'un scalaire.
# `pepin scan <provider> export.json` est une entree de premier rang alimentee par
# un tiers : « cidrs »: "0.0.0.0/0" y arrive tel quel, et `some x in "0.0.0.0/0"`
# est indefini en Rego — la regle se taisait sur un SSH grand ouvert.
cidr_list(v) := v if is_array(v)

cidr_list(v) := [v] if is_string(v)

cidr_list(v) := [] if {
	not is_array(v)
	not is_string(v)
}

# sensitive_ports — ports d'administration distante, de partage de fichiers, de
# bases de données et d'orchestration à ne jamais exposer à Internet (0.0.0.0/0).
# Sourcé sur les benchmarks CIS et les listings CSPM courants.
sensitive_ports := {
	22, # SSH
	23, # Telnet
	3389, # RDP
	5900, # VNC
	5985, 5986, # WinRM (HTTP/HTTPS)
	21, # FTP
	445, # SMB
	139, # NetBIOS
	389, 636, # LDAP / LDAPS
	2049, # NFS
	3306, # MySQL/MariaDB
	5432, # PostgreSQL
	1433, # MSSQL
	1521, # Oracle
	27017, 27018, 27019, # MongoDB
	6379, # Redis
	11211, # Memcached
	9200, 9300, # Elasticsearch (HTTP / transport)
	5601, # Kibana
	5984, # CouchDB
	9042, # Cassandra
	8086, # InfluxDB
	9092, # Kafka
	5672, 15672, # RabbitMQ (AMQP / management)
	2375, 2376, # Docker daemon (plain / TLS)
	2379, 2380, # etcd
	6443, # Kubernetes API server
	10250, # Kubelet
}

# sensitive_udp_ports — services UDP à ne jamais exposer à Internet : vecteurs d'amplification
# DDoS et services non authentifiés (sourcé CIS + advisories US-CERT sur l'amplification).
sensitive_udp_ports := {
	53, # DNS
	123, # NTP
	161, # SNMP
	111, # rpcbind / portmapper
	389, # LDAP (CLDAP amplification)
	500, # IKE/IPsec
	1900, # SSDP/UPnP
	2049, # NFS
	11211, # Memcached
	5353, # mDNS
}

# provider_of — provider d'une ressource (pour `labels.provider` des règles
# communes ; tiré de la ressource, jamais codé en dur).
provider_of(r) := object.get(r, "provider", "")

# Localisation des régions — données de RÉFÉRENCE (souveraineté, CLD-GVN-3).
# Critère de classement : GÉOGRAPHIE EUROPÉENNE + absence de loi extraterritoriale
# — PAS la simple décision d'adéquation RGPD (qui couvre aussi US/UK/Japon, non
# souverains au sens européen ; le CLOUD Act suit la nationalité du fournisseur,
# pas la localisation). Trois classes :
#   _eu_regions      : États membres de l'UE → conforme.
#   _trusted_regions : EEE (NO/IS/LI) + Suisse → européen, niveau de protection
#                      adéquat, sans extraterritorialité type CLOUD Act → écart MINEUR.
#   _noneu_regions   : hors espace souverain européen (US, Asie…) → écart MAJEUR.
# La règle ne se déclenche que sur une région CATALOGUÉE : une région inconnue
# n'émet aucun finding (table extensible, pas de faux positif).
_eu_regions := {
	"scaleway": {"fr-par", "nl-ams", "pl-waw"},
	"outscale": {"eu-west-2", "cloudgouv-eu-west-1"},
	"exoscale": {"de-fra-1", "de-muc-1", "at-vie-1", "at-vie-2", "bg-sof-1", "hr-zag-1"},
}

_trusted_regions := {
	"scaleway": set(),
	"outscale": set(),
	"exoscale": {"ch-gva-2", "ch-dk-2"},
}

_noneu_regions := {
	"scaleway": set(),
	"outscale": {"us-east-2", "us-west-1", "ap-northeast-1"},
	"exoscale": set(),
}

# region_in_eu — la région du provider est dans l'Union européenne (GVN-3 strict).
region_in_eu(p, reg) if reg in _eu_regions[p]

# region_trusted — région hors UE mais dans l'espace européen de confiance
# (EEE + Suisse) : écart mineur, pas d'exposition extraterritoriale.
region_trusted(p, reg) if reg in _trusted_regions[p]

# region_known — la région est cataloguée (UE, confiance, ou hors souverain).
region_known(p, reg) if reg in _eu_regions[p]

region_known(p, reg) if reg in _trusted_regions[p]

region_known(p, reg) if reg in _noneu_regions[p]

# ─────────────────────────── Configuration des contrôles ───────────────────────────
#
# Quatre contrôles sont RÉGLABLES (issues #47, #48, #49, #61). Leur configuration
# effective est injectée par `pepin scan` dans `input.config`, exactement comme
# `input.evaluated_at` : le moteur partagé ne passe que l'input, et une
# configuration portée par l'input est scellée dans l'input.json du bundle — donc
# rejouée à l'identique par `verify --re-derive`.
#
# Chaque lecture porte son DÉFAUT ici. Ce défaut a deux emplois, et un seul est
# celui du produit :
#
#   - `opa test` évalue les règles sans passer par le scan : sans défaut local,
#     tous les tests de règles deviendraient des tests de configuration ;
#   - une règle externe (--policy-dir) ou un input.json d'une version antérieure
#     n'apportent pas de `config` : le repli est alors le comportement d'avant.
#
# Le défaut Rego et le profil par défaut Go (internal/policy) doivent donc dire la
# MÊME chose. Ce n'est pas laissé à la relecture : TestDefaultConfigMatchesTheRegoFallback
# scanne chaque fixture du dépôt avec et sans configuration injectée et exige des
# findings identiques.

_config := object.get(input, "config", {})

_config_tagging := object.get(_config, "tagging", {})

_config_snapshots := object.get(_config, "snapshots", {})

_config_secrets := object.get(_config, "secrets", {})

# normalize_tag_key — forme COMPARABLE d'un nom d'étiquette : minuscules, sans
# séparateur. `CostCenter`, `cost-center`, `cost_center` et `Cost Center` s'y
# réduisent tous à `costcenter`. Une convention d'écriture n'est pas une norme :
# refuser `cost-center` à qui n'écrit pas `CostCenter` est un faux positif, et un
# outil qui crie au loup sur une typographie finit désactivé.
normalize_tag_key(k) := lower(regex.replace(k, `[-_. /:]`, ""))

# required_tags_billable / required_tags_network — les étiquettes exigées, sous la
# forme [{name, keys}] : le nom LOGIQUE (celui que lit un humain dans le message)
# et les écritures acceptées, déjà normalisées côté Go.
required_tags_billable := object.get(_config_tagging, "required", _default_required_billable)

required_tags_network := object.get(_config_tagging, "network_required", _default_required_network)

# tagged_resource_types — les types de ressources sur lesquels l'étiquetage de
# gouvernance est exigé (critère : facturable ET étiquetable, cf. internal/policy).
tagged_resource_types := object.get(_config_tagging, "resource_types", _default_tagged_types)

# missing_required_tags — les noms logiques d'étiquettes qui MANQUENT sur la liste
# de tags donnée. Une étiquette présente mais vide ne documente rien : elle compte
# comme manquante, comme dans has_tag.
missing_required_tags(tags, required) := [req.name |
	some req in required
	not _tag_satisfied(tags, req)
]

_tag_satisfied(tags, req) if {
	some t in tags
	normalize_tag_key(object.get(t, "key", "")) in req.keys
	object.get(t, "value", "") != ""
}

# required_tags_label — la liste des étiquettes exigées, telle qu'une remédiation
# la cite. Dérivée de la politique EFFECTIVE et non écrite en dur : une
# remédiation qui nommerait quatre étiquettes figées serait fausse dès qu'une
# organisation en déclare d'autres — et une remédiation fausse coûte plus cher
# qu'une remédiation absente, parce qu'elle est suivie.
required_tags_label(required) := concat(", ", [req.name | some req in required])

# snapshot_max_age_days / snapshot_max_age_ns — la fenêtre de fraîcheur d'une
# snapshot. Les deux formes existent parce que le message la cite en JOURS (ce
# qu'un humain règle) et que la comparaison se fait en nanosecondes (ce que
# time.parse_rfc3339_ns rend).
snapshot_max_age_days := object.get(_config_snapshots, "max_age_days", _default_snapshot_max_age_days)

snapshot_max_age_ns := ((snapshot_max_age_days * 24) * 3600) * 1000000000

# snapshot_accepted_states — les états NATIFS d'une snapshot réellement
# exploitable. Ancrés sur le contrat de chaque API, jamais devinés :
#   Outscale Snapshot.State ∈ in-queue|pending|completed|error|deleting
#     (osc-api/outscale.yaml, schéma Snapshot) → `completed` ;
#   Exoscale block-storage-snapshot.state ∈ partially-destroyed|destroying|
#     creating|created|promoting|error|destroyed|allocated
#     (openapi-v2.exoscale.com, schéma block-storage-snapshot) → `created`.
snapshot_accepted_states := object.get(_config_snapshots, "accepted_states", _default_snapshot_states)

# secret_min_confidence — le niveau de confiance minimal qu'une détection doit
# porter pour être signalée.
secret_min_confidence := object.get(_config_secrets, "min_confidence", _default_min_confidence)

# confidence_rank — rang ordonné d'un niveau de confiance. Un niveau inconnu vaut
# le rang le plus bas : une détection ne se perd pas parce qu'on n'a pas su lire
# son étiquette.
confidence_rank(level) := r if {
	ranks := {"low": 0, "medium": 1, "high": 2}
	r := object.get(ranks, level, 0)
}

# meets_confidence — la détection atteint le seuil configuré.
meets_confidence(level) if confidence_rank(level) >= confidence_rank(secret_min_confidence)

# ── Le profil PAR DÉFAUT, répliqué pour `opa test` (cf. commentaire d'en-tête) ──

_default_required_billable := [
	{"name": "CostCenter", "keys": ["billing", "billingcode", "cc", "costcenter"]},
	{"name": "Project", "keys": ["app", "application", "project", "service"]},
	{"name": "Env", "keys": ["env", "environment", "stage"]},
	{"name": "Owner", "keys": ["contact", "owner", "responsible", "team"]},
]

_default_required_network := [
	{"name": "Owner", "keys": ["contact", "owner", "responsible", "team"]},
	{"name": "Project", "keys": ["app", "application", "project", "service"]},
	{"name": "Env", "keys": ["env", "environment", "stage"]},
]

_default_tagged_types := [
	"blockstorage_snapshot",
	"blockstorage_volume",
	"compute_image",
	"compute_instance",
	"kubernetes_cluster",
	"load_balancer",
	"managed_database",
	"object_storage_bucket",
]

_default_snapshot_max_age_days := 7

_default_snapshot_states := ["completed", "created"]

_default_min_confidence := "low"
