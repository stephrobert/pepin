# ADR-0016 — Tout ce que Pépin publie est vérifiable, et la commande qui le vérifie est écrite

```yaml
status: Accepted
date: 2026-09-08
scope:
  - security
  - ci
  - release
```

## Contexte

Pépin audite la posture d'un cloud et vend l'**opposabilité** de ses verdicts. Un
outil qui affirme cela et dont la propre chaîne de publication ne se vérifie pas
n'a aucun argument : le premier auditeur qui regarde s'arrête là.

La chaîne existait déjà, et elle est solide — **79 actions sur 79 épinglées par SHA
complet**, permissions minimales par job, provenance SLSA, signature keyless cosign,
SBOM CycloneDX attesté, image distroless, rulesets de branche et de tag. Ce qui
n'existait pas, c'est son **énoncé**.

Le coût de ce silence s'est présenté sur la v0.3.0. Le SBOM `sbom.cdx.json` est
publié comme artefact de release et n'est dans **aucune** somme de `checksums.txt` —
alors qu'un commentaire du workflow affirme, à côté de la signature, que « une
signature sur checksums.txt couvre chaque artefact : leur empreinte y est ». Le
commentaire était faux et personne ne l'a vu, parce qu'aucune règle écrite ne disait
ce que « couvrir » devait vouloir dire.

## Décision

**Tout artefact publié est couvert par au moins une chaîne vérifiable, et la
documentation nomme la commande qui la vérifie.**

Les deux moitiés comptent autant l'une que l'autre. Une chaîne que personne ne sait
invoquer n'est pas auditable — elle est seulement présente.

Concrètement, ce qui est tenu :

| Mécanisme | Ce qu'il couvre |
|---|---|
| Épinglage par SHA complet | toute action tierce, sans exception |
| Version explicite | tout OUTIL qu'une action installe |
| Permissions minimales, déclarées par job | le jeton de chaque étape |
| `checksums.txt` + `cosign sign-blob` | l'empreinte de **chaque** artefact publié |
| Provenance SLSA (`attest-build-provenance`) | les binaires, comme sujets |
| Attestation SBOM (`attest-sbom`, CycloneDX) | le contenu du SBOM, lié aux binaires |
| Image distroless, sans shell ni gestionnaire | la surface d'exécution |
| Rulesets de branche et de tag | ce qui peut entrer et ce qui peut être publié |

## Justification

La signature d'un condensé de condensés (`checksums.txt`) est ce qui rend la
couverture **extensible sans nouvelle signature** : un artefact ajouté n'a qu'à
entrer dans le fichier. C'est aussi ce qui rend l'omission facile — d'où
l'invariant, et la porte de préflight qui le tient.

L'attestation SBOM est un mécanisme distinct et complémentaire : elle lie le
**contenu** du SBOM aux binaires, indépendamment du fichier publié. Les deux sont
gardés, parce qu'ils ne répondent pas à la même question. La somme dit « ce fichier
est celui qui a été publié » ; l'attestation dit « ce contenu décrit ce binaire ».

## Alternatives écartées

**Épingler les actions par étiquette de version** (`@v4`). Rejeté : une étiquette se
déplace, donc elle n'épingle rien. C'est le vecteur d'attaque le plus documenté de
l'écosystème.

**Épingler l'action et laisser flotter ce qu'elle installe.** C'était l'état réel, et
il a cassé toutes les PR le 2026-09-08 : `jdx/mise-action` était correctement épinglée
par SHA, mais sans `version:` elle résolvait vers la dernière étiquette de mise — et
`v2026.9.3` existait comme étiquette sans que ses binaires soient publiés. Cinq
tentatives, 404, tout le dépôt à l'arrêt.

Une chaîne ne vaut que son maillon le moins figé, et celui-là était choisi pour nous
chaque jour par le dernier qui taguait en amont. L'invariant porte donc désormais sur
l'outil, pas seulement sur l'action qui l'installe.

**Signer chaque artefact séparément.** Rejeté : autant de signatures à vérifier, et
un vérificateur qui en oublie une ne le saura pas. Un condensé unique signé donne
une seule commande, donc une commande qu'on exécute.

**Se reposer sur la seule attestation GitHub.** Rejeté : elle est excellente et elle
dépend d'un service. Une somme signée par cosign se vérifie hors ligne, avec un
bundle joint à la release.

**Publier le SBOM sans le couvrir**, au motif qu'il est attesté par ailleurs.
C'est exactement l'état constaté sur la v0.3.0, et il est refusé : l'attestation ne
dit rien du **fichier publié**, et le chemin de vérification documenté ne la
mentionnait pas. Un mécanisme correct et introuvable ne protège personne.

## Conséquences

Ajouter un artefact à une release coûte désormais une ligne dans `checksums.txt` et
une phrase dans la documentation. C'est peu, et c'est le prix pour que la promesse
reste vraie.

Le préflight refuse un tag si un artefact destiné à la publication n'est couvert par
aucune somme. La porte est mécanique, parce que l'attention ne l'est pas : ce défaut
a survécu à trois releases.

## Invariants

- Toute action tierce est épinglée par un SHA de 40 caractères.
- Toute action qui INSTALLE un outil lui donne une version explicite. L'épinglage
  d'une action ne fige que l'action : ce qu'elle télécharge ensuite reste choisi par
  l'amont si personne ne le dit.
- Tout artefact publié figure dans `checksums.txt`, donc sous la signature.
- La documentation donne, pour chaque artefact, la commande qui le vérifie.
- Aucun secret de fournisseur n'entre en CI (ADR-0012).

*Gardes : `TestEveryPublishedArtefactIsChecksummed`,
`TestEveryActionIsPinnedToAFullSHA` et `TestEveryInstallerActionPinsItsTool` ; le préflight rejoue la vérification avant tout
tag.*

## Validation

`mise run prepush` porte les deux tests. Le workflow de release vérifie ensuite la
signature sur ses propres artefacts avant de publier.

## Liens

ADR-0012 · issue #126 · `tools/release/preflight.sh` · `.github/workflows/release.yml`.
