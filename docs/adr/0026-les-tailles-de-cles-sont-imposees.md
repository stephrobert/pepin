# ADR-0026 — Les tailles de clés sont imposées sur chaque chemin cryptographique

```yaml
status: Accepted
date: 2026-11-07
scope:
  - security
  - collect
  - release
```

## Contexte

ADR-0025 a fermé le périmètre TLS, pas celui du critère OpenSSF `crypto_keylength`.
Les témoins restants ont montré que SigV4 et EXO2 acceptaient des secrets de
13 octets, et que `verify` déléguait une clé RSA 1024 à Cosign sans garde locale.

## Décision

`internal/cryptopolicy` porte les seuils communs : RSA 2048, EC 224 ou Ed25519,
et au moins 14 octets effectifs de secret HMAC, sans compter les préfixes constants.
TLS et signatures réutilisent la même validation de clé publique. La préparation
du fournisseur, les signataires directs et le SDK S3 appliquent la garde HMAC.

La taille du matériau n'est **pas** une mesure d'entropie. Les secrets doivent
venir d'une génération cryptographique sûre ; ni leur diversité textuelle ni leur
longueur ne permettent de le déduire. Les identifiants porteurs/en-tête ne sont pas
des paramètres de clé sélectionnés pour une primitive cryptographique.

`verify --pubkey` lit une clé publique PEM locale, la valide puis transmet à
Cosign un instantané canonique à permissions privées. Un chemin distant/KMS ne
permet pas cette garde sans importer un résolveur : son opérateur exporte d'abord
la clé publique en PEM. La vérification de signature reste confiée à Cosign.

La release conserve son Cosign keyless épinglé : son défaut P-256/SHA-256 est
vérifié dans le code public de la version utilisée et protégé contre une
substitution de clé ou d'algorithme dans le workflow.

Cette décision complète ADR-0025 sans changer son choix de compatibilité HTTP.
Elle ne modifie pas le défaut de vérification d'intégrité non authentifiée :
`--require-signature` reste nécessaire lorsqu'une signature est exigée.

## Alternatives écartées

- Déclarer le critère satisfait avec TLS seulement : laisse deux mécanismes ouverts.
- Estimer l'entropie par une expression régulière ou la diversité des caractères :
  confond une apparence avec une génération aléatoire.
- Valider un fichier puis faire relire son chemin original par Cosign : permet
  son remplacement entre la garde et la vérification.
- Importer les bibliothèques Cosign/KMS dans le binaire : rouvre la dépendance
  lourde écartée par l'architecture du vérificateur ; le PEM public suffit.

## Invariants

- 13 octets sont refusés, 14 acceptés : `TestRequestMACRejectsSecretsBelow112Bits`,
  `TestProviderRejectsWeakMACSecretsBeforeCollection`,
  `TestS3RejectsShortSigningSecretBeforeSDKRequest`.
- La taille est celle des octets réellement signés, pas des caractères ni une
  estimation d'entropie : `TestHMACKeySizeIsMeasuredInBytesNotCharactersOrEntropy`.
- Les clés publiques faibles sont refusées avant Cosign :
  `TestVerifyRefusesWeakSignatureKeyBeforeCosign`, `TestPublicPEMKeyStrength`.
- Cosign lit l'instantané validé, nettoyé après usage :
  `TestCosignReadsTheValidatedKeySnapshot`.
- Un vérificateur annulé ne réussit pas :
  `TestCancelledSignatureVerificationIsNotSuccess`. Annulation et délai dépassé
  restent identifiables après le démarrage du sous-processus :
  `TestSignatureVerificationPreservesCancellationAfterCosignStarts`.
- La release conserve ses clés keyless par défaut :
  `TestReleaseUsesDefaultKeylessCosignKeys`, y compris les options sur des lignes
  continuées : `TestKeylessGuardChecksContinuedSigningOptions`.
- Le binaire vérifie de vraies signatures RSA 2048/P-256 et refuse leur altération :
  `TestNativeCosignKeyStrengthAndSignature`, témoin opt-in sans identifiant cloud.

## Validation

`mise run test`, `mise run audit` et le témoin natif documenté dans `SECURITY.md`.
Le témoin génère ses clés jetables ; il n'utilise ni compte ni ressource cloud.
Cosign peut publier le digest synthétique et sa signature dans son journal public.

## Liens

Issue #281 · ADR-0012 · ADR-0016 · ADR-0025 · `SECURITY.md` et `SECURITY.fr.md`.
