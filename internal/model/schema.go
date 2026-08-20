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
const InventoryFormat = "pepin-inventory/v4"

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
