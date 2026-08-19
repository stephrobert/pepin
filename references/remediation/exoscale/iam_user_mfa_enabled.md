# iam_user_mfa_enabled : réglage de compte, hors périmètre Terraform

**Exigence** : CLD-IAM-3 (MFA imposée sur les accès d'administration et la console).
**Fournisseur** : exoscale. **Sévérité** : high.

## Pourquoi ce n'est pas un module Terraform

Le provider Terraform `exoscale/exoscale` ne déclare **aucune ressource d'utilisateur**.
Son domaine IAM se limite à `exoscale_iam_role`, `exoscale_iam_api_key`,
`exoscale_iam_access_key` et `exoscale_iam_org_policy` (vérifié sur le schéma du
provider, `terraform providers schema -json`). La double authentification est un
réglage porté par le compte de la personne, activé depuis le portail : il n'existe pas
d'expression déclarative de ce réglage, et en fabriquer une donnerait une preuve qui
ne se déploie pas.

## La remédiation, telle qu'Exoscale la documente

Depuis le menu `Account` du portail, `Account Details`, onglet `Password and Security` :

1. `Set up two-factor verification`, puis saisir le mot de passe du compte.
2. Scanner le QR code avec une application d'authentification (TOTP), ou saisir le
   secret à la main.
3. Saisir le code rendu par l'application pour confirmer l'appairage.
4. **Conserver les codes de secours** affichés à l'issue de la configuration : sans
   eux, la perte de l'appareil rend le compte inaccessible, et le support n'a qu'une
   capacité limitée à rétablir l'accès.

Deux précautions que la documentation Exoscale souligne, et qui font la différence
entre une MFA activée et une MFA exploitable :

- enregistrer une **clé publique SSH** sur le compte : Exoscale s'en sert comme
  moyen de reprise, en faisant signer un défi avec la clé privée correspondante ;
- conserver le secret TOTP et les codes de secours dans le coffre de l'organisation,
  pas sur l'appareil qui porte l'application.

## Ce que Pépin peut en conclure

La règle lit l'attribut normalisé `mfa_enabled` du type `iam_user`, projeté depuis le
champ `two-factor-authentication` de l'utilisateur Exoscale. Elle ne se déclenche que
si l'attribut vaut explicitement `false` : un scan qui ne collecte pas ce champ ne
conclut pas `pass`, il n'évalue simplement rien. Autrement dit, ce contrôle demande une
**collecte live** ; un plan Terraform ne portera jamais l'information, puisque le
réglage n'y existe pas.

## Sources

- `references/docs/exoscale/platform-two-factor-auth.md` (procédure officielle,
  codes de secours, reprise par clé SSH).
- Schéma du provider `exoscale/exoscale` : absence de ressource utilisateur.
- `internal/commonrules/rules/iam_user_mfa_enabled.rego` ;
  `providers/exoscale.yaml` (contrat `iam_user`).
- `docs/controls/iam_user_mfa_enabled.md` : la page générée du contrôle.
