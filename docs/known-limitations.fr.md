> [🇬🇧 English](known-limitations.md) · 🇫🇷 Français

# Limites connues et angles morts

Pour un scanner de posture, une limite connue fait partie du contrat de confiance. Une limite
tue se découvre au pire moment : pendant un audit.

Cette page dit ce que Pépin **ne sait pas** mesurer, et pourquoi. Elle porte sur la **v0.2.0**
et se régénère avec le code : les tableaux ci-dessous sont calculés depuis le référentiel, les
descripteurs de fournisseurs et le verrou du `pass` (voir
[Comment cette page reste vraie](#comment-cette-page-reste-vraie)).

## Les cinq catégories

| # | Catégorie | Qui peut la lever |
|---|---|---|
| 1 | **API du fournisseur** : l'API n'expose pas le champ | le fournisseur |
| 2 | **Provider Terraform** : le schéma n'a pas l'attribut, ou il est `known after apply` | HashiCorp et les mainteneurs du provider Terraform |
| 3 | **Limitation Pépin** : la donnée existe et est joignable, mais n'est pas collectée ou pas câblée | ce projet |
| 4 | **Non observable techniquement** : la propriété ne vit pas là où une API peut la voir | personne, par construction |
| 5 | **Exigence organisationnelle ou contractuelle** : l'exigence ne porte pas sur la configuration | vous, avec des documents, pas avec un scanner |

Les catégories 1, 2 et 4 ressortent en `not-applicable` (justifié) ou en `not-evaluated` (avec
un motif). La catégorie 3 ressort en `not-evaluated` et c'est celle qui se réduit avec le temps.
La catégorie 5 n'a aucun statut, parce que ces exigences ne sont pas au référentiel actif.

## 1 et 4 : déclarés non applicables, avec justification

Chaque entrée ci-dessous vient d'un contrat de fournisseur (`providers/<nom>.yaml`, `contrat`).
Le scan rend `not-applicable` et porte la justification dans `waiver.justification`, citée mot
pour mot. Un N/A non justifié n'est jamais produit.

<!-- pepin:gen not-applicable-list -->
| Contrôle | Fournisseur | Justification consignée au contrat |
|---|---|---|
| `blockstorage_snapshot_not_public` | exoscale | Snapshots block-storage Exoscale non exportables/partageables (doc officielle) : le risque d'exposition publique est structurellement absent, conforme par construction (STO-2). |
| `blockstorage_snapshot_not_public` | scaleway | Les snapshots block Scaleway (api/block/v1) n'exposent aucun mécanisme de partage ou d'export public : le risque d'exposition publique est structurellement absent, conforme par construction (STO-2). |
| `blockstorage_volume_encryption` | outscale | osc-sdk-go/v2 Volume n'expose aucun champ de chiffrement ; le chiffrement au repos est côté invité (EncFS/LUKS), responsabilité du client → non observable côté plateforme (CHF-2). |
| `blockstorage_volume_encryption` | scaleway | Chiffrement au repos des volumes block côté invité (LUKS/Cryptsetup), responsabilité du client (responsabilité partagée) ; l'API block n'expose aucun champ de chiffrement → non observable côté plateforme (CHF-2). |
| `iam_user_mfa_enabled` | outscale | type de ressource « iam_user » absent de l'API outscale |
| `loadbalancer_http_redirect_to_https` | exoscale | type de ressource « load_balancer » absent de l'API exoscale |
| `loadbalancer_http_redirect_to_https` | outscale | Le LBU Outscale ne peut pas rediriger : `ListenerRule.Action` est documenté « always forward » au contrat OAPI (aucune action de redirection), et aucun attribut de redirection n'existe sur `Listener`. Le mécanisme est inexistant → contrôle non applicable (CHF-1). |
| `loadbalancer_logging_enabled` | exoscale | type de ressource « load_balancer » absent de l'API exoscale |
| `loadbalancer_ssl_listeners` | exoscale | type de ressource « load_balancer » absent de l'API exoscale |
| `objectstorage_bucket_kms_encryption` | exoscale | SOS chiffre au repos par défaut (SSE-SOS, clés gérées par Exoscale, type SSE-S3) mais n'expose pas de BYOK/KMS géré par le client au niveau bucket (SSE-C reste par-objet, non observable) → le contrôle BYOK-au-bucket est sans objet (CHF-4). |
| `objectstorage_bucket_kms_encryption` | outscale | OOS chiffre côté serveur en AES256 avec une clé FOURNISSEUR ; il n'existe ni service KMS ni clé maître gérée par le client, donc pas de BYOK à auditer au niveau bucket (CHF-4). NB : l'activation du SSE elle-même est opt-in et observable — elle relève d'un contrôle distinct, pas de ce N/A. |
<!-- /pepin:gen not-applicable-list -->

La plus récurrente mérite son paragraphe. **Le chiffrement au repos des volumes block n'est pas
observable chez Outscale ni chez Scaleway** : il se fait dans l'invité (LUKS/cryptsetup), sous
la responsabilité du client dans le partage de responsabilités, et l'API block n'expose aucun
champ de chiffrement. Chez Exoscale, il est transparent et conforme par construction. Aucun
contournement au niveau de l'API ; s'il vous faut une preuve, produisez-la depuis l'intérieur de
l'instance.

## 3 : les contrôles qui ne peuvent aujourd'hui jamais atteindre `pass`

Ces contrôles sont déclarés pour au moins un fournisseur, et pourtant **aucun fournisseur ni
aucune source** ne peut actuellement lever les quatre verrous du `pass`. Ils peuvent encore
émettre un `fail` quand une règle se déclenche, mais ils ne confirmeront jamais une conformité.

<!-- pepin:gen never-pass -->
| Contrôle | Sévérité | Motif |
|---|---|---|
| `governance_resource_required_tags` | medium | aucun type de ressource visé et le contrôle ne lit pas le descripteur du fournisseur : le verrou du « pass » ne peut pas être levé, le scan rend « not-evaluated » tant qu'aucun écart n'est détecté |
<!-- /pepin:gen never-pass -->

**Conséquence.** Un tenant qui satisfait réellement l'un de ces contrôles obtient tout de même
`not-evaluated`, et non `pass`. C'est la réponse honnête, et c'est aussi un écart à combler : le
correctif est dans `internal/assess` et dans les collecteurs, pas dans la documentation.

**Contournement.** Aucun dans Pépin. Ne construisez pas une porte de CI qui attend de ces
contrôles qu'ils passent au vert.

## 2 et 3 : observables par une seule source

Le référentiel déclare ces contrôles pour le fournisseur, mais une seule des deux sources sait
réellement les décider. Le motif indiqué est celui qui s'applique à la source qui ne le peut
pas.

<!-- pepin:gen single-source -->
| Contrôle | Fournisseur | Observable uniquement via | Motif |
|---|---|---|---|
| `blockstorage_snapshot_not_public` | outscale | live | cette source ne produit aucune ressource de type « blockstorage_snapshot » |
| `blockstorage_volume_snapshots_exist` | outscale | live | cette source ne produit aucune ressource de type « blockstorage_volume » |
| `compute_instance_deletion_protection` | outscale | live | attribut décisif « deletion_protection » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `compute_instance_no_secrets_in_user_data` | scaleway | terraform | attribut décisif « user_data » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `compute_instance_public_ip_with_open_securitygroup` | scaleway | live | attribut décisif « public_ip » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `database_backup_enabled` | scaleway | terraform | cette source ne produit aucune ressource de type « managed_database » |
| `database_encryption_at_rest_enabled` | scaleway | terraform | cette source ne produit aucune ressource de type « managed_database » |
| `database_service_not_open_to_internet` | scaleway | terraform | cette source ne produit aucune ressource de type « managed_database » |
| `governance_resource_region_in_eu` | outscale | live | attribut décisif « region » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `iam_accesskey_expiration_set` | outscale | live | cette source ne produit aucune ressource de type « access_key » |
| `iam_account_mfa_enforced` | outscale | live | cette source ne produit aucune ressource de type « api_access_policy » |
| `iam_apiaccesspolicy_max_key_expiration` | outscale | live | cette source ne produit aucune ressource de type « api_access_policy » |
| `iam_apiaccessrule_defined` | outscale | live | cette source ne produit aucune ressource de type « api_access_summary » |
| `iam_apiaccessrule_no_public_cidr` | outscale | live | cette source ne produit aucune ressource de type « api_access_rule » |
| `iam_no_root_access_key` | outscale | live | cette source ne produit aucune ressource de type « access_key » |
| `iam_policy_no_privilege_escalation` | scaleway | terraform | cette source ne produit aucune ressource de type « iam_policy » |
| `iam_user_mfa_enabled` | exoscale | live | cette source ne produit aucune ressource de type « iam_user » |
| `iam_user_mfa_enabled` | scaleway | live | cette source ne produit aucune ressource de type « iam_user » |
| `kubernetes_cluster_auto_upgrade_enabled` | outscale | live | cette source ne produit aucune ressource de type « kubernetes_cluster » |
| `kubernetes_cluster_control_plane_highly_available` | outscale | live | cette source ne produit aucune ressource de type « kubernetes_cluster » |
| `kubernetes_cluster_deletion_protection` | outscale | live | cette source ne produit aucune ressource de type « kubernetes_cluster » |
| `kubernetes_cluster_not_publicly_accessible` | outscale | live | cette source ne produit aucune ressource de type « kubernetes_cluster » |
| `loadbalancer_logging_enabled` | outscale | live | cette source ne produit aucune ressource de type « load_balancer » |
| `loadbalancer_ssl_listeners` | outscale | live | cette source ne produit aucune ressource de type « load_balancer » |
| `network_peering_cross_organization` | outscale | live | cette source ne produit aucune ressource de type « network_peering » |
| `network_securitygroup_default_deny` | scaleway | terraform | cette source ne produit aucune ressource de type « security_group » |
| `network_securitygroup_default_restrict_traffic` | outscale | live | attribut décisif « security_group_name » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `objectstorage_bucket_default_encryption` | outscale | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_kms_encryption` | scaleway | live | attribut décisif « sse_kms_enabled » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `objectstorage_bucket_object_lock_enabled` | exoscale | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_object_lock_enabled` | outscale | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_public_access` | exoscale | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_public_access` | outscale | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_versioning_enabled` | exoscale | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_versioning_enabled` | outscale | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_versioning_enabled` | scaleway | live | attribut décisif « versioning » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
<!-- /pepin:gen single-source -->

Deux asymétries structurelles expliquent la plupart de ces lignes.

**La collecte live porte toujours la région**, ce que ne fait pas un plan Terraform, sauf si un
mapping en nomme le champ. `governance_resource_region_in_eu` est donc décidable en live et
`not-evaluated` sur la plupart des plans.

**Un plan Terraform omet tout ce qui est `known after apply`.** C'est une limite de la source,
pas un défaut de Pépin, et elle a une conséquence qui mérite d'être nommée : sur un plan, une
instance réellement dépourvue de groupe de sécurité est **indistinguable** d'une instance dont
le groupe de sécurité est créé par le même plan, puisque `security_group_id` n'est alors pas
dans `planned_values`. Pépin traite l'attribut comme *non collecté* : le contrôle rend
`not-evaluated` avec son motif, plutôt qu'un `fail` deviné sur une machine qui est en réalité
rattachée. C'est exactement le rôle du verrou décrit dans
[le modèle d'assessment](concepts/assessment-model.fr.md) : mieux vaut ne pas conclure que
conclure faux, dans un sens comme dans l'autre.

**Contournement :** auditer en `--live` la configuration effective, ou référencer dans le plan
une valeur littérale connue au stade plan quand c'est le cas d'usage.

## La photographie complète

Par fournisseur et par source, sur l'ensemble des contrôles du référentiel :

<!-- pepin:gen coverage-totals -->
| Fournisseur | Source | ✅ `supported` | ◐ `partial` | ∅ `not-applicable` | ✗ `unsupported` |
|---|---|---:|---:|---:|---:|
| exoscale | terraform | 21 | 1 | 5 | 30 |
| exoscale | live | 25 | 1 | 5 | 26 |
| outscale | terraform | 17 | 4 | 4 | 32 |
| outscale | live | 39 | 1 | 4 | 13 |
| scaleway | terraform | 18 | 6 | 2 | 31 |
| scaleway | live | 16 | 3 | 2 | 36 |
| kubernetes | live | 4 | 0 | 0 | 53 |
<!-- /pepin:gen coverage-totals -->

Le détail par contrôle, avec le motif de chaque case qui n'est pas pleinement supportée, est la
[matrice de couverture](coverage.fr.md).

## Preuves de remédiation

Au-delà de la remédiation **textuelle** portée par chaque finding (garantie, elle, par
`TestEveryFindingCarriesRemediation`), le dépôt vise une **preuve** de remédiation par couple
(contrôle, fournisseur) sous `references/remediation/` : un module Terraform autonome, ou une
note documentée. À ce jour :

<!-- pepin:gen remediation-coverage -->
| Fournisseur | Preuves de remédiation |
|---|---:|
| exoscale | 26 / 26 |
| kubernetes | 0 / 4 |
| outscale | 0 / 40 |
| scaleway | 0 / 25 |
| **Total** | **26 / 95** |
<!-- /pepin:gen remediation-coverage -->

Ce contrôle n'est **délibérément pas** branché sur `mise run validate` : tous fournisseurs
confondus le compte reste partiel, et une porte perpétuellement rouge est une porte qu'on
apprend à ignorer. Exoscale est le premier fournisseur à 100 %, et un test tient cet acquis :
`TestExoscaleRemediationCoverageStaysComplete` échoue dès qu'un contrôle exoscale arrive sans
sa preuve. Les autres fournisseurs rejoindront cette garde en atteignant 100 %.
`mise run check-remediation` donne la liste par contrôle.

**Conséquence pour vous :** chaque finding vous dit quoi faire, en prose. Tous les findings ne
sont pas accompagnés d'un module Terraform testé qui le prouve.

## Le contrat de véracité, et ce qu'il doit encore

Pour un scanner de posture, l'unité qui compte n'est pas `fixture → Rego → FAIL attendu`. C'est
la chaîne entière : **réponse d'API ou plan → collecteur → normalisation → garde de capacité →
Rego → assessment → verdict**. L'incident des politiques EIM inline l'a prouvé : une policy
accordant `Action: "*"` échappait à *tous* les contrôles `iam_policy_*`, la règle Rego n'était
pas fausse, et la donnée n'arrivait simplement jamais jusqu'à elle. Un test Rego parfait serait
resté vert pendant que le scanner produisait un faux vert.

Un scénario de véracité s'exécute donc contre le **binaire**, sur toute la chaîne, et lit le
statut que publie l'assessment. Chaque chemin — contrôle × fournisseur × source — doit prouver
les verdicts qu'il peut réellement atteindre : trois pour un chemin qui sait conclure (`fail`,
`pass`, `not-evaluated`), un pour un chemin qui ne peut pas lever le verrou du `pass`, un pour
un chemin que le contrat du fournisseur déclare non applicable.

Ce qui n'est pas encore prouvé est **compté**, pas masqué :

<!-- pepin:gen veracity-debt -->
| Chiffre | Nombre |
|---|---:|
| Chemins contrôle × fournisseur × source sur lesquels Pépin conclut | 178 |
| Chemins dont tous les verdicts atteignables sont prouvés de bout en bout | 5 |
| Verdicts à prouver au total | 458 |
| Verdicts restant à prouver | 445 |
<!-- /pepin:gen veracity-debt -->

Le reste est listé chemin par chemin dans `internal/veracity/testdata/debt.txt`. Ce registre est
une porte dans les deux sens : une obligation non prouvée qui n'y figure *pas* fait échouer la
construction — un contrôle ajouté sans ses scénarios ne peut donc pas passer — et une ligne qui
n'est plus due la fait échouer aussi.

**Pourquoi une dette comptée plutôt qu'une matrice verte.** Quatre scénarios par chemin
feraient quelque sept cents cas. Personne ne peut *éprouver* sept cents cas — c'est-à-dire les
casser un par un pour vérifier qu'ils rougissent —, et une matrice engendrée par gabarit serait
exactement le faux vert que ce scanner existe pour combattre : un test qui passe parce que ses
cas sont creux ne mesure que lui-même.

## Limites de l'outil lui-même

### Un bundle de preuve non caviardé est un artefact sensible

`--seal` embarque l'inventaire évalué **brut** dans `input.json` : user-data (là même où Pépin
trouve les secrets en dur), documents de politique IAM, policies de bucket. Pépin l'annonce sur
stderr au moment du scellement.

- **Contournement :** `--seal --redact` remplace les valeurs d'attributs sensibles par des
  empreintes. Le finding reste, la valeur part.
- **Contrepartie :** un bundle caviardé est **incompatible avec `verify --re-derive`**, la
  détection ne pouvant pas rejouer sur de la donnée caviardée. Un bundle partagé s'appuie alors
  sur la signature cosign.

### Un plan Terraform n'est pas un état

`--terraform` audite l'état **planifié**. Il ne dit rien de la dérive, rien des ressources
créées hors du code, rien des attributs encore inconnus au stade plan. C'est pour cela que le
verdict dit « périmètre déclaré (plan Terraform, état planifié) » et non « conforme ». Pour la
configuration effective, utilisez `--live`.

### `--live` voit exactement ce que voient vos identifiants

Une permission manquante au rôle en lecture seule se traduit par un `not-evaluated`, pas par une
erreur. C'est le bon mode de défaillance, et depuis la v0.3.0 c'est aussi un mode **bruyant** :
le scan enregistre chaque unité de collecte qu'il n'a pas pu lire, imprime un relevé de
capacités avant tout verdict, nomme l'unité manquante comme motif de chaque contrôle touché, et
ne rend pas `0`. Un scan sous-privilégié ne produit donc plus un rapport plus silencieux qu'un
scan privilégié : il produit un rapport qui dit ce qu'il n'a pas pu voir. Voir
[le code de sortie 3](reference/exit-codes.fr.md#3--le-scan-nétablit-pas-la-conformité).

Ce qui reste vrai : Pépin ne peut pas vous dire ce qu'un identifiant *plus large* aurait trouvé.
Il rend compte de la surface qu'on lui a donnée, et de l'absence de cette surface, jamais de ce
qui s'étend au-delà.

### Les permissions minimales sont documentées, pas mesurées

Chaque page de fournisseur liste les appels d'API que fait un scan et les permissions de lecture
qu'ils exigent. Ces listes sont **dérivées des descripteurs de collecte**, c'est-à-dire des
endpoints que la spec déclare, et confrontées à la documentation IAM publique du fournisseur.
Chaque page distingue les lignes confirmées par la documentation de celles qui restent non
vérifiées.

Deux moitiés, et elles ne sont pas dans le même état. La moitié **endpoints** est désormais
mesurée : une session enregistrée prouve que le collecteur émet bien ce que le descripteur
déclare, donc la liste n'est plus une affirmation sur du code que personne n'a regardé tourner
([Tracer les appels réels](guides/tracing-api-calls.fr.md)). La moitié **droits** ne l'est pas,
et ne peut pas l'être d'ici : confirmer que `InstancesReadOnly` et rien de plus suffit à
`ListSecurityGroups` exige de lancer un scan avec un rôle délibérément réduit contre un tenant
réel. Ce dépôt ne détient aucun identifiant cloud, par construction, et aucun contrôle
automatisé n'atteint une API de fournisseur. Chez Outscale, l'essentiel des lignes est
`a_verifier` pour une raison plus tranchée encore : les noms d'action y sont **inférés** de la
syntaxe EIM documentée, et non recopiés d'un catalogue de permissions publié. Un droit inféré
est une supposition, et c'est précisément ce qu'un CSPM ne doit pas vous demander d'accorder.

### Les faits de souveraineté sont déclarés, pas vérifiés

`governance_provider_sovereignty` évalue le bloc `souverainete` du descripteur du fournisseur :
siège, contrôle capitalistique, statut SecNumCloud, exposition extraterritoriale, avec leurs
sources. Ce sont des **attestations transcrites depuis des sources publiques**, pas des mesures.
La preuve du résultat le dit, et le contrôle est exclu du décompte « quelque chose a été
mesuré » qui pilote le code de sortie `3`.

### `evidence.proves` est toujours vide

L'objet `evidence` de l'assessment porte un champ `proves`, un tableau de trois cases hérité du
module partagé `scankit` (`[running, persistent, reboot-survivable]`). Cette notion appartient
au durcissement d'hôte, pas à la posture cloud : Pépin ne la renseigne jamais, et elle se
sérialise en `["", "", ""]`. Les consommateurs doivent l'ignorer, ce n'est pas un signal Pépin.

### La colonne « live » est dérivée ; seule sa plomberie est observée

La matrice de couverture est **calculée depuis les descripteurs** : elle dit ce que la spec de
collecte live et le mapping Terraform d'un fournisseur sont déclarés projeter, et ce que le
contrat d'API marque comme vérifié. Aucun contrôle automatisé de ce dépôt n'appelle l'API d'un
fournisseur.

Depuis la v0.4.0, une moitié de cette affirmation n'est plus une simple déclaration. Une session
enregistrée d'appels HTTP réels, prise contre un **émulateur local**, `feint serve`, sans le
moindre identifiant cloud, est rejouée à chaque build
(`internal/genprovider/testdata/transcripts/`, [Tracer les appels réels](guides/tracing-api-calls.fr.md)).
Elle mesure que les endpoints annoncés par un descripteur partent réellement sur le réseau, que
les jointures parent/enfant tirent sur un identifiant lu dans la réponse du parent, et que les
paramètres de pagination sont ceux qui sont déclarés. Un endpoint déclaré et jamais émis casse
désormais le build.

Ce que cet enregistrement n'établit **pas**, et qu'une case verte « live » ne signifie donc
toujours pas :

- que le fournisseur rende le champ sous ce nom et ce type : le contrat natif se lit dans le
  SDK, il ne se mesure pas ici ;
- que ses bornes réelles de pagination correspondent aux bornes déclarées ;
- comment il se comporte sous limitation de débit ;
- qu'il refuse par un `403` plutôt que par un `200` accompagné d'un corps d'erreur.

Une case verte signifie donc « ce descripteur projette l'attribut décisif, le contrat est marqué
vérifié, et l'appel qui irait le chercher est démontrablement émis ». Elle ne signifie pas
« une API a rendu ce champ lors d'une exécution mesurée ». Si les deux divergeaient sur votre
tenant, le scan le dirait par un `not-evaluated` accompagné de son motif, jamais par un vert
silencieux.

### Un émulateur n'est pas le fournisseur

Les enregistrements ci-dessus viennent de `feint`, un émulateur local de Scaleway, Outscale et
Exoscale. La distinction qu'il impose mérite d'être posée seule, parce que la confondre
fabriquerait exactement la fausse confiance que cette page existe pour empêcher : **un émulateur
prouve ce que Pépin fait, pas ce que le cloud répond.**

Une conséquence est structurelle, et non accidentelle. L'émulateur **accepte n'importe quel
identifiant**. C'est mesuré : sans en-tête d'authentification il rend `200`, avec un jeton bidon
il rend `200`, et il n'expose aucune injection de panne. Il ne peut donc jamais produire de
`403`, et aucun enregistrement pris contre lui ne peut mesurer la classification d'un vrai refus
de droits. Cette classification est éprouvée par `internal/collect/status_test.go` contre une
socket qui refuse vraiment ; ce qui reste inconnu, c'est de savoir si tel fournisseur refuse
bien avec ce statut.

Il en va de même de chaque valeur d'un enregistrement : elles sont frappées par l'émulateur au
démarrage, pas rendues par un cloud. C'est exactement pour cela qu'elles sont sûres à committer,
et exactement pour cela qu'elles ne prouvent rien du contrat d'un fournisseur.

### Le fichier et la ligne d'un finding exigent les sources `.tf` à côté du plan

`terraform show -json` ne porte ni fichier ni ligne : la représentation `configuration` d'une
ressource contient `address`, `type`, `name` et `expressions`, et rien sur le document dont elle
provient. Pépin *mesure* donc l'origine dans les `.tf` posés à côté du plan. Trois conséquences,
et aucune n'est contournée :

- Un plan produit ailleurs et transmis seul obtient un **module**, sans fichier ni ligne.
- Un module tiré d'un registre ou d'un dépôt git n'a aucun répertoire dans l'arbre de travail :
  ses ressources gardent leur module, et rien de plus.
- Un en-tête `resource "type" "nom"` écrit sur plusieurs lignes, ou déclaré dans un fichier
  `.tf.json`, n'est pas trouvé ; l'origine est alors absente plutôt qu'approximative.

Aucun drapeau ne permet de désigner un autre arbre de sources, et c'est délibéré : résoudre un
en-tête de bloc contre un arbre d'où le plan ne vient pas produirait une ligne plausible et
fausse, ce qui est pire que pas de ligne du tout.

### Rien n'est mesuré entre deux runs

Un résultat décrit un instant. Pépin n'a ni agent, ni mode veille, ni historique. La posture
continue est un problème d'ordonnancement que vous résolvez en CI, et les bundles scellés sont
ce que vous conservez.

### Une snapshot récente n'est pas une politique de sauvegarde

`blockstorage_volume_snapshots_exist` mesure une chose : un volume en usage a-t-il, dans la
fenêtre configurée (7 jours par défaut), au moins une snapshot dont l'état natif la dit
terminée ? L'état est vérifié, pas seulement la date.

**Quel état signifie « terminée » est lu, pas observé.** `completed` chez Outscale et `created`
chez Exoscale viennent des schémas OpenAPI publiés par ces fournisseurs, cités dans les
contrats. Aucune snapshot dans un état terminal n'a jamais été vue par ce dépôt, et les
enregistrements contre l'émulateur ne comblent pas ce trou non plus : il rend une liste vide
pour ces deux types, donc aucune valeur d'état ne les a traversés. Si un fournisseur rendait un
jour une valeur hors de son propre énuméré, le contrôle ne conclurait pas, car le verrou porte
sur la valeur et non sur son absence ; mais rien ici ne remarquerait la dérive avant un scan
réel.

Il ne prouve pas que la snapshot soit **restaurable** — aucune restauration n'est tentée —,
qu'elle soit **complète** au sens applicatif, qu'une **rétention** soit respectée (une seule
snapshot satisfait le contrôle), ni qu'une **politique de sauvegarde** existe. Symétriquement,
un volume sauvegardé autrement — sauvegarde applicative, réplication, service de backup du
fournisseur, outil externe — est signalé ici : un faux positif assumé, à traiter par une
dérogation datée plutôt qu'en désactivant le contrôle.

Ce contrôle ne participe donc à aucune affirmation « sauvegarde conforme ». Son libellé le dit,
dans les deux langues, et le référentiel le consigne dans la description du contrôle.

### La détection de secrets a des niveaux de confiance, pas des certitudes

Les heuristiques génériques (`password=…`, `api_key=…`) ne distinguent pas
`password=changeme123456` d'un vrai secret. Chaque finding porte `labels.confidence`
(`high` | `medium` | `low`) pour qu'un pipeline puisse trier, et le seuil de signalement est
configurable. Le relever est un assouplissement : « aucun secret en clair » ne se prouve plus
dès lors qu'on a choisi de ne pas regarder une partie de ce qui a été trouvé, donc la
correspondance CLD-CMP-9 tombe et le rapport le dit. La valeur détectée, elle, n'apparaît
jamais, quel que soit le niveau.

### Un réglage assoupli retire la correspondance, pas la mesure

Un contrôle dont la configuration sort de sa contrainte normative garde son statut : il a bel
et bien été évalué, contre une barre plus basse. Ce qu'il perd, ce sont ses `references` — il
ne prétend plus couvrir CIS, ISO ni SecNumCloud. Pépin ne sait pas vous dire si votre propre
barre convient à votre contexte ; il sait seulement refuser qu'une barre abaissée conserve un
badge qu'elle ne mérite plus. Voir
[Configurer les contrôles](guides/control-configuration.fr.md).

## 5 : les exigences auxquelles un scanner ne peut pas répondre

Certaines exigences SCSL, SecNumCloud et ISO portent sur l'organisation, les contrats, les
procédures ou le personnel : clauses de réversibilité, contrôle du personnel, procédures
d'incident, gestion des sous-traitants. Elles n'ont **aucune surface de configuration** : elles
ne sont donc pas au référentiel actif, et surtout pas silencieusement rapportées comme
conformes. `referentiel/gaps.md` suit ce qui est trié et pourquoi, et `pepin scsl` rapporte la
cohérence avec l'index SCSL gelé.

## Limites levées

Aucune à ce jour : cette page paraît avec la v0.2.0. Quand une limite est levée, elle sort des
sections ci-dessus et vient ici avec la version qui l'a levée, pour qu'un lecteur d'un rapport
plus ancien sache ce qui était vrai à l'époque.

| Limite | Levée en |
|---|---|
| _(aucune)_ | |

## Comment cette page reste vraie

Les tableaux ci-dessus sont calculés par `internal/docgen` depuis `referentiel/controles.yaml`,
`providers/*.yaml` et le verrou du `pass` de `internal/assess`, avec les mêmes fonctions que
celles qu'appelle le scan, pas une seconde implémentation. `mise run gen-docs` les régénère ;
`TestGeneratedDocsAreUpToDate` fait échouer la construction si ce qui est committé ne correspond
plus à ce que le code calcule. Une limite ne peut pas disparaître discrètement de cette page, et
une limite nouvelle ne peut pas discrètement en rester absente.

## Voir aussi

- [Matrice de couverture](coverage.fr.md) : le détail par contrôle.
- [Le modèle d'assessment](concepts/assessment-model.fr.md) : pourquoi `not-evaluated` existe.
- [Périmètre et non-objectifs](concepts/scope.fr.md) : ce que Pépin n'a jamais prétendu faire.
