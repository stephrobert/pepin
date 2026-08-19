> [🇬🇧 English](install.md) · 🇫🇷 Français

# Installer Pépin

Un binaire Go, pas de démon, pas de dépendance. Quatre portes d'entrée,
chacune vérifiée avant que quoi que ce soit s'exécute : le binaire publié,
l'image de conteneur, l'action GitHub, le modèle GitLab. `0.1.0` ci-dessous
nomme une version publiée : un `latest` mutable installe ce qui est le plus
récent, c'est-à-dire un binaire que personne ne sait nommer après coup.

## Le binaire publié

Demande `cosign` (ou `gh`, plus bas) et rien d'autre. Chaque fichier est
téléchargé sur disque et vérifié avant d'être exécuté.

```bash
base=https://github.com/stephrobert/pepin/releases/download/v0.1.0

case "$(uname -s)-$(uname -m)" in
  Linux-x86_64)  asset=pepin-linux-amd64 ;;
  Linux-aarch64) asset=pepin-linux-arm64 ;;
  Darwin-x86_64) asset=pepin-darwin-amd64 ;;
  Darwin-arm64)  asset=pepin-darwin-arm64 ;;
  *) echo "pas de binaire publié pour $(uname -s)-$(uname -m)" >&2; exit 1 ;;
esac

curl -fsSLO "$base/$asset"
curl -fsSLO "$base/checksums.txt"
curl -fsSLO "$base/checksums.txt.cosign.bundle"

# Qui a produit la liste d'empreintes, avant de croire une seule empreinte
# dedans. L'identité nomme le workflow de release et la ref du tag, pas le
# dépôt : un motif large accepterait n'importe quel workflow ayant un jour
# obtenu id-token: write.
cosign verify-blob --bundle checksums.txt.cosign.bundle \
  --certificate-identity-regexp '^https://github\.com/stephrobert/pepin/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt

sha256sum -c checksums.txt --ignore-missing

install -m 0755 "$asset" ~/.local/bin/pepin
```

Avec `gh`, qui contrôle la provenance de build à la place : elle prouve quel
workflow et quel commit ont produit le binaire.

```bash
gh release download v0.1.0 --repo stephrobert/pepin --pattern 'pepin-linux-amd64'
gh attestation verify pepin-linux-amd64 --repo stephrobert/pepin \
  --signer-workflow stephrobert/pepin/.github/workflows/release.yml
```

Compiler depuis les sources demande une toolchain Go et renonce à tous les
contrôles ci-dessus, puisque rien n'est signé avant d'être publié :

```bash
go install github.com/stephrobert/pepin@latest
```

## L'image de conteneur

Pour la CI qui consomme une image plutôt qu'un binaire. Le binaire dedans est
le binaire publié : rien n'est compilé dans le Dockerfile, donc les
empreintes, le SBOM et la provenance de la release décrivent aussi le contenu
de l'image. La base est distroless (certificats racines pour le TLS de
`--live`, utilisateur 65532, pas de shell, pas de gestionnaire de paquets),
et le workflow de release refuse de pousser une image dont le `pepin version`
n'est pas le tag ou dont les codes de sortie ont bougé.

Un tag par release, pas de `latest`. Vérifier, puis lancer :

```bash
cosign verify ghcr.io/stephrobert/pepin:v0.1.0 \
  --certificate-identity-regexp '^https://github\.com/stephrobert/pepin/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com

# auditer un plan Terraform : aucun identifiant, rien de provisionné
docker run --rm -v "$PWD:/work" ghcr.io/stephrobert/pepin:v0.1.0 \
  scan scaleway --terraform /work/plan.json
```

Trois points pratiques, tous mesurés :

