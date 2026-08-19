> [🇬🇧 English](RELEASING.md) · 🇫🇷 Français

# Publier une release

Une release est un tag. Il n'y a de version à monter dans aucun fichier :
`release.yml` grave le tag dans le binaire via `-ldflags`
(`github.com/stephrobert/pepin/cmd.version`), et un binaire construit autrement
répond `pépin 0.1.0-dev`. Rien ne peut dériver du tag, parce que rien d'autre
ne porte le numéro.

## Quel numéro

Les commits décident, pas le goût. Les sujets suivent les
[Conventional Commits](https://www.conventionalcommits.org/fr/v1.0.0/), et
commitizen en dérive l'incrément (`.cz.toml`, `version_provider = "scm"` :
les tags sont la seule source) :

```bash
uvx --from 'commitizen==4.16.5' cz bump --dry-run   # ex. « tag to create: v0.2.0 »
```

`fix` monte le patch, `feat` le mineur, un `!` marque la rupture, et
`major_version_zero` garde une rupture dans `0.x`. Le preflight ci-dessous
vérifie que le tag demandé est celui que les commits impliquent, et rapporte
plutôt qu'il ne refuse quand commitizen est inaccessible : un mainteneur sans
l'outillage Python doit pouvoir couper une release.

## En couper une

1. **Déplacer les entrées `Unreleased`** de [CHANGELOG.md](./CHANGELOG.md)
   **et** de [CHANGELOG.fr.md](./CHANGELOG.fr.md) sous un nouveau titre
   `## [X.Y.Z]`, dans les deux fichiers. Le workflow de release lit la section
   anglaise pour le corps de la GitHub Release ; le preflight refuse une
   version présente dans une langue et absente de l'autre, parce que le dépôt
   promet leur synchronisation.

2. **Merger vers `main` par une pull request, et attendre une CI verte.** Le
   tag se construit depuis ce commit, et une release publiée ne se rejoue pas.

3. **Lancer le preflight**, qui rejoue hors ligne tout ce qui doit tenir :

   ```bash
   mise run release-check -- v0.1.0
   ```

   Il vérifie : un arbre propre sur `main` ; le tag libre en local *et* sur
   origin ; `mise run test` (Go avec `-race`, les suites Rego et les gardes de
   surface gelée), `mise run validate` et `mise run vet` ; **zéro dérive
   SCSL** (voir plus bas) ; les codes de sortie répondus par le **binaire
   construit** sur les inventaires d'exemple plutôt que relus dans une
   constante ; un bundle scellé par ce binaire qui se vérifie, se re-dérive,
   **et se fait refuser dès qu'un octet est altéré** (un `verify` qui accepte
   tout ne prouve rien) ; la version que les commits impliquent ; et les deux
   CHANGELOG portant la section. Il rapporte chaque verdict au lieu de
   s'arrêter au premier, et n'imprime les commandes de tag que quand tout
   tient.

   Il ne relance volontairement **pas** `mise run audit` ni le contrôle de
   schéma OSCAL du NIST : ils demandent le réseau et des outils que la machine
   peut ne pas avoir. La CI les fait tourner à chaque push ; le preflight lui
   demande si elle l'a fait, sur ce commit exact.

4. **Taguer et pousser** :

   ```bash
   git tag -a v0.1.0 -m "v0.1.0"
   git push origin v0.1.0
   ```

Pousser le tag est ce qui publie. Cela ne s'annule pas discrètement : un tag se
supprime des deux côtés, et une release qui a atteint le monde a été
téléchargée.

### Ce qui reste manuel

- `framework-scsl` doit être cloné à côté du dépôt : le contrôle de dérive
  SCSL lit l'index vivant, et une release coupée sans lui est refusée plutôt
  que coupée en aveugle.
- `gh` doit être authentifié pour que le preflight interroge la CI.
- Déplacer les entrées de CHANGELOG est de l'écriture, pas de la génération,
  dans les deux langues.

## Ce que le tag déclenche

`.github/workflows/release.yml`, sur `v*` :

