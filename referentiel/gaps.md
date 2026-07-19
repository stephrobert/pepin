# Trous de couverture SCSL-CLD — analyse honnête

L'index SCSL gelé compte **82 exigences** de posture cloud (`CLD-*`). Pépin en couvre
**30** par un contrôle actif ancré. Ce document classe les **48 trous** restants (hors les
4 agrégats `CLD-GEN-*` du socle essentiel, couverts par les contrôles fins) — **sans les
maquiller en contrôles** : un scanner de configuration ne peut, et ne doit, pas prétendre
vérifier ce qui est organisationnel ou contractuel. Cette transparence EST une preuve
d'opposabilité : un auditeur sait exactement ce que l'outil couvre et ce qu'il ne couvre pas.

Régénérer l'écart : `pepin scsl --index <framework-scsl>/api/v1/exigences.json`.

## A. Comblé par mapping (fait)

- **`CLD-CMP-9`** — *aucun secret en clair dans les user-data*. La règle
  `compute_instance_no_secrets_in_user_data` le vérifie exactement ; elle était mappée à
  tort sur `CLD-CMP-2` (IMDSv2, qu'elle ne teste pas). Remappée sur CMP-9.

## B. Backlog technique — comblable, mais exige une NOUVELLE collecte

Vérifiables par l'outil **si** l'API du provider expose l'attribut et que le collecteur le
normalise (chaque ajout = 1 contrat vérifié + 1 collecte + 1 règle + tests + **scan réel**).

| Exigence | Niv. | Attendu | Donnée à collecter |
|---|---|---|---|
| `CLD-CMP-2` | R1 | IMDSv2 imposé (hop limit 1, IMDSv1 off) | config IMDS de l'instance (si exposée) |
| `CLD-CMP-8` | R1 | rôle d'instance au moindre privilège | rôle/scope d'identité machine attaché |
| `CLD-IAM-5` | R2 | blocage après N échecs d'authentification | politique de verrouillage de compte |
| `CLD-IAM-11` | R1 | politique de mot de passe robuste | politique de compte du provider |
| `CLD-IAM-13` | R2 | désactivation des identifiants inactifs | `last_used`/`created` des access keys |
| `CLD-IAM-16` | R2 | right-sizing des permissions inutilisées | analyse d'usage (access analyzer) — souvent non exposée |
| `CLD-LOG-6` | R2 | journaux de flux réseau activés | config flow-logs (si exposée) |
| `CLD-CHF-5` | R3 | certificats émis par une CA d'un État UE | émetteur des certs des listeners TLS |
| `CLD-NET-9` | R2 | interfaces exposées inventoriées + authentifiées | corrélation SG ouverts + endpoints (partiel) |
| `CLD-GVN-8` | R2 | régions inutilisées désactivées/restreintes | équivalent souverain d'un SCP region-deny (à établir) |

## C. Internes Kubernetes — hors API du provider

Le durcissement du cluster (RBAC, Pod Security Standards, NetworkPolicy, API server, etcd,
admission control, identités de pods) vit **dans le cluster**, pas dans l'API cloud managée
(SKS/Kapsule) qui n'expose que le plan de contrôle. Nécessite un **mode de collecte via
kubeconfig** (introspection du cluster), distinct des collecteurs providers actuels.

`CLD-K8S-4` (RBAC), `CLD-K8S-5` (PSS), `CLD-K8S-6` (NetworkPolicy), `CLD-K8S-7` (API server),
`CLD-K8S-8` (chiffrement etcd), `CLD-K8S-9` (images signées), `CLD-K8S-10` (secrets externes),
`CLD-K8S-11` (admission policy), `CLD-K8S-12` (identités de pods).

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
- **Journalisation (processus/rétention)** : `CLD-LOG-2` (rétention 6 mois), `CLD-LOG-3` (SIEM
  central), `CLD-LOG-4` (synchro d'horloge), `CLD-LOG-5` (corrélation/détection), `CLD-LOG-7`
  (intégrité des journaux).
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
| C — internes Kubernetes (kubeconfig) | 9 | roadmap outil (mode k8s) |
| D — hors périmètre outil (organisationnel/contractuel) | 29 | attestation, pas une règle |

La priorité honnête n'est **pas** d'atteindre « 82/82 » : ~29 exigences ne sont pas
vérifiables par lecture de configuration. La cible réaliste est de couvrir les buckets **B**
et **C** (identité machine, politiques de compte, clés inactives, flow logs, durcissement
k8s) au fil des contrats providers vérifiés, et de **déclarer explicitement** le bucket D
comme relevant de la qualification du fournisseur et des processus du client.