- **Les identifiants n'entrent jamais dans l'image.** Pour `--live`, passer
  les variables natives du provider au lancement et rien d'autre :
  `docker run --rm -e OSC_ACCESS_KEY -e OSC_SECRET_KEY -e OSC_REGION ghcr.io/stephrobert/pepin:v0.1.0 scan outscale --live`
  (nommer les variables sans `=` les transmet depuis votre environnement,
  sans mettre leur valeur dans la ligne de commande ni dans l'historique du
  shell).
- **Sceller un bundle dans un volume monté demande votre uid**, parce que
  l'image tourne en 65532 et que le volume vous appartient :
  `docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/work" ... scan ... --seal /work/bundle --redact`.
- **`pepin verify --pubkey` ne fonctionne pas dans l'image** : il exécute le
  binaire `cosign`, qu'une image distroless n'a volontairement pas. Vérifier
  les signatures sur l'hôte ; `pepin verify` sans `--pubkey` (intégrité
  seule) fonctionne partout.

## GitHub Actions

L'action composite télécharge le binaire publié, **vérifie son SHA-256 contre
la liste d'empreintes de la release avant de rien exécuter** (un job de CI de
ce dépôt corrompt un octet de ce téléchargement et exige le refus), scanne,
et traduit les codes de sortie en porte de CI :

```yaml
- uses: stephrobert/pepin/.github/actions/pepin-scan@v0.1.0
  with:
    version: 0.1.0
    provider: scaleway
    terraform-plan: plan.json
```

`fail-on-nonconformity: 'false'` rapporte la posture (l'exit 1 devient un
avertissement) au lieu d'en faire une porte ; une erreur technique (exit 2)
fait échouer le job quoi que dise cette entrée, parce qu'une erreur avalée
est une posture que personne n'a mesurée. Les identifiants ne sont jamais des
entrées de l'action : pour `live: true`, mettre les variables natives du
provider dans `env:` depuis les secrets du dépôt. Exemple complet :
[`examples/github-actions/pepin.yml`](../examples/github-actions/pepin.yml).

## GitLab CI

Même doctrine, en modèle incluable : binaire vérifié en `before_script`,
codes de sortie comme contrat, mode rapport via
`allow_failure: exit_codes: [1, 3]` (jamais 2) :

```yaml
include:
  - remote: 'https://raw.githubusercontent.com/stephrobert/pepin/v0.1.0/examples/gitlab-ci/pepin.gitlab-ci.yml'
```

Exemple complet : [`examples/gitlab-ci/`](../examples/gitlab-ci/).

## Les identifiants, dans tous les modes

Pépin lit les variables d'environnement **natives** de chaque provider (ou
son fichier de configuration natif) : un identifiant qui marche déjà pour le
CLI du provider marche ici, et rien n'est renommé en chemin.

| provider | environnement |
|---|---|
| Scaleway | `SCW_ACCESS_KEY`, `SCW_SECRET_KEY`, `SCW_DEFAULT_ORGANIZATION_ID`, `SCW_DEFAULT_REGION` |
| Outscale | `OSC_ACCESS_KEY`, `OSC_SECRET_KEY`, `OSC_REGION` |
| Exoscale | `EXOSCALE_API_KEY`, `EXOSCALE_API_SECRET`, `EXOSCALE_ZONE` |

Seul `--live` en a besoin. Le mode plan Terraform et le mode inventaire
tournent sans aucun compte, et c'est pourquoi chaque exemple de CI commence
par une porte sur le plan. Pépin n'imprime jamais un secret ; ce qui mérite
votre attention, c'est le **bundle de preuve** (`--seal`) : il embarque
l'inventaire évalué d'un tenant réel, et un user-data ou un document de
policy de cet inventaire peut porter les secrets mêmes que les règles
détectent. En CI, sceller avec `--redact` (le défaut de l'action GitHub),
garder une rétention d'artefact courte, et se souvenir que les artefacts d'un
projet public se téléchargent. Ne retirer `--redact` que si un bundle doit
supporter `pepin verify --re-derive`, et traiter alors le bundle comme un
secret.