- construit les binaires `linux` et `darwin` pour `amd64` et `arm64`, tag
  gravé dedans, et **refuse de publier** un binaire qui ne répond pas le tag à
  `version` ou qui a perdu les codes de sortie documentés,
- génère les empreintes SHA-256 et un SBOM CycloneDX (Syft),
- enregistre la **provenance de build SLSA** et atteste le SBOM,
- signe les empreintes en **Cosign keyless**,
- construit et pousse l'**image de conteneur** (`ghcr.io/stephrobert/pepin`,
  un tag par release, pas de `latest`) depuis les binaires linux ci-dessus,
  rien n'étant compilé dans le Dockerfile, après avoir refusé une image dont
  le `pepin version` n'est pas le tag ou dont les codes de sortie ont bougé à
  travers `docker run` ; l'image reçoit sa propre provenance SLSA, sa propre
  attestation de SBOM et une signature keyless sur son digest,
- crée la GitHub Release avec chaque artefact attaché, dont
  `provenance.intoto.jsonl`, le fichier que le contrôle *Signed-Releases*
  d'OpenSSF Scorecard cherche, distinct de l'attestation enregistrée via
  l'API GitHub,
- puis éprouve l'action composite (`.github/actions/pepin-scan`) contre la
  release qui vient d'être publiée, seul moment où ce test peut exister, et
  exige le contrat de codes documenté : 0 passe, 1 fait échouer le job, 1 ne
  fait qu'avertir sous `fail-on-nonconformity: 'false'`, et 2 échoue quoi que
  dise cette entrée.

N'importe qui peut alors vérifier qu'un binaire vient bien du workflow de ce
dépôt :

```bash
gh release download v0.1.0 --repo stephrobert/pepin --pattern 'pepin-linux-amd64' --pattern 'checksums.txt*'
gh attestation verify pepin-linux-amd64 --repo stephrobert/pepin

cosign verify-blob --bundle checksums.txt.cosign.bundle \
  --certificate-identity-regexp 'https://github.com/stephrobert/pepin/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
sha256sum -c checksums.txt --ignore-missing
```

## Ce dont vous pouvez dépendre, et pour combien de temps

Les surfaces ci-dessous sont le tout. **Ce qui n'est pas sur cette liste ne
porte aucune promesse** ; en particulier, l'*ensemble* des contrôles grandit à
chaque release, et un verdict peut légitimement changer quand une règle cesse
d'être fausse ; les deux ont leur ligne de CHANGELOG, aucun n'est une rupture.

