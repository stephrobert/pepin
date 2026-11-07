> [🇬🇧 English](SECURITY.md) · 🇫🇷 Français

# Politique de sécurité

Pépin est un outil de sécurité : sa fiabilité est une exigence, pas une option.

## Signaler une vulnérabilité

Merci de **ne pas** ouvrir d'issue publique pour une vulnérabilité.

Signalez-la en privé ici :
**https://github.com/stephrobert/pepin/security/advisories/new**
(GitHub → **Security** → **Report a vulnerability**). L'avis reste privé entre
vous et le mainteneur jusqu'à la publication d'un correctif.

Si ce formulaire ne vous est pas accessible, contactez le mainteneur par un canal
privé plutôt que par une issue publique.

Merci d'inclure :

- une description de la vulnérabilité et de son impact ;
- les étapes de reproduction (version/commit, commande, sortie) ;
- le cas échéant, une proposition de correctif.

Nous accusons réception sous quelques jours ouvrés et vous tenons informé de la
correction et de sa publication.

## Périmètre

Sont particulièrement concernés :

- l'**intégrité du résultat** : un contrôle qui sortirait `pass` alors qu'il n'a
  pas été réellement évalué, ou une preuve/bundle falsifiable sans détection ;
- la **gestion des identifiants** : tout chemin où un secret (clé, jeton) pourrait
  fuiter dans un argument de ligne de commande, un log ou un artefact ;
- l'**exécution** : injection via un descripteur de provider, un plan Terraform ou
  un inventaire fournis en entrée.

## Conception sécurisée

Les frontières de confiance sont les identifiants et réponses d'API des
fournisseurs, les plans Terraform, les inventaires, les politiques Rego tierces
et les bundles de preuve. Une ressource auditée n'est pas fiable simplement
parce que son fournisseur l'a renvoyée.

| Principe | Application et preuve |
|---|---|
| Économie de mécanisme | Les règles communes utilisent des observations typées ; la collecte propre à chaque fournisseur reste séparée de l'évaluation. Voir [ADR-0001](docs/adr/0001-regles-communes-providers-collecteurs.md). |
| Refus par défaut | Des observations absentes ne prouvent pas la conformité. Une dérogation produit `exempted`, pas `pass`. Voir [évaluation](internal/assess/) et [dérogations](internal/exempt/). |
| Médiation complète | Les observations nécessaires sont vérifiées avant d'affirmer le `pass` d'un contrôle ; la vérification d'un bundle recoupe empreintes et cohérence au lieu de croire un verdict stocké. Voir [ADR-0018](docs/adr/0018-ce-quun-bundle-prouve-de-lui-meme.md). |
| Conception ouverte | Règles, contrats des fournisseurs, évaluation et vérification sont publics ; la conception ne repose pas sur des algorithmes cachés. |
| Séparation des privilèges | La cohérence interne d'un bundle et l'identité du signataire sont deux vérifications distinctes. Une empreinte dans le bundle n'est pas une ancre de confiance externe. |
| Moindre privilège | Utiliser les permissions minimales documentées du fournisseur et un kubeconfig restreint. Les droits viennent de l'identité fournie ; Pépin ne les réduit pas automatiquement. Certains chemins ont des limites de permissions documentées. |
| Moindre mécanisme commun | La résolution des identifiants du fournisseur est séparée de leur projection dans l'inventaire normalisé. Des tests authentifient des requêtes depuis des fichiers d'identifiants sans exporter ceux-ci dans l'inventaire. Voir [tests de projection des identifiants](internal/genprovider/genprovider_test.go). |
| Acceptabilité | Les résultats typés distinguent échec, mesure non prise en charge et dérogation ; les erreurs expliquent le refus d'évaluation ou de vérification. Voir la [référence CLI](docs/reference/cli.fr.md). |

### Erreurs courantes et protections

