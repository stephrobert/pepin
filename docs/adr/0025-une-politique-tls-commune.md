# ADR-0025 — Une politique TLS commune, sans confondre TLS et identifiants

```yaml
status: Accepted
date: 2026-11-07
scope:
  - security
  - collect
```

## Contexte

Les témoins de l'issue #281 ont montré que le client Kubernetes acceptait RSA
1024 pour sa clé client et pour un serveur de confiance. REST et S3 utilisaient
d'autres constructions HTTP. Une garde limitée au kubeconfig laisserait les
autres collecteurs sans la même exigence.

## Décision

Kubernetes, REST, EIM, OKS et S3 utilisent `internal/tlspolicy` : TLS 1.2 minimum,
AEAD/ECDHE en TLS 1.2, RSA 2048 minimum, EC 224 minimum, ou Ed25519. Go fournit
TLS 1.3 et les primitives. La garde examine les clés de toute chaîne client
fournie et d'au moins une chaîne serveur déjà validée par Go.

La vérification habituelle de confiance, d'expiration et de nom reste active.
Le transport REST/S3 conserve les réglages HTTP standard, notamment le proxy.
Les échecs de collecte restent enregistrés dans l'inventaire.

Le mainteneur a choisi de conserver HTTP, notamment pour MinIO local. Cette
compatibilité n'est pas une protection TLS. La robustesse des secrets API et
des clés de signature n'est pas déduite de la preuve TLS ni de la longueur
textuelle des identifiants.

## Alternatives écartées

- Une garde Kubernetes seulement : laisse REST et S3 divergents.
- Désactiver la confiance puis vérifier seulement la taille des clés : accepte
  un serveur non autorisé.
- Interdire HTTP dans ce correctif : changement de compatibilité non retenu.
- Marquer `crypto_keylength` satisfait sur la seule preuve TLS : omet les
  autres mécanismes du critère.

## Invariants

- RSA 1024 est refusé, RSA 2048 et P-256 restent acceptés :
  `TestClientKeyStrength`, `TestServerKeyStrength`,
  `TestRESTAndS3UseTheSameTLSKeyPolicy`.
- Toute chaîne de confiance utilisée est forte :
  `TestEveryCertificateInATrustedPathMustBeStrong`.
- La confiance n'est pas désactivée et le transport global n'est pas modifié :
  `TestPolicyIsIndependentAndDoesNotDisableTrust`.
- Un refus REST est enregistré comme incomplet avant toute requête HTTP :
  `TestRESTAndS3UseTheSameTLSKeyPolicy`.
- Les messages de refus existent dans les deux langues :
  `TestRefusalsAreBilingual`.

## Validation

`mise run test` et `mise run audit`. Les témoins sont locaux, sans identifiant
cloud ni ressource provisionnée, conformément à ADR-0012.

## Liens

Issue #281 · ADR-0012 · ADR-0016 · ADR-0023 · `SECURITY.md` et `SECURITY.fr.md`.
