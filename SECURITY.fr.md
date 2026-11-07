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
- Le [client Kubernetes](internal/kubeauth/kubeauth.go) vérifie la CA configurée
  et exige TLS 1.2 ou ultérieur. L'accord de clés et l'aléatoire TLS viennent de
  Go ; Pépin ne fournit pas de générateur aléatoire personnalisé.
- Les signatures des bundles sont vérifiées séparément de leurs empreintes.
  Une intégrité vérifiée sans signature ne prouve pas l'auteur ; exiger la
  signature lorsque l'appelant a besoin d'une preuve authentifiée.
- Pépin n'authentifie pas d'utilisateurs externes par une base locale de mots de
  passe. Les identifiants des fournisseurs servent à l'authentification sortante.
- **Limite de robustesse des clés :** Pépin n'impose actuellement aucune taille
  minimale aux clés TLS client fournies par l'opérateur. Fournir des clés RSA
  d'au moins 2048 bits ou des clés de courbe elliptique d'au moins 224 bits, et
  suivre la politique d'identifiants du fournisseur. Cette recommandation
  n'est pas une vérification implémentée et ne suffit pas à établir le critère
  OpenSSF `crypto_keylength`.

## Bonnes pratiques d'usage

- Les identifiants ne passent que par l'environnement ou la configuration native
  du provider, jamais en argument de ligne de commande.
- Vérifiez l'intégrité d'un dossier de preuve avec `pepin verify`, et sa
  **signature** avec `pepin verify --pubkey <clé>` (le scellement cosign est à
  l'identité de l'opérateur).
- Épinglez les versions ; n'exécutez que des binaires dont vous vérifiez la
  provenance.
