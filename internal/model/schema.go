package model

import (
	"strconv"
	"strings"
)

// Le schéma de l'inventaire normalisé est un CONTRAT INTERNE, pas un détail
// d'implémentation.
//
// L'inventaire est déjà le point de passage de tout : la collecte y projette, les
// règles s'y évaluent, l'assessment en dérive, le bundle le scelle. Chaque nouvel
// usage le fige un peu plus PAR ACCIDENT, et une évolution du modèle casse alors un
// consommateur en silence. On le nomme donc, on le version, et on le gèle.
//
// # Ce qui est GARANTI
//
//   - L'enveloppe : un objet portant `provider` (chaîne) et `resources` (tableau).
//     Le scan y ajoute `evaluated_at` (RFC3339 UTC), l'instant d'évaluation unique
//     auquel les règles sensibles au temps s'ancrent, et `config`, la
//     configuration effective des contrôles à laquelle les règles réglables
//     s'ancrent. Les deux sont ÉCRITS UNE FOIS : un input.json rejoué garde les
//     siens, sans quoi le rejeu appliquerait l'horloge et la politique du jour à
//     un dossier d'hier.
//   - La ressource : `provider`, `type`, `id`, `name`, `attributes` toujours
//     présents ; `region` et `provenance` présents quand ils sont renseignés.
//   - `attributes` est une carte PLATE de nom d'attribut vers valeur JSON. Un nom
//     d'attribut est en snake_case et AGNOSTIQUE du fournisseur : c'est ce que les
//     règles lisent, et c'est pourquoi une règle est commune à tous les clouds.
//   - `provenance` est indexée par les MÊMES noms d'attributs, jamais imbriquée
//     dans une valeur. Une clé peut y exister sans que l'attribut soit dans
//     `attributes` : c'est un champ cherché et non exposé par la source.
//   - Un type de ressource est en snake_case, au singulier, préfixé par sa famille
//     de service neutre (compute_, network_, objectstorage_ côté contrôle, et côté
//     ressource : compute_instance, security_group_rule, object_storage_bucket…).
//   - Un attribut ABSENT n'est jamais forcé à une valeur : « non collecté » et
//     « collecté à faux » ne se confondent pas. C'est l'invariant dont tout le
//     modèle de confiance dépend.
//
// # Ce qui n'est PAS garanti
//
//   - L'ORDRE des ressources et des attributs : rien ne le fixe, et s'en servir
//     casserait au premier changement de pagination.
//   - La PRÉSENCE d'un attribut donné sur une ressource donnée : elle dépend du
//     fournisseur, des droits du jeton et de la source (un plan Terraform ignore
//     tout de l'état effectif). C'est précisément ce que `provenance` documente.
//   - L'exhaustivité de l'inventaire : un scan mesure ce à quoi ses identifiants
//     donnent accès, jamais « tout le tenant ».
//   - Les valeurs elles-mêmes : elles reflètent le contrat natif du fournisseur,
//     qui peut changer sans que Pépin en décide.
//
// # Comment il bouge
//
// La forme est gelée dans cmd/testdata/frozen/inventory.json, avec l'énumération
// des types et de leurs attributs communs. Un changement de forme — un champ
// d'enveloppe, un champ de ressource, un type, un attribut — fait rougir le gel :
// il se décide, il incrémente `InventoryFormat`, et il s'écrit au CHANGELOG. C'est
// volontairement le même égard que pour la surface CLI.