| surface | signal de version | gelée dans |
|---|---|---|
| les verbes, flags et codes de sortie de la CLI | `cliSurfaceVersion` (`cmd/surface.go`) | `cmd/testdata/frozen/cli.json` |
| `--format json` (`findings` + `summary`) | `findingsSurfaceVersion` (`cmd/surface.go`) | `cmd/testdata/frozen/findings.json` |
| le document assessment (`--format assessment`, `assessment.json` d'un bundle) | `assess.AssessmentSurfaceVersion` | `cmd/testdata/frozen/assessment.json` |
| le bundle de preuve (fichiers, rôles, manifest) | le suffixe `/vN` du champ `format` de `manifest.json` (`assess.BundleFormat`) | `cmd/testdata/frozen/bundle.json` |
| `--format oscal` | OSCAL 1.1.2 lui-même | le contrôle de schéma NIST dans `ci.yml` |
| les identifiants `code` des contrôles | (aucun) | `referentiel/controles.yaml`, gardé par `mise run validate` |

Ce qui est gelé est la **forme** (chemins de champs et types JSON, noms de
verbes et de flags), jamais une valeur. Deux tests la gardent à chaque
`mise run test` : `TestTheFrozenSurfacesStillMatchTheirFixture` échoue quand
une forme bouge sans sa fixture, et `TestASurfaceChangeDemandsItsVersionBump`
échoue quand la fixture a bougé sans la version déclarée.

Changer une de ces surfaces à dessein tient en quatre pas, dans un seul
commit : changer le code ; `mise run frozen-update` (il ajoute à l'historique
de la fixture, à la version suivante, sans jamais réécrire une entrée) ;
incrémenter la constante correspondante (les tests restent rouges jusqu'à ce
pas, et c'est le but) ; écrire la ligne de CHANGELOG dont l'incrément est le
signal.

### Le signal qu'un consommateur lit

Seul le bundle porte sa version sur le fil : le champ `format` de
`manifest.json`, `pepin-assessment-bundle/vN`. Un vérificateur qui rencontre
une version qu'il ne connaît pas doit **s'arrêter plutôt que deviner**. Les
autres surfaces n'ont pas de champ pour la porter : leur signal est ce
tableau, les constantes, et la ligne de CHANGELOG que chaque incrément doit.

### Le préavis

**Une release mineure.** Un champ, un verbe ou un code de sortie qui va
disparaître est déprécié dans une release et retiré au plus tôt dans la
suivante, et les deux événements sont des lignes des CHANGELOG. Une release
plutôt qu'un nombre généreux, à dessein : c'est un projet 0.x à un seul
mainteneur, et une promesse d'une version mineure, tenue, vaut plus qu'une
plus longue qui serait rompue.

## La dérive amont : l'index SCSL

Les contrôles de Pépin mappent les exigences CLD gelées du framework SCSL, et
cet index vit hors de ce dépôt et bouge sous lui. Un contrôle dont la
référence normative a été reformulée en amont sans que personne ne retrie est
le même défaut qu'une opération non triée dans n'importe quel scanner : le
rapport citerait une référence qui ne dit plus ce que le mapping supposait.

```bash
mise run scsl-drift          # exit 2 si l'index vivant a bougé depuis la baseline
mise run scsl-drift-update   # réécrit la baseline, après triage humain
```

La baseline (`referentiel/scsl-baseline.json`) est committée : le savoir de ce
qui a été trié voyage avec le dépôt. La convention de sortie de l'outillage
(0 rien à signaler, 1 erreur, **2 dérive non triée**) est volontairement
distincte de celle de `pepin scan` (`0` conforme, `1` non-conformité, `2`
erreur technique) : l'une parle à une porte de release, l'autre à une porte de
conformité, et le preflight les distingue.

## À partir de la deuxième release

Un contrôle ne peut pas exister avant une première release, et le dire vaut
mieux qu'une porte verte : rejouer la release précédente contre celle-ci. Le
risque qu'il ferme est celui de l'auditeur : un bundle scellé par la release
N−1 doit encore se vérifier et se re-dériver sous la release N, sinon le
changement qui l'a cassé doit être une ligne de CHANGELOG et un incrément du
format de bundle. Dès que `v0.1.0` existe, avant chaque tag :

```bash
git worktree add /tmp/pepin-prev v0.1.0 && (cd /tmp/pepin-prev && mise run build)
/tmp/pepin-prev/pepin scan scaleway examples/scaleway/inventory.json --seal /tmp/bundle-prev || true
./pepin verify /tmp/bundle-prev --re-derive   # le NOUVEAU binaire lit le VIEUX bundle
git worktree remove /tmp/pepin-prev
```

Quand cette commande méritera l'automatisation, sa place est dans le
preflight, à côté de l'aller-retour sur bundle frais.

## Versionnement

[Semantic Versioning](https://semver.org/). Avant 1.0, le mineur bouge sur
tout ce qu'un consommateur peut observer sur les surfaces ci-dessus.

Ce qui compte comme rupture est plus étroit qu'il n'y paraît. **Couvrir plus**
(nouveaux contrôles, nouveaux providers, nouveaux attributs collectés) n'est
jamais une rupture : un tenant qui gagne des findings a gagné de la
couverture, et la porte `--strict` existe précisément pour les pipelines qui
veulent le voir. **Un verdict corrigé pour coller au contrat réel du provider
est un correctif, pas une rupture**, même quand un pipeline dépendait du
verdict faux : un test qui reposait sur un faux `pass` mesurait Pépin plutôt
que le cloud. Ce qui casse est du côté de ce projet : les formes, codes,
verbes et formats du tableau ci-dessus.
