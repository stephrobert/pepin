> [🇬🇧 English](README.md) · 🇫🇷 Français

# Pépin

[![CI](https://github.com/stephrobert/pepin/actions/workflows/ci.yml/badge.svg)](https://github.com/stephrobert/pepin/actions/workflows/ci.yml)
[![CodeQL](https://github.com/stephrobert/pepin/actions/workflows/codeql.yml/badge.svg)](https://github.com/stephrobert/pepin/actions/workflows/codeql.yml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/stephrobert/pepin/badge)](https://securityscorecards.dev/viewer/?uri=github.com/stephrobert/pepin)
[![Licence : Apache-2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](go.mod)
[![OSCAL 1.1.2](https://img.shields.io/badge/OSCAL-1.1.2-1f6feb.svg)](https://pages.nist.gov/OSCAL/)
[![SecNumCloud 3.2](https://img.shields.io/badge/SecNumCloud-3.2-0b3d91.svg)](https://cyber.gouv.fr/secnumcloud-pour-les-fournisseurs-de-services-cloud)

**Pépin trouve les pépins de votre cloud souverain.**

![Pépin scannant un plan Terraform volontairement non conforme, puis le même module corrigé : le verdict et le code de sortie changent](docs/assets/quickstart.gif)

*Chaque commande de cet enregistrement s'exécute réellement. Il est régénéré à
chaque release depuis `tools/demo/quickstart.tape`, et le preflight refuse de
taguer une version que le GIF ne montre pas.*

Pépin est un scanner de posture (CSPM) pour les clouds **souverains européens**
(Exoscale, Outscale, Scaleway). Il évalue la configuration effective d'un tenant
contre un référentiel commun ancré sur **SCSL**, **SecNumCloud 3.2**, **CIS
Controls v8** et **ISO/IEC 27001:2022 / 27017**, et produit un résultat
**opposable** : un statut typé par contrôle, ses références normatives exactes,
et un dossier de preuve scellé.

## Ce qui distingue Pépin

- **Souverain d'abord** : vocabulaire natif de chaque provider (Net/Subnet/EIM,
  BSU/OOS, Kapsule…), jamais par comparaison à un hyperscaler non souverain.
- **Résultat opposable, pas juste des findings** : chaque contrôle est
  `pass` / `fail` / `not-applicable` (justifié) / `not-evaluated` (avec la raison).
  Un « pass » n'est affirmé que si la donnée nécessaire a réellement été
  collectée : « aucun finding » n'est jamais confondu avec « conforme ».
- **Deux sources** : collecte **live** via l'API du provider, ou audit d'un
  **plan Terraform** (`terraform show -json`) — sans rien provisionner.
- **Dossier de preuve** : `--seal` écrit un bundle horodaté (inventaire évalué,
  assessment, **OSCAL 1.1.2**, manifeste à empreintes) que `pepin verify`
  recontrôle, avec vérification de **signature cosign** optionnelle.

## Démarrage

Les binaires publiés (empreintes, signature, provenance SLSA), l'image de
conteneur, l'action GitHub et le modèle GitLab sont documentés dans
[docs/install.fr.md](docs/install.fr.md). Depuis les sources :

```bash
# build (Go 1.26+)
go build -o pepin .

# auditer un plan Terraform (aucune ressource provisionnée)
terraform plan -out tfplan && terraform show -json tfplan > plan.json
./pepin scan scaleway --terraform plan.json

# collecte live (identifiants via l'environnement / config native du provider)
./pepin scan outscale --live --region eu-west-2

# dossier de preuve scellé + vérification
./pepin scan scaleway --terraform plan.json --seal ./bundle
./pepin verify ./bundle --pubkey cosign.pub
```

Formats de sortie : `--format table|json|assessment|oscal|sarif`.
Codes de sortie : `0` conforme · `1` non-conformité · `2` erreur · `3` rien de mesuré
ou, avec `--strict`, écarts medium/low restants (exploitables en CI). Un scan qui n'a
collecté aucune ressource ne rend jamais `0` : un résultat vide n'est pas un résultat conforme.

## Documentation

- [Démarrage rapide](docs/getting-started/quickstart.fr.md) : cinq minutes, aucun compte
  cloud, un échec réel, sa correction, et un second scan qui dit autre chose.
- [Lire un scan](docs/getting-started/understanding-a-scan.fr.md) : un run réel, commenté
  ligne à ligne, jusqu'au code de sortie.
- [Le modèle d'assessment](docs/concepts/assessment-model.fr.md) : ce que `pass`, `fail`,
  `not-applicable` et `not-evaluated` affirment réellement.
- [Matrice de couverture](docs/coverage.fr.md) : ce qui est mesurable, par fournisseur et par
  source. **Générée** depuis le référentiel et les descripteurs, vérifiée en CI.
- [Limites connues](docs/known-limitations.fr.md) : les angles morts, nommés.
- [Périmètre et non-objectifs](docs/concepts/scope.fr.md) : ce qu'un rapport Pépin n'est pas.
- [Installation](docs/install.fr.md) · [Feuille de route](docs/roadmap.md) (document de
  travail interne)

## Architecture (en bref)

- `internal/collect` : moteur de collecte déclaratif (specs YAML → modèle normalisé).
- `providers/*.yaml` : descripteurs d'un provider (auth, collecte live, mapping
  Terraform, contrat d'API). Un provider = trois sources dans un seul fichier YAML.
- `internal/commonrules/rules/*.rego` : règles **communes** (OPA/Rego) qui évaluent
  le modèle normalisé, indépendamment du provider.
- `referentiel/` : le référentiel de contrôles (code neutre → sévérité, SCSL,
  correspondances de normes, fournisseurs). Source de vérité, testée contre
  l'invention de références.
- `internal/assess` : construit l'assessment opposable (statuts, preuves,
  provenance) et le bundle scellé (OSCAL, empreintes, cosign).

Voir [CONTRIBUTING.md](CONTRIBUTING.md) pour ajouter un provider ou une règle, et
[SECURITY.md](SECURITY.md) pour la divulgation de vulnérabilités.

## Portée

Pépin évalue la configuration d'un **tenant** (côté commanditaire). Les
correspondances normatives citées (SecNumCloud, ISO, CIS) sont **indicatives** :
un rapport Pépin n'est **pas** une preuve de qualification ou de certification,
lesquelles portent sur le **prestataire** de service cloud, pas sur un scan de
tenant.

## Licence

Apache-2.0 — voir [LICENSE](LICENSE) et [NOTICE](NOTICE).