// InventoryFormat identifie le schéma de l'inventaire normalisé, version comprise
// (`/vN`). Il VOYAGE avec le bundle de preuve (manifest.inventory_schema) : un
// consommateur qui rencontre une version qu'il ne connaît pas doit s'arrêter
// plutôt que deviner la forme de ce qu'il lit.
// v2 : l'enveloppe porte `collection`, l'état de ce que la collecte a pu lire
// (unités tentées, complètes ou non, avec la classe de leur échec ; types de la
// source qu'aucune spec ne projette). Ajout PUR — aucun champ existant ne bouge —
// mais un ajout que le contrat doit annoncer : un consommateur qui rejoue un
// inventaire sans lire `collection` conclurait plus fermement que Pépin ne l'a
// fait, ce qui est précisément l'erreur que le champ existe pour empêcher.
//
// v3 : la ressource porte `source`, l'origine du code d'infrastructure qui la
// déclare (fichier, ligne, module). Présente sur un plan Terraform quand les
// sources HCL ont pu être lues, ABSENTE partout ailleurs — une collecte live ne
// sait pas d'où vient une ressource, et rien n'y est inventé.
// v4 : deux ajouts, tous deux PURS.
//   - L'enveloppe porte `config`, la configuration EFFECTIVE des contrôles sous
//     laquelle l'inventaire a été évalué (réglages d'étiquetage, de fraîcheur de
//     snapshot, de détection de secrets). Elle voyage AVEC l'inventaire pour la
//     même raison que `evaluated_at` : un input.json rejoué doit rendre le même
//     verdict, or un verdict dépend désormais aussi des réglages. Un consommateur
//     qui l'ignore lirait un résultat sans savoir sous quelle exigence il a été
//     rendu — ce que ce champ existe précisément pour empêcher.
//   - Une `blockstorage_snapshot` porte `state`, l'état natif qui dit si la
//     snapshot est terminée (Outscale Snapshot.State, Exoscale
//     block-storage-snapshot.state).
//
// v5 : une `access_key` porte `creation_date`, la date d'émission native
//
//	(osc-sdk-go v2.24.0 AccessKey.CreationDate, vérifiée dans model_access_key.go).
//	Ajout PUR — aucun champ existant ne bouge. Il est nécessaire parce que la
//	ROTATION ne se lit nulle part : CLD-IAM-2 demande « une expiration ET une
//	rotation », l'expiration se lit sur un champ, la rotation se déduit de l'âge.
//	Un consommateur qui l'ignore ne perd rien de ce qu'il lisait déjà.
//
// v6 : une `security_group_rule` ENTRANTE porte `peer_security_group_ids`, les groupes
//
//	dont les membres sont admis comme source (osc-sdk-go v2.24.0
//	SecurityGroupRule.SecurityGroupsMembers, vérifié dans
//	model_security_groups_member.go). Ajout PUR — aucun champ existant ne bouge.
//	Une règle entrante admet une source par CIDR OU par groupe ; sans ce champ,
//	« pas de CIDR » ne dit pas si la règle n'admet personne ou si elle admet un
//	groupe qu'on n'a pas lu. Le contrôle du SG « default » a besoin de la
//	différence pour nommer ce qu'il a trouvé, et la DÉDUIRE d'une absence serait la
//	fabrication que l'ADR-0014 refuse. Mappé sur l'entrant seul : rien ne lit la
//	contrepartie sortante.
//
// v7 : un nouveau type, `network_interface` — une ressource par carte réseau, portant
//
//	`nic_id`, `vm_id`, `public_ip` et `security_group_ids` (contrat osc-sdk-go
//	v2.24.0 Vm.Nics []NicLight). Ajout PUR : aucun type ni attribut existant ne bouge.
//	Il est nécessaire parce que l'exposition est une propriété de la CARTE, et que
//	l'aplatir sur la machine perd l'APPARIEMENT — une carte publique au groupe fermé
//	plus une carte privée au groupe ouvert se lisent, une fois réunies, comme
//	« publique et ouverte ». Un consommateur qui l'ignore lit ce qu'il lisait déjà.
//
//	La même version porte un second ajout PUR : `governance_provider` peut porter
//	`secnumcloud_regions`, le périmètre de régions que la qualification COUVRE. Il
//	est là parce qu'une qualification ne porte pas sur un fournisseur entier :
//	celle d'Outscale couvre `cloudgouv-eu-west-1` et elle seule, et un tenant en
//	`eu-west-2` lisait « qualifié » à son sujet. L'attribut `secnumcloud` est
//	désormais rendu POUR la région scannée (`qualifie` | `hors_perimetre` |
//	`perimetre_inconnu`), et le périmètre voyage à côté pour qu'un consommateur
//	puisse le vérifier plutôt que de croire le statut sur parole.
//
// v8 : une `compute_instance` peut porter `public_interface`, le fait qu'elle ait ou
//
//	non une interface PUBLIQUE (Exoscale : public-ip-assignment none|inet4|dual, API
//	v2 Compute get-instance). Ajout PUR. Il est nécessaire parce que, chez un
//	fournisseur dont les groupes de sécurité filtrent l'interface publique, une
//	instance qui n'en a pas se voit attacher une liste vide PAR CONSTRUCTION : le
//	contrôle « VM sans groupe de sécurité » criait alors sur chaque instance privée,
//	avec une remédiation que l'API ignore. Déduire le fait d'une absence de
//	`public_ip` ne distinguerait pas « privée » de « non collectée ».
//
// v9 : une ressource peut porter `provider_managed`, le fait qu'elle soit créée,
//
//	utilisée et supprimée par la PLATEFORME pour un composant que l'exploitant ne
//	fait pas tourner (Exoscale : le rôle `sks-ccm-<cluster>` du cloud controller
//	manager, GET /v2/iam-role). Ajout PUR. Il est nécessaire parce que ce rôle est
//	`editable: true` : le filtre des rôles prédéfinis ne le couvrait pas, et chaque
//	cluster apportait deux écarts IAM que personne ne pouvait faire disparaître. Le
//	fait est déclaré par le descripteur, avec sa source ; aucune règle ne connaît de
//	convention de nommage.
//
// v10 : la collecte live Scaleway produit le type `security_group`, porteur de
//
//	`inbound_default_policy` et `outbound_default_policy` (accept|drop), de sa
//	description et de ses étiquettes. Ajout PUR : aucun type existant ne change.
//	Il est nécessaire parce que `network_securitygroup_default_deny` — sévérité
//	`high` — revenait « non évalué » en live alors que sa donnée était DÉJÀ sur le
//	fil : le collecteur paginait la même liste pour joindre les règles à leur
//	groupe, sans jamais projeter le champ. Une lacune de couverture, pas une lacune
//	d'API (issue #195). Contrat vérifié contre l'API réelle le 2026-09-11.
//
// v11 : une `access_key` peut porter `owner_user_id` et `owner_application_id`, et un
//
//	`iam_user` son `user_type` (`owner` | `member`). Ajout PUR. Il est nécessaire parce
//	que `iam_no_root_access_key` ne savait pas conclure en live chez Scaleway : la
//	donnée était dans la liste de clés DÉJÀ interrogée, et personne ne la projetait.
//	Les deux champs d'appartenance vont ensemble — une clé appartient à un
//	utilisateur OU à une application, et ne projeter que le premier ferait taire le
//	contrôle sur les clés d'application, dont l'appartenance est pourtant observée.
//	`user_type` est le SEUL marqueur du propriétaire de l'organisation :
//	`account_root_user_id`, porté par le même enregistrement, désigne autre chose
//	(mesuré le 2026-09-11). Contrat vérifié contre l'API réelle.
const InventoryFormat = "pepin-inventory/v11"

// InventorySchemaVersion extrait le N du suffixe `/vN`. Comme pour le bundle, la
// constante et le signal sur le fil sont la même chose : impossible de faire
// diverger la version déclarée de la version publiée.
func InventorySchemaVersion() int {
	i := strings.LastIndex(InventoryFormat, "/v")
	if i < 0 {
		return 0
	}
	n, err := strconv.Atoi(InventoryFormat[i+2:])
	if err != nil {
		return 0
	}
	return n
}
