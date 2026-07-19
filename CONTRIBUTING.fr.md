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

## Commits

Conventional Commits (`feat`/`fix`/`docs`/`refactor`/`test`/`chore`), sujet à
l'impératif.
