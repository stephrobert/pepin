# Trous de couverture SCSL-CLD — analyse honnête

L'index SCSL compte **83 exigences** de posture cloud (`CLD-*`). Pépin en couvre
**35** par un contrôle actif ancré (dont 4 via le mode `--kubeconfig`). Ce document classe les
trous restants (hors les
4 agrégats `CLD-GEN-*` du socle essentiel, couverts par les contrôles fins) — **sans les
maquiller en contrôles** : un scanner de configuration ne peut, et ne doit, pas prétendre
vérifier ce qui est organisationnel ou contractuel. Cette transparence EST une preuve
d'opposabilité : un auditeur sait exactement ce que l'outil couvre et ce qu'il ne couvre pas.

Régénérer l'écart : `pepin scsl --index <framework-scsl>/api/v1/exigences.json`.

## A. Comblé (fait)

- **Politiques EIM *inline*** — elles échappaient à TOUS les contrôles `iam_policy_*`
  (faux négatif : une policy `Action:*` attachée à un utilisateur laissait le contrôle au
  vert). Collectées depuis la chaîne à 3 niveaux `ReadUsers` → `ReadUserPolicies` →
  `ReadUserPolicy` (hors portée du `for_each` mono-niveau, d'où un collecteur Go).
  **Vérifié de bout en bout** : sans le correctif la policy est invisible, avec lui
  `iam_policy_no_administrative_privileges` la détecte.

- **`CLD-CMP-9`** — *aucun secret en clair dans les user-data*. La règle
  `compute_instance_no_secrets_in_user_data` le vérifie exactement ; elle était mappée à
  tort sur `CLD-CMP-2` (IMDSv2, qu'elle ne teste pas). Remappée sur CMP-9.

## B. Backlog technique — comblable, mais exige une NOUVELLE collecte

Vérifiables par l'outil **si** l'API du provider expose l'attribut et que le collecteur le
normalise (chaque ajout = 1 contrat vérifié + 1 collecte + 1 règle + tests + **scan réel**).

| Exigence | Niv. | Attendu | Donnée à collecter |
|---|---|---|---|
| `CLD-CMP-2` | R1 | IMDSv2 imposé (hop limit 1, IMDSv1 off) | **mécanisme absent chez Outscale** (cf. E) ; à vérifier chez les autres |
| `CLD-CMP-8` | R1 | rôle d'instance au moindre privilège | **pas d'identité machine chez Outscale** (cf. E) |
| `CLD-IAM-5` | R2 | blocage après N échecs d'authentification | politique de verrouillage — **absente chez Outscale** (cf. E) |
| `CLD-IAM-11` | R1 | politique de mot de passe robuste | politique de compte — **absente chez Outscale** (cf. E) |
| `CLD-IAM-13` | R2 | désactivation des identifiants inactifs | `last_used` — **non exposé chez Outscale** (cf. E) |
| `CLD-IAM-16` | R2 | right-sizing des permissions inutilisées | **aucune API d'usage chez Outscale** (cf. E) |
| `CLD-LOG-6` | R2 | journaux de flux réseau activés | config flow-logs — **absente chez Outscale** (cf. E) ; à vérifier chez les autres |
| `CLD-CHF-5` | R3 | certificats émis par une CA d'un État UE | **émetteur non exposé chez Outscale** (cf. E) |
| `CLD-NET-9` | R2 | interfaces exposées inventoriées + authentifiées | corrélation SG ouverts + endpoints (partiel) |
| `CLD-GVN-8` | R2 | régions inutilisées désactivées/restreintes | **mécanisme absent chez Outscale** (cf. E) ; à vérifier chez les autres |

### Absences VÉRIFIÉES côté fournisseur (E)

Un écart n'est opposable que s'il distingue « pas encore collecté » de « le mécanisme
n'existe pas ». Les faits ci-dessous sont vérifiés sur le **contrat d'API officiel** et par
**scan réel**, pas supposés — ils sortent donc du backlog : aucun collecteur ne les comblera.

| Exigence | Fournisseur | Fait vérifié | Source |
|---|---|---|---|
| `CLD-LOG-6` | outscale | **aucune action `ReadFlowLogs`** dans l'OAPI : les journaux de flux réseau n'existent pas au catalogue. L'absence n'est PAS un trou de l'outil. | contrat `outscale/osc-api` (`outscale.yaml`) |
| `CLD-LOG-2` | outscale | le journal d'API (`ReadApiLogs`) est en **lecture seule** et borné à ~32 jours : aucun réglage de rétention n'est exposé au tenant, la cible de 6 mois ne peut pas être satisfaite par configuration. | contrat OAPI + doc `docs.outscale.com` |
| `CLD-K8S-*` (audit) | outscale | les journaux d'audit OKS sont **produits automatiquement** vers OOS : un contrôle « audit activé » serait toujours vert et sans valeur ; l'angle utile est la PROTECTION des journaux (OOSAccess). | doc OKS « Managing Cluster Logs » |
| `CLD-IAM-11` | outscale | **aucun réglage de politique de mot de passe** : la seule politique de compte exposée est `ApiAccessPolicy`, limitée à `MaxAccessKeyExpirationSeconds` et `RequireTrustedEnv`. | contrat OAPI |
| `CLD-IAM-5` | outscale | **protection assurée NATIVEMENT, non configurable** : le contrat documente un verrouillage automatique (4 échecs consécutifs → 1 min, +1 min par 4 échecs, jusqu'à 10 min ; clés d'accès ET mots de passe). Il n'existe aucun réglage exposé au tenant, donc rien à auditer — mais l'exigence est **satisfaite par le fournisseur**, pas « absente ». | contrat OAPI |
| `CLD-CMP-2` | outscale | **aucune action IMDS/metadata** au catalogue (235 actions) et aucun champ `Vm` de type token/hop-limit : l'imposition d'IMDSv2 n'est ni configurable ni observable par l'API. | contrat OAPI |
| `CLD-CMP-8` | outscale | **aucun schéma `Role`/`Profile`** et aucun champ de rôle sur `Vm` : il n'existe pas d'identité machine attachée à une instance — l'exigence « rôle au moindre privilège » est sans objet ici. | contrat OAPI |
| `CLD-CHF-5` | outscale | `ServerCertificate` n'expose **que** `{ExpirationDate, Id, Name, Orn, Path, UploadDate}` : **aucun émetteur/CA**, donc l'origine du certificat n'est pas observable. | contrat OAPI |
| `CLD-IAM-16` | outscale | **aucune API d'analyse d'usage** (pas d'équivalent access-analyzer / last-accessed) : le right-sizing des permissions ne peut pas être mesuré, seulement supposé. | contrat OAPI |
| `CLD-GVN-8` | outscale | **aucun mécanisme de restriction de région** : seules `ReadRegions`/`ReadSubregions` existent (lecture seule, schéma `{Endpoint, RegionName}`), et l'OAPI **ne connaît aucune `Condition`** de policy EIM — il n'existe donc pas d'équivalent au SCP region-deny. | contrat OAPI |
| `CLD-IAM-13` | outscale | le schéma `AccessKey` n'expose **pas de `LastUsed`** (seulement `CreationDate`, `ExpirationDate`, `LastModificationDate`, `State`) : l'inactivité d'une clé n'est pas observable — un contrôle « clés inutilisées » serait une inférence, pas une mesure. | contrat OAPI |

## C. Internes Kubernetes — état DANS le cluster

Le durcissement du cluster vit **dans le cluster**, pas dans l'API cloud managée
(OKS/SKS/Kapsule) qui n'expose que le plan de contrôle. Le **mode `--kubeconfig`**
(provider `kubernetes`, portée `in-cluster`) comble désormais une partie de ce bucket.

**Couvert** (mode `--kubeconfig`, validé sur un cluster réel) : `CLD-K8S-4` (RBAC —
cluster-admin au-delà de `system:masters`), `CLD-K8S-5` (Pod Security Standards),
`CLD-K8S-6` (NetworkPolicy), `CLD-K8S-10` (gestionnaire de secrets externes).

**Hors de portée sur un cluster MANAGÉ — responsabilité du fournisseur** : `CLD-K8S-7`
(durcissement de l'API server : `anonymous-auth`, mode d'autorisation, journal d'audit) et
`CLD-K8S-8` (chiffrement des secrets etcd via KMS). Ces réglages vivent dans le plan de
contrôle opéré par le fournisseur : le client n'y a **aucun accès**, même avec un kubeconfig
d'administration. Ils relèvent de la qualification du prestataire, pas d'un scan de tenant.

**Mesurables seulement par INFÉRENCE — non livrés** : la doctrine du projet est de ne jamais
présenter une inférence comme une mesure (c'est ce qui a fait tomber F2 et produit le bucket E).

| Exigence | Ce qui bloque la mesure |
|---|---|
| `CLD-K8S-11` (moteur de politiques d'admission bloquant) | l'API ne distingue pas un **moteur de politiques** (Kyverno, Gatekeeper) d'un **webhook propre à un composant** : sur un cluster réel on n'observe que `cert-manager-webhook` et `ingress-nginx-admission`, plus des `ValidatingAdmissionPolicy` appartenant au fournisseur. Conclure « un webhook existe donc c'est conforme » serait un faux vert ; les trier exige une liste blanche de moteurs connus, donc une heuristique. |
| `CLD-K8S-9` (images signées, registre autorisé) | il faudrait savoir ce que le contrôleur d'admission **impose**, pas seulement qu'il existe : cela suppose d'interpréter le contenu de politiques, non de lire un état. |
| `CLD-K8S-12` (identités de pods scopées, IMDS injoignable) | l'injoignabilité de l'IMDS depuis un pod est une propriété **réseau à l'exécution**, pas un état déclaratif lisible dans l'API. |

Un contrôle explicitement étiqueté « heuristique » resterait possible pour K8S-11 si l'on
accepte d'assumer la liste blanche — décision produit, pas décision technique.

## D. Hors périmètre d'un scanner de configuration

Exigences **organisationnelles, procédurales ou contractuelles** : elles se prouvent par des
processus, des politiques ou la **qualification SecNumCloud du fournisseur**, jamais par la
lecture d'une configuration de tenant. Les fabriquer en « contrôles » serait malhonnête (le
travers déjà retiré du catalogue (checks propres aux hyperscalers non souverains)). À traiter par attestation/documentation, pas
par une règle.

- **Compute/durcissement** : `CLD-CMP-3` (séparation prod/hors-prod), `CLD-CMP-4` (anti-maliciel),
  `CLD-CMP-5` (gestion des vulnérabilités), `CLD-CMP-6` (postes d'admin dédiés), `CLD-CMP-7`
  (durcissement hyperviseur/hôte — côté fournisseur).
- **IAM (processus)** : `CLD-IAM-7` (flux d'admin chiffrés e2e), `CLD-IAM-8` (revue annuelle des
  droits), `CLD-IAM-9` (séparation des tâches), `CLD-IAM-10` (identités nominatives, pas de
  compte partagé), `CLD-IAM-14` (cartographie des chemins d'attaque), `CLD-IAM-15` (accès JIT).
- **Journalisation (processus)** : `CLD-LOG-3` (SIEM central), `CLD-LOG-4` (synchro
  d'horloge), `CLD-LOG-5` (corrélation/détection), `CLD-LOG-7` (intégrité des journaux).
  (`CLD-LOG-2`, rétention, est classée en **E** : chez Outscale le journal d'API est en
  lecture seule et borné, donc l'écart est un fait fournisseur vérifié, pas un processus.)
- **Gouvernance/contrat** : `CLD-GVN-2` (hygiène ANSSI), `CLD-GVN-5` (programme d'audit),
  `CLD-GVN-6` (transparence localisation/sous-traitants), `CLD-GVN-7` (notification d'accès),
  `CLD-GVN-9` (couverture de découverte — méta), `CLD-GVN-10` (demandes d'autorités),
  `CLD-GVN-11` (réversibilité/portabilité), `CLD-GVN-12` (restitution/suppression en fin de contrat).
- **Chiffrement/gouvernance des clés** : `CLD-CHF-3` (clés conformes ANSSI, gestion des accès).
- **Réseau (détection)** : `CLD-NET-8` (IDS/IPS, défense en profondeur).
- **Stockage (processus)** : `CLD-STO-5` (sauvegarde hors-ligne de la config), `CLD-STO-6`
  (effacement sécurisé/destruction de clés en fin de vie), `CLD-STO-7` (découverte et
  classification des données sensibles — outil dédié).

## Synthèse

| Bucket | Nombre | Nature |
|---|---|---|
| A — comblé (mapping) | 1 | fait |
| B — backlog technique (nouvelle collecte + scan réel) | 10 | roadmap outil |
| C — internes Kubernetes | 4 couvertes · 2 hors portée (managé) · 3 non mesurables sans inférence | mode `--kubeconfig` livré |
| D — hors périmètre outil (organisationnel/contractuel) | 29 | attestation, pas une règle |
| E — absence (ou satisfaction native) VÉRIFIÉE côté fournisseur | 11 (outscale) | ni trou de l'outil, ni réglage du client |

La priorité honnête n'est **pas** d'atteindre « 83/83 » : ~29 exigences ne sont pas
vérifiables par lecture de configuration. La cible réaliste est de couvrir les buckets **B**
et **C** (identité machine, politiques de compte, clés inactives, flow logs, durcissement
k8s) au fil des contrats providers vérifiés, et de **déclarer explicitement** le bucket D
comme relevant de la qualification du fournisseur et des processus du client.

Le bucket **E** est ce qui rend l'écart opposable : quand le mécanisme n'existe PAS chez le
fournisseur, le dire — sourcé — vaut mieux qu'un contrôle qui resterait éternellement « non
évalué ». C'est aussi une donnée de comparaison entre clouds souverains, que personne d'autre
ne publie.

**Une exigence manquante n'est pas une fatalité** : quand un contrôle est légitime et
observable mais qu'aucune exigence SCSL ne le porte, on **ajoute l'exigence à SOCLE**
(correspondance ancrée + rattachement à un vecteur), puis on écrit le contrôle. Fait pour
`SOCLE-CLD-CMP-10` (protection contre la suppression, ancrée CSA CCM CCC-04, vecteur V-CLD-10).
