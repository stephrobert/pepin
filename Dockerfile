# L'image du scanner : `pepin scan` dans un conteneur, rien d'autre.
#
# Le binaire est COPIÉ, jamais compilé ici. Deux raisons, les mêmes que pour
# les binaires publiés :
#
#  1. Les octets de l'image sont les octets de la release. release.yml compile
#     `pepin-linux-<arch>` une fois, calcule ses empreintes, signe la liste et
#     atteste sa provenance ; une seconde compilation dans un Dockerfile serait
#     un second artefact sans aucune de ces propriétés.
#  2. Pas de seconde racine de confiance : compiler ici obligerait à épingler
#     et suivre une image `golang` pour refaire ce que la CI fait déjà avec la
#     toolchain qu'elle épingle déjà.
#
# La base est distroless/static, PAS scratch, et c'est mesuré : `pepin scan
# --live` parle en TLS aux API réelles des providers, donc l'image doit porter
# les certificats racines ; scratch casserait précisément le mode live, en
# échouant sur chaque poignée de main TLS (x509: certificate signed by unknown
# authority). distroless/static ajoute ~2 Mo : le magasin de CA, tzdata et un
# /etc/passwd avec l'utilisateur non root 65532. Pas de shell, pas de
# gestionnaire de paquets : un scanner qui lit des identifiants réels n'offre
# aucun outil à quiconque prendrait pied dedans. Épinglée par digest (l'index
# multi-arch) : un tag de base est une référence mutable.
#
# Limite assumée : `pepin verify --pubkey` exécute le binaire `cosign`
# (cmd/verify.go), absent d'une image sans shell. La vérification de signature
# d'un bundle se fait sur l'hôte ; `pepin verify` sans --pubkey (intégrité
# seule) fonctionne dans l'image.
#
# Les identifiants n'entrent JAMAIS dans l'image : ni ARG, ni COPY d'un profil.
# Ils se passent au lancement (`docker run -e OSC_ACCESS_KEY=...` ou
# --env-file), et seulement pour `--live`. Le mode Terraform et l'export JSON
# n'en demandent aucun.
#
# Construction depuis la racine du dépôt, binaire d'abord :
#
#   CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/pepin-linux-amd64 .
#   docker build -t pepin .
#   docker run --rm -v "$PWD:/work" pepin scan scaleway --terraform /work/plan.json
#
# Le build multi-arch de release passe TARGETOS/TARGETARCH par plateforme et un
# VERSION qui doit répondre au tag : release.yml refuse de pousser une image
# dont le `pepin version` n'est pas le tag publié.

FROM gcr.io/distroless/static-debian12:nonroot@sha256:1b7b9f0f0e0a1d2155f531db587cc48ec26aaf97ab64364225f5bf18a054e66a

# Prédéfinis par buildx par plateforme ; par défaut, un `docker build` nu marche.
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev

LABEL org.opencontainers.image.title="pepin" \
      org.opencontainers.image.description="CSPM scanner for the European sovereign clouds (Exoscale, Outscale, Scaleway). Scans a Terraform plan, a JSON export, or the live API; exit codes 0 compliant, 1 non-compliance, 2 error." \
      org.opencontainers.image.source="https://github.com/stephrobert/pepin" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${VERSION}"

COPY dist/pepin-${TARGETOS}-${TARGETARCH} /pepin

# Explicite même si la base :nonroot le porte déjà : un scanner n'a aucune
# raison d'être root, il lit un fichier monté ou une API distante et écrit au
# pire un bundle dans /work. Numérique, parce qu'une policy d'admission qui
# exige runAsNonRoot ne sait pas résoudre un nom.
USER 65532:65532

# Le point de montage documenté : l'entrée (plan.json, inventaire) s'y monte,
# le bundle de preuve (--seal) s'y écrit. Appartient à root dans la base, d'où
# un volume monté en pratique : -v "$PWD:/work".
WORKDIR /work

ENTRYPOINT ["/pepin"]

# Sans argument : l'aide, code de sortie 0. Un conteneur lancé nu documente
# comment s'en servir au lieu d'échouer.
CMD ["--help"]
