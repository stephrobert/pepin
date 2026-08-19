# governance_provider_sovereignty : pas de montage Terraform, une décision

**Exigence** : CLD-GVN-4 (fournisseur établi dans l'UE, soustrait à un contrôle
capitalistique extra-UE déterminant ; exposition extraterritoriale évaluée).
**Fournisseur** : exoscale. **Sévérité** : high.

## Pourquoi ce n'est pas un module Terraform

Ce contrôle ne lit aucune ressource du tenant. Il évalue un type synthétique
`governance_provider`, produit une fois par scan depuis la section `souverainete` du
descripteur `providers/exoscale.yaml` : des faits sur le fournisseur lui-même, pas sur
ce que le client a déployé. Aucune ressource Terraform, chez aucun fournisseur, ne
change ces faits. Publier un `main.tf` ici laisserait croire qu'un `terraform apply`
corrige la situation, ce qui serait exactement le genre de fausse promesse que Pépin
existe pour éviter.

## Ce que la règle constate chez Exoscale

Deux écarts, chacun ancré sur des sources publiques et consigné dans le descripteur :

- **Siège hors Union européenne** : Akenes SA, siège en Suisse (`juridiction: CH`,
  `eu_etabli: false`). La Suisse relève de l'espace européen de confiance, pas de
  l'Union.
- **Contrôle capitalistique extra-UE déterminant** (`controle_capitalistique:
  extra_ue`) : Akenes SA → A1 Digital → A1 Telekom Austria Group, dont América Móvil
  (Mexique) détient 60,8 % et l'ÖBAG (État autrichien) 28,4 % (09/2025). Le contrôle
  déterminant est donc extra-UE.

Le descripteur porte aussi `exposition_extraterritoriale: false` : la chaîne CH/AT/MX
n'est pas soumise au Cloud Act états-unien, et aucune loi extraterritoriale mexicaine
équivalente n'est établie. C'est une nuance qui compte dans une analyse de risque, et
elle ne lève pas les deux écarts ci-dessus.

## Les seules remédiations réelles

1. **Retenir un fournisseur établi dans l'UE** pour la charge concernée, idéalement
   qualifié SecNumCloud. C'est la seule remédiation qui fait disparaître le finding,
   et elle est contractuelle, pas technique.
2. **Documenter une acceptation du risque**, motivée et datée, si l'exigence
   souveraine ne s'applique pas à cette charge. Pépin continuera de rendre `fail` :
   le rapport dit ce qui est, l'acceptation vit dans votre système de gestion des
   risques, pas dans le scanner. Un outil qui laisse taire un écart parce qu'on l'a
   accepté n'est plus opposable.

Un cas particulier ne change rien à ce qui précède : héberger dans une zone
européenne d'Exoscale (`at-vie-1`, `de-fra-1`…) satisfait `CLD-GVN-3` (localisation)
et **pas** `CLD-GVN-4` (établissement et contrôle du fournisseur). Les deux exigences
sont distinctes, et la première ne rachète pas la seconde.

## Sources

- `providers/exoscale.yaml`, section `souverainete` (faits et chaîne capitalistique
  sourcés : exoscale.com/about-us ; newsroom.a1.com ; répartition du capital d'A1
  Telekom Austria Group, 09/2025).
- `internal/commonrules/rules/governance_provider_sovereignty.rego` : ce que chacun
  des trois écarts (siège, capital, extraterritorialité) déclenche.
- `docs/controls/governance_provider_sovereignty.md` : la page générée du contrôle.
