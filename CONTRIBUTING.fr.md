> [🇬🇧 English](CONTRIBUTING.md) · 🇫🇷 Français

# Contribuer à Pépin

Merci de contribuer. Pépin est un outil d'audit : la barre de qualité est haute
parce qu'un faux résultat détruit sa raison d'être (l'opposabilité).

## Portes de qualité

```bash
mise run build   # compile
mise run test    # go test ./... -race + tests Rego (opa)
mise run audit   # vet + lint (golangci-lint) + gosec + govulncheck + osv
```

Ne proposez pas de changement si `mise run test` ou `mise run audit` échoue.
`opa test`, `opa check --strict` et `opa fmt` doivent rester propres.

## Règles non négociables

1. **Ancrage sur le contrat de l'API.** Ne jamais inventer le modèle de ressources
   d'un provider : le modèle évalué reflète le contrat natif du SDK/API. Un champ
   non vérifié ne s'emploie pas ; un champ **dérivé** est marqué « DÉRIVÉ » avec sa
   formule. Citez la source dans l'en-tête de chaque règle.
2. **Ne jamais inventer une référence normative.** Toute exigence SCSL (`CLD-*`) et
   toute correspondance de norme est vérifiée contre le texte officiel. Des tests
   (`TestSCSLReferencesExist`, `TestFrameworkReferencesExist`) refusent un id
   inexistant ; ajoutez d'abord l'exigence à la source si elle manque.
3. **Config effective.** Un contrôle interroge l'état résolu, jamais un fichier de
   service. Chaque règle porte une **garde de capacité** : si l'attribut nécessaire
   n'a pas été collecté, elle ne se déclenche pas (pas de faux positif).
4. **Aucun changement de règle sans scan réel.** Une nouvelle règle ou l'extension
   d'un fournisseur DOIT être validée par un scan réel (collecte live, ou audit
   d'un plan Terraform) avant d'annoncer une couverture. Un contrôle sans
   validation reste `fournisseurs: []` (écrit et testé, activation gelée).
5. **Aucune ressource de test laissée en vie.** Si un test provisionne du cloud,
   il DOIT être détruit à la fin (`terraform destroy`). Préférez le plan Terraform
   (`scan --terraform`), qui ne provisionne rien.
6. **Aucun secret en dur** (clés, mots de passe) : options CLI / environnement /
   `random_password` dans les fixtures.

## Ajouter un provider

Un provider = un fichier `providers/<nom>.yaml` : identité, auth, résolution des
identifiants, endpoint S3, `collecte` (API live), `mapping_terraform`, et le
`contrat` (état par type : `verifie` / `a_verifier` / `absent`, avec sa source).

## Ajouter un contrôle

1. Écrire la règle `internal/commonrules/rules/<code>.rego` (émet le `code` neutre,
   avec garde de capacité) et son test `_test.rego`.
2. Déclarer le contrôle dans `referentiel/controles.yaml` (sévérité, SCSL,
   correspondances de normes, `fournisseurs`).
3. Valider par un scan réel, puis activer (`fournisseurs`).

Tout ce que l'utilisateur lit s'écrit **deux fois**, côte à côte, parce que Pépin
détecte la langue du lecteur. Une règle émet `message`/`remediation` en français
et `labels.message_en`/`labels.remediation_en` en anglais ; un contrôle porte
`titre_en`, `description_en` et `remediation_en` ; un contrat de provider porte
`reason_en` à côté de `reason`. `mise run validate` refuse toute absence : un
rapport anglais ne doit jamais basculer au français en milieu de phrase. Le
français est la langue de référence du contenu normatif, l'anglais en est la
traduction.

## Documentation et commits

Les docs du dépôt sont **bilingues** : l'anglais est primaire (`README.md`), le
français en est la contrepartie (`README.fr.md`), reliés par un sélecteur de
langue. Les commits suivent les Conventional Commits
(`feat`/`fix`/`docs`/`refactor`/`test`/`chore`), sujet à l'impératif.

**La documentation fait partie du changement, pas d'un suivi.** L'essentiel de
`docs/` est généré depuis le binaire et le référentiel, et
`TestGeneratedDocsAreUpToDate` casse la CI dès que cela dérive. Donc, avant
d'ouvrir une pull request :

- lancer `mise run gen-docs` et committer ce qu'il régénère ;
- relire les pages que le changement rend **fausses**, pas seulement les pages
  générées : `docs/known-limitations.md` quand un angle mort se comble, la page
  du provider touché, `docs/coverage.md` ;
- tenir les deux langues synchronisées ;
- ajouter une ligne au CHANGELOG, dans les deux langues, dès que le changement
  déplace un **verdict** sur un tenant inchangé, une surface analysable ou un
  code de sortie.

Une page qui décrit un produit disparu est pire qu'une page absente : elle
inspire une confiance qu'elle ne peut pas tenir. La question qui tranche :
*quelqu'un qui lit la documentation sans lire le code serait-il induit en erreur
par ce changement ?*