| Risque | Protection et limite |
|---|---|
| Exposition d'identifiants | Configuration native ou variables d'environnement, projection des seuls champs déclarés et détection de secrets en CI. Les rapports peuvent encore contenir des données sensibles du tenant : les protéger séparément. |
| Exfiltration par une politique | Les politiques Rego utilisateur sont du code non fiable. Un témoin HTTP local vérifie leur refus réseau, y compris les références de schémas JSON distants. Voir [ADR-0021](docs/adr/0021-une-regle-chargee-a-lexecution-est-du-code-non-fiable.md) et son [test de régression](cmd/policy_network_test.go). |
| Entrée mal formée ou preuve fabriquée | Parseurs et chargeurs de politique valident les entrées ; une observation absente ne peut pas être inventée. Fuzzing et tests d'altération éprouvent les refus. Voir [campagnes](.github/workflows/campaigns.yml) et [tests de vérification](internal/assess/bundle_test.go). |
| Surface d'entrée excessive | Les descripteurs n'acceptent que des types d'authentification connus et valident leur forme ; une entrée arbitraire ne devient pas une configuration valide. Voir [validation des descripteurs](internal/genprovider/validate.go). |
| Chaîne d'approvisionnement non fiable | Actions et outils épinglés, analyse des dépendances, releases signées et provenance. Voir [ADR-0016](docs/adr/0016-chaine-dapprovisionnement-verifiable.md). |
| Permissions cloud excessives | Utiliser les permissions minimales documentées ; ne jamais supposer qu'un scanner rend un identifiant privilégié « lecture seule ». La CI ne détient aucun identifiant cloud. Voir [ADR-0012](docs/adr/0012-aucun-identifiant-en-ci.md). |

Cette page décrit les protections et leurs limites, pas les connaissances
d'une personne. Les réponses OpenSSF `know_secure_design` et
`know_common_errors` exigent la confirmation d'un développeur principal.

### Cryptographie

Pépin utilise la cryptographie : le fait de ne pas être une bibliothèque
cryptographique ne justifie pas de mettre tous ces critères en N/A.

- Les empreintes SHA-256 appellent `crypto/sha256` de Go. L'authentification EXO2
  appelle `crypto/hmac` avec SHA-256 ; SigV4 utilise le signataire du SDK AWS.
  Pépin implémente le format de requête, pas un nouveau hachage ou MAC. Voir
  [authentification](internal/collect/auth.go).
- La [politique TLS commune](internal/tlspolicy/tlspolicy.go) s'applique à
  Kubernetes, aux collecteurs REST (dont EIM et OKS) et à S3. Elle impose TLS 1.2
  ou ultérieur, les suites TLS 1.2 AEAD/ECDHE, des clés RSA d'au moins 2048 bits,
  EC d'au moins 224 bits, ou Ed25519. Go fournit TLS 1.3, l'accord de clés et
  l'aléatoire. Go vérifie toujours le nom du serveur, l'expiration et la confiance.
  Tous les certificats d'au moins une chaîne serveur vérifiée respectent la
  politique, CA comprise. Tous les certificats d'une chaîne client fournie par
  l'opérateur sont vérifiés au chargement. Les refus sont explicites ; les échecs
  de collecte restent enregistrés, jamais transformés en inventaires vides réussis.
  Un cluster privé utilisant RSA 1024 doit renouveler ses clés.
- Les signatures des bundles sont vérifiées séparément de leurs empreintes.
  Une intégrité vérifiée sans signature ne prouve pas l'auteur ; exiger la
  signature lorsque l'appelant a besoin d'une preuve authentifiée.
- La [politique de clés commune](internal/cryptopolicy/keys.go) refuse aussi les
  secrets HMAC EXO2, SigV4 et S3 de moins de 14 octets UTF-8 effectifs (112 bits
  de matériau). La préparation du fournisseur, les signatures directes et le
  chemin SDK appliquent cette garde, sans dérogation. C'est une vérification
  de taille, **pas une estimation d'entropie** : utiliser les secrets émis par
  le fournisseur ou du matériau généré cryptographiquement.
