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

# is_public_cidr — le CIDR couvre l'Internet public (IPv4 ou IPv6).
is_public_cidr(cidr) if cidr == "0.0.0.0/0"

is_public_cidr(cidr) if cidr == "::/0"

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
# Outscale `in-use`, Exoscale `attached`. Un état inconnu n'est pas considéré en
# usage (pas de faux positif).
volume_in_use(v) if object.get(v.attributes, "state", "") in {"in-use", "attached"}

# covers_port — la règle de filtrage couvre le port p (schéma SG commun :
# port_from/port_to). Plage [port_from, port_to] ; port_to absent ou < port_from
# (ex. règle à port unique, to non renseigné) ⇒ borne haute = port_from.
covers_port(rule, p) if {
	from := object.get(rule, "port_from", 0)
	to := max([from, object.get(rule, "port_to", from)])
	from <= p
	p <= to
}

# proto_covers — la règle couvre le protocole `want` (ou « all »). Schéma SG
# commun : protocol ∈ tcp|udp|icmp|all.
proto_covers(rule, want) if lower(object.get(rule, "protocol", "")) == want

proto_covers(rule, _) if lower(object.get(rule, "protocol", "")) == "all"

# sg_inbound_from_internet — règle entrante acceptante dont la source couvre
# Internet ; renvoie true si au moins un CIDR public est présent.
sg_inbound_from_internet(rule) if {
	lower(object.get(rule, "direction", "")) == "inbound"
	object.get(rule, "action", "accept") != "drop"
	some cidr in object.get(rule, "cidrs", [])
	is_public_cidr(cidr)
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

# required_tags — étiquettes de gouvernance obligatoires.
required_tags := ["CostCenter", "Project", "Env", "Owner"]

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
	"exoscale": {"de-fra", "de-muc", "at-vie", "bg-sof", "hr-zag", "de-fra-1", "de-muc-1", "at-vie-1", "at-vie-2", "bg-sof-1", "hr-zag-1"},
}

_trusted_regions := {
	"scaleway": set(),
	"outscale": set(),
	"exoscale": {"ch-gva-2", "ch-dk-2"},
}

_noneu_regions := {
	"scaleway": set(),
	"outscale": {"us-east-2", "us-west-1", "ap-northeast-1", "cn-southeast-1"},
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
