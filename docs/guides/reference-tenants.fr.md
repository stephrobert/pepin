> [🇬🇧 English](reference-tenants.md) · 🇫🇷 Français

# Les tenants de référence

Une fixture est écrite par l'auteur de la règle. Elle prouve donc que la règle **se
déclenche** — jamais qu'elle a **raison** sur une configuration que personne n'a conçue
pour elle. C'est un test auto-confirmant : il mesure l'intention de son auteur, pas le
parc.

Le précédent qui tranche : le rejeu de stacks Terraform de tiers contre le binaire a
trouvé, en une séance, un faux positif `CRITICAL` sur la configuration Scaleway la plus
courante. Aucune fixture maison ne l'aurait révélé, parce qu'aucune fixture maison ne
décrit une configuration **correcte** que son auteur n'a pas dessinée.

Un tenant de référence est la réponse : une configuration réelle, publiée, licenciée,
épinglée à un commit, rejouée à chaque build, et comparée aux verdicts consignés à côté
d'elle.

## Ce qui vit dans le dépôt

```
references/tenants/<fournisseur>/<nom>/
  tenant.yaml     provenance (dépôt, commit, chemin, licence), posture, pourquoi ce n'est pas auto-confirmant
  plan.json       le plan Terraform, réduit à ce que Pépin lit
  expected.txt    le verdict par contrôle, généré, relu
```

Rien n'est provisionné. `terraform plan` ne crée aucune ressource cloud, donc il n'y a
rien à détruire — c'est ce que `CONTRIBUTING.fr.md` préfère à un scan live.

## Ce qu'un tenant de référence prouve, et ce qu'il ne prouve pas

| Il établit | Il n'établit **pas** |
|---|---|
| qu'une configuration tierce réelle produit le verdict consigné ici | quoi que ce soit de l'inventaire live d'un tenant réel |
| qu'un contrôle ne se déclenche PAS sur une configuration que personne n'a écrite pour lui (faux positif) | que l'API du fournisseur réponde ce que le plan annonce |
| qu'un contrôle se déclenche ENCORE sur une configuration réellement vulnérable (faux négatif) | les droits qu'un scan demande, ni la classe d'un refus |
| qu'un `not-evaluated` est atteint avec le type de ressource réellement présent | les bornes de pagination ni la limitation de débit du fournisseur |

Un plan porte l'état **planifié**. Ce qu'un fournisseur **répond** reste dû à une collecte
live, et les [Limites connues](../known-limitations.fr.md) le disent à sa place.

## Le plan est réduit, et c'est une garde

Pépin ne lit d'un plan que deux choses (`internal/tfparse.ParsePlan`) : `planned_values`
(ou `values`), et les `source` des appels de modules sous `configuration.root_module`.
Tout le reste — `variables`, `provider_config`, `prior_state`, `resource_changes` — est
ignoré par le produit, et c'est précisément là qu'un plan pris sur un tenant **réel**
porterait ses identifiants, ses UUID et ses adresses.

Un tenant de référence ne porte donc que ce que Pépin lit, et toute valeur que Terraform
lui-même marque `sensitive` y est mise à null.
`TestNoReferenceTenantPlanCarriesMoreThanPepinReads` refuse tout le reste. La réduction est
une discipline de sécurité avant d'être une discipline de taille — elle divise pourtant le
corpus par sept.

## Les deux postures

`tenant.yaml` déclare une posture, et une porte la confronte à ce que le scan mesure :

- **`exposed`** — le tenant relève au moins un écart. Un tenant qui cesserait d'en relever
  est un faux négatif candidat.
- **`hardened`** — le tenant ne relève aucun écart `critical`/`high`. C'est le
  contre-témoin, et le seul endroit où un faux positif se voit.

Une posture ne se décrète jamais. `TestEveryPostureIsTheOneMeasured` échoue dès que le
manifeste et le binaire ne disent pas la même chose.

## Régénérer

```bash
mise run tenants-refresh    # re-dérive les plans depuis les amonts épinglés
mise run tenants-update     # re-dérive les verdicts attendus
mise run veracity-update    # re-dérive le registre de dette de véracité
```

`scripts/reference-tenant.sh --all` clone chaque amont au commit consigné, en fait un plan
hors ligne avec des identifiants factices et publics, puis réduit le résultat. **Les six
plans reviennent à l'octet près.** C'est ce qui permet à un relecteur de vérifier qu'un
tenant vient bien du dépôt tiers qu'il nomme, et non d'un fichier que Pépin a fini par
s'écrire à lui-même.

> **Un verdict qui bascule sur un amont épinglé est une décision, pas un rafraîchissement.**
> Soit le produit s'est amélioré, soit il a régressé sur une configuration que personne n'a
> écrite pour lui. Le dire avant de régénérer.

## Ce qu'un tenant paie au contrat de véracité

Le [contrat de véracité](../../CONTRIBUTING.fr.md) compte, par contrôle × fournisseur ×
source, les verdicts prouvés de bout en bout à travers le binaire. Un tenant de référence
en prouve une partie, et ils sont comptés **une fois**, dans le même registre que les
scénarios écrits à la main — deux chiffres de couverture divergeraient, et celui qui
diverge est toujours celui qu'on lit.

Tout verdict observé ne compte pas. Le **filtre substantiel** (`tenants.Substantive`) :

- `fail`, `pass`, `not-applicable` comptent toujours — la chaîne a conclu sur une donnée
  réelle ;
- `not-evaluated` ne compte que si le tenant porte réellement une ressource du type visé.
  Sinon le verdict dit « ce tenant n'a rien de cette sorte », ce qui est vrai, utile, et
  n'éprouve **pas** la garde de capacité.

Sans ce filtre, six tenants paieraient quatre-vingt-dix-sept obligations au lieu de
cinquante, et la moitié le seraient par des absences. Un compteur qu'on fait baisser avec
des cases vides est pire qu'un compteur qui ne baisse pas : il déplace le faux vert dans le
tableau de bord.

## À lire à côté

- [Tracer les appels réels](tracing-api-calls.fr.md) : l'autre moitié, ce que le collecteur émet.
- [Limites connues](../known-limitations.fr.md) : la dette de véracité, comptée et publiée.
- [Plan Terraform ou collecte live](../concepts/terraform-vs-live.fr.md) : ce que chaque source peut conclure.