- `verify --pubkey` exige une clé publique PEM locale (PKIX ou RSA PKCS#1).
  Les mêmes seuils RSA/EC/Ed25519 sont vérifiés avant Cosign. Celui-ci reçoit
  un instantané temporaire à permissions privées de la clé validée, pas le
  chemin original remplaçable après la garde. Exporter d'abord les clés publiques
  KMS en PEM ; les références distantes ne sont pas acceptées sans contrôle.
- Pépin n'authentifie pas d'utilisateurs externes par une base locale de mots de
  passe. Les identifiants des fournisseurs servent à l'authentification sortante.

#### Périmètre cryptographique et tailles imposées

| Mécanisme | Paramètres et responsabilité | Vérification / limite |
|---|---|---|
| Empreintes d'inventaire, politique et bundle | SHA-256, choisi par Pépin | Empreinte de 256 bits, bibliothèque standard |
| MAC de requêtes EXO2, SigV4 et S3 | HMAC-SHA256 ; secret émis par le fournisseur ou l'opérateur S3 local | Au moins 14 octets d'entrée, vérifiés avant signature ; empreinte SHA-256 |
| TLS Kubernetes, REST, EIM, OKS et S3 | TLS Go avec la politique commune ci-dessus | Clés client et chaînes serveur vérifiées ; RSA 1024 refusé |
| Signatures de bundle | Clé publique PEM de l'opérateur et Cosign | Même garde RSA/EC/Ed25519, instantané validé, puis vérification réelle de signature |
| Signatures de release | Cosign v3.1.3 épinglé, défaut keyless | Clé éphémère ECDSA P-256/SHA-256 ; la CI refuse la substitution de clé/algorithme |
| Jetons porteurs et identifiants en en-tête | Identifiants d'authentification sortante, pas des clés choisies pour une primitive cryptographique | Non générés par Pépin ; leur longueur n'est pas une estimation d'entropie |
| Endpoints HTTP, dont MinIO local | Transport existant sans TLS | Toujours accepté par compatibilité ; aucune confidentialité ni garde TLS |

Les [tests](internal/tlspolicy/collectors_test.go) exercent de vraies connexions
TLS locales pour REST et S3, avec les [tests Kubernetes](internal/kubeauth/kubeauth_test.go) :
RSA 1024 est refusé, RSA 2048 et P-256 restent acceptés, et un refus REST est
enregistré comme incomplet avant l'arrivée de toute requête HTTP authentifiée.
Les tests HMAC refusent 13 octets et acceptent 14. Les tests de signature
refusent RSA 1024 avant Cosign et exercent le vrai binaire avec des signatures
RSA 2048/P-256 et des données altérées. Le témoin natif est explicite :

```bash
PEPIN_COSIGN_INTEGRATION=1 mise exec -- go test ./cmd -run '^TestNativeCosignKeyStrengthAndSignature$' -v -count=1
```

Il génère ses clés de test, sans identifiant cloud ; Cosign peut contacter son
service public de transparence. Les suites par défaut testent les gardes sans
réseau. Le défaut de release épinglé est vérifié dans
[la génération de clés Cosign](https://github.com/sigstore/cosign/blob/v3.1.3/pkg/cosign/keys.go).
Ces gardes établissent la taille des clés, pas la qualité de l'aléatoire du
fournisseur. La proposition OpenSSF `crypto_keylength` est désormais **Met** ;
publication et enregistrement du questionnaire sont des étapes distinctes.

## Bonnes pratiques d'usage

- Les identifiants ne passent que par l'environnement ou la configuration native
  du provider, jamais en argument de ligne de commande.
- Vérifiez l'intégrité d'un dossier de preuve avec `pepin verify`, et sa
  **signature** avec `pepin verify --pubkey <clé>` (le scellement cosign est à
  l'identité de l'opérateur).
- Épinglez les versions ; n'exécutez que des binaires dont vous vérifiez la
  provenance.
