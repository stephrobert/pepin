> [🇬🇧 English](SECURITY.md) · 🇫🇷 Français

# Politique de sécurité

Pépin est un outil de sécurité : sa fiabilité est une exigence, pas une option.

## Signaler une vulnérabilité

Merci de **ne pas** ouvrir d'issue publique pour une vulnérabilité. Utilisez le
canal privé de signalement des vulnérabilités de GitHub (**Security → Report a
vulnerability**), ou contactez le mainteneur par un canal privé.

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

## Bonnes pratiques d'usage

- Les identifiants ne passent que par l'environnement ou la configuration native
  du provider, jamais en argument de ligne de commande.
- Vérifiez l'intégrité d'un dossier de preuve avec `pepin verify`, et sa
  **signature** avec `pepin verify --pubkey <clé>` (le scellement cosign est à
  l'identité de l'opérateur).
- Épinglez les versions ; n'exécutez que des binaires dont vous vérifiez la
  provenance.
