# Roadmap pepin — vers la parité pavois + un fond opposable

> **Document de travail interne, en français, hors documentation produit.** La feuille
> de route publique est [`ROADMAP.md`](../ROADMAP.md) (anglais primaire, directionnelle,
> sans verdict d'audit) ; les limites qui concernent un utilisateur vivent dans
> [`docs/known-limitations.md`](../docs/known-limitations.md). Ce fichier garde ce qui
> n'a pas sa place dans l'une ni dans l'autre : verdicts d'audit bruts, faux verts
> constatés, bugs moteur, notes de séquencement. Rien ici n'est un engagement, et rien
> ici n'est vérifié par la CI.
>
> Consolide les **5 audits** de juillet 2026 (complétude du corpus + Exoscale + Outscale
> + Scaleway, relus champ par champ contre les SDK/OAPI officiels) et le plan de parité
> produit.

## Avancement (journal)

- ✅ **Renommage** `ANSSI SecNumCloud 3.2` + description recadrée.
- ✅ **Axe 2 §1** (corrections sans scan) : souveraineté Exoscale `extra_ue`, raison N/A
  KMS réécrite, `iam_no_root_access_key` → `requiredAttr` (faux-PASS live → NotEvaluated),
  `loadbalancer_http_redirect_to_https` N/A Outscale + dormant, `blockstorage_snapshot_not_public`
  N/A Scaleway, purges de zones/commentaires. *Validé : build + 193 Rego + go tests + scan démo.*
- ✅ **Axe 2 §0 (partiel, sans compte)** : protocole IP **numérique** Outscale (F7 :
  `6/17/1` → tcp/udp/icmp, faux négatif SSH ouvert) + `volume_in_use` Scaleway `in_use`
  (underscore). *Validés par tests dédiés (moteur + Rego).*
- ✅ **Scan LIVE Outscale cloudgouv** (compte `souverain`, lecture seule) : réussi.
  Baseline honnête `fail:36 · pass:16 · N/A:4 · not-evaluated:9`. **F2 (région de
  signature) INFIRMÉ** : le scope `eu-west-2` est accepté par l'OAPI cloudgouv → aucun
  changement. Fixes §1 confirmés en live (loadbalancer N/A, iam_no_root not-evaluated).
- ✅ **§2 câblage live Outscale** :
  - `subnet` (`ReadSubnets` → `MapPublicIpOnLaunch`) : validé live (pass, subnets réels).
  - `load_balancer` (`ReadLoadBalancers`) + nouveau transform moteur **`snake_keys`**
    (projection récursive PascalCase→snake_case des structures imbriquées). Validé
    **doublement** : test unitaire déterministe + **bout-en-bout live** (LBU non conforme
    créé → 2 FAIL `ssl_listeners`/`logging` → **détruit**, baseline retrouvée).
  - **profondeur EIM + `iam_no_root`** (F3/F4) : double collecte taggée `scope: account`
    (ReadAccessKeys appelant) et `scope: eim` (`for_each` ReadUsers → ReadAccessKeys par
    user, avec `owner_user`) ; dérivation Rego de `root_owned` par **différence
    d'ensembles** (clé compte ∉ ensemble EIM), règle agnostique préservée ; gate
    `requiredAttr` généralisé en **any-of** (`root_owned` OU `scope`). Corrige aussi F4
    (clés EIM auditées pour l'expiration). Validé live : **3 clés root** détectées, 2 clés
    EIM collectées (elles ont une expiration → correctement non flaggées). Tests Rego
    (dérivation scope) + non-régression Scaleway.
  - **`compute_image`** (STO-2, F6) : `ReadImages` filtré sur le compte (`for_each
    ReadAccounts` → `Filters.AccountIds=self`, impératif contre la tempête de faux
    positifs du catalogue public) ; `public = PermissionsToLaunch.GlobalPermission`.
    Validé live **avec un vrai constat** : 4 OMIs self collectées (pas le catalogue),
    dont **1 réellement publique** → FAIL. Aucun provisioning nécessaire.
  - **OKS / Kubernetes managé** ×4 : API DISTINCTE de l'OAPI (host `api.{region}.oks.
    outscale.com`, `/api/v2`, auth **bi-en-têtes** AccessKey/SecretKey) → **collecteur Go
    dédié** `internal/oks` (le moteur YAML, SigV4 + host unique, ne peut pas l'exprimer),
    branché comme `objectstorage` via `collectkit`. Mapping ancré sur des **clusters réels**
    (`GET /clusters/all`) : `admin_whitelist`, `cp_multi_az`, `disable_api_termination`,
    `auto_maintenances.minor_upgrade_maintenance.enabled`, `version`.
    **Validé live** (eu-west-2, 3 clusters) : les 4 contrôles s'évaluent et remontent de
    vrais écarts, dont un **API Kubernetes ouverte à `0.0.0.0/0`**. Aucun cluster créé.
    **Robustesse** : OKS indisponible (région sans OKS, quota, droits) n'échoue plus le
    scan — avertissement explicite + contrôles « non évalués » (jamais faussement conformes) ;
    baseline cloudgouv inchangée.
- ⚠️ **OKS n'existe pas partout** : indisponible sur cloudgouv-eu-west-1 (quota non activé) ;
  disponible en eu-west-2. Le compte cloudgouv « souverain » et le compte OKS sont DISTINCTS.
- ✅ **F6** (snapshots de tiers → filtre self) et **F8** (règle d'accès API exigeant un
  certificat client → plus de faux positif) corrigés et validés sur les deux comptes.
- 🔶 **F5** (policies EIM *inline*) : contrat officiel obtenu (`ReadUserPolicies`→`PolicyNames`,
  `ReadUserPolicy`→`PolicyDocument`), mais chaîne à **3 niveaux** (hors `for_each` mono-niveau)
  et **0 occurrence** sur les deux comptes (le modèle Outscale utilise des policies *liées*,
  déjà collectées ; 0 policy de scope OWS). Laissé ouvert plutôt qu'implémenté à l'aveugle.

### Ce que « tout couvrir » veut dire, mesuré (pas supposé)

`pepin scsl` donne la couverture réelle : **30/82 exigences cloud**, **Outscale 28/82**
(le meilleur des trois). Par famille : NET 7/9, IAM 6/16, STO 5/8, CHF 3/5, GVN 3/12,
K8S 3/12, CMP 2/9, **LOG 1/7**.

Le reste n'est PAS « des collecteurs qui manquent » : les exigences non couvertes sont
majoritairement **non observables dans un scan de configuration de tenant** —
organisationnelles/contractuelles (revues de droits, séparation des tâches, identités
nominatives, notification client, effacement en fin de contrat), **non exposées par l'API**
(politique de mot de passe, verrouillage de compte, IMDS, flow logs), ou relevant d'outils
d'un autre type (IDS/WAF, corrélation SIEM, classification de données, JIT/CIEM).
→ Ces exigences doivent aller dans **`gaps.md` (écarts structurels assumés et sourcés)**,
pas devenir des contrôles inventés. C'est précisément l'atout d'opposabilité du projet.
- **Baseline Outscale cloudgouv à ce stade** : `fail:40 · pass:16 · N/A:4 · not-evaluated:7`
  (départ : 36/15/4/10). +5 pass/fail nets = subnet, LB (câblés), iam_no_root (3),
  compute_image (1), moins les not-evaluated résorbés.
- ✅ **§0 pagination OAPI (F1)** : le moteur ne suivait aucune pagination → troncature
  silencieuse au-delà d'une page. Ajout de deux styles injectant les paramètres dans le
  **body** du POST (contrat OAPI) : `token-body` (NextPageToken) et `offset-body`
  (FirstItem + ResultsPerPage). Déclaré `size: 1000` sur 11 listes. Confirmé en réel que
  l'OAPI pagine sur demande (ResultsPerPage=50 → 50 + token) ; **validé live** en forçant
  de petites pages : subnet **7/7** (token, 3 pages), iam_policy **3/3** (offset, 2 pages) ;
  baseline inchangée en taille de production. Unit tests des deux styles.
- ⏳ **§3** : nouvelles familles (comptes réels, teardown).

## North star

Faire de **pepin** un CSPM multi-cloud **souverain** (Exoscale / Outscale / Scaleway /
OVH) **crédible, opposable et publiable**, à parité avec son frère **pavois** :
même socle `scankit`, mêmes portes qualité, site bilingue généré par le binaire,
release signée. La barre : **le meilleur fond sourcé du marché souverain**, et une
posture *honnête* (jamais de faux vert, chaque N/A justifié et opposable).

---

## 1. État des lieux (les 5 audits)

### Notes de couverture réelle

| Provider  | Note | Ce qui est câblé | Le vrai problème |
|-----------|:----:|------------------|------------------|
| Scaleway  | **5,5/10** | justesse 9/10 | ~⅓ de la couverture affichée est du **faux vert en live** ; PaaS/IAM-settings inexploités |
| Exoscale  | **6/10** | natif, honnête | 2 faux PASS, 1 N/A périmé, **DBaaS entièrement absent**, souveraineté à trancher |
| Outscale  | **6,5/10** | 0 champ faux | **pagination tronquée à 100**, signature mono-région, périmètre IAM superficiel |
| Corpus (complétude) | ~40-45 % d'un CSPM de référence, **~57 % du SCSL** vérifiable-par-config | fort IAM/réseau/stockage/souveraineté | faible PaaS / journalisation-org / cycle de vie crypto |

### Le fil rouge systémique (commun aux 3 providers)

**Le contrat `etat:` est par *type*, pas par *(type × mode)*.** Il ne sait pas dire
« vérifié en Terraform, absent en live ». Résultat : des contrôles déclarés
`fournisseurs: [x]` s'affichent **verts en scan live sans jamais être évalués**,
parce que le verrou d'opposabilité (`providerVerified`, `cmd/scan.go:420`) répond
`true` dès que le type est `verifie`, même quand l'**attribut décisif** n'est pas
collecté. C'est le prolongement exact du trou « verified-par-type-pas-par-attribut »
déjà attaqué. **Correction de fond** : granularité du verrou à l'**attribut** (déjà
amorcée via `attrCollected`), et discipline « un type non collecté en live ⇒ pas
affiché couvert en live ».

Second motif : **chaque provider a une famille produit entière sous-exploitée**
(Scaleway : PaaS + IAM security-settings ; Exoscale : DBaaS + org-policy ;
Outscale : LBU + OKS + profondeur EIM).

---

## 2. Axe 1 — Produit & publication (parité pavois)

Rendre pepin publiable. Ordre gravé (détail dans le plan) :

- **Track S — publier `scankit` v0.1.0** (fait EN PREMIER) : LICENSE Apache-2.0,
  README, tests des 4 packages, CI durcie, tag → **basculer pepin du `replace`
  local vers le tag**.
- **P0 — Fondation dépôt** : `git init`, LICENSE Apache-2.0 + NOTICE, **hygiène de
  publication** (assainir/gitignorer `*.tfstate*`, IPs, IDs de compte), docs repo EN
  (README badgé, ARCHITECTURE, CONTRIBUTING, SECURITY, CoC, CHANGELOG, **ROADMAP.md
  dérivé de ce fichier**).
- **P1 — Le binaire émet ses données de site** (anti-dérive) : `pepin docs cli`,
  `pepin referentiel export`, `pepin oscal`.
- **P2 — Site Astro bilingue FR/EN** (`site/`), calqué sur confkit : collections de
  contenu, pages `[lang]/`, générateur Python, garde-fous anti-dérive.
- **P3 — Workflows durcis** (skill `github-secure-pipeline`) : harden-runner +
  `permissions:{}` + SHA-pin partout, dependency-review, deps/Trivy, plumber, release.
- **P4/P5 — Release** goreleaser + SLSA + cosign ; tâches `mise` site/génération.

---

## 3. Axe 2 — Fond : complétude & justesse (les audits)

### §0 — Bugs moteur transverses (priorité haute, corrigent des faux résultats)

Découverts par l'audit Outscale, ils affectent la **correction** du scan, pas juste
la couverture :

1. **Pagination OAPI absente et inexprimable** (`F1`, critique). `ReadPolicies`/
   `ReadUsers` ont un défaut documenté à **100** → la collecte `iam_policy` Outscale
   est **tronquée silencieusement**, en violation de la doctrine « jamais de
   troncature muette » (`engine.go:163-199`). Le moteur ne sait pas injecter le
   `NextPageToken` **dans le body POST** ni gérer l'offset `FirstItem`. → Ajouter
   deux styles : `token-body` et `offset-body`.
2. ~~**Région de signature figée `eu-west-2`** (`F2`)~~ **INFIRMÉ par scan réel**
   (2026-07-19, compte `souverain`) : un scan live de **`cloudgouv-eu-west-1`** avec
   le code actuel (scope de signature `eu-west-2`) **réussit** et collecte les
   ressources — l'OAPI cloudgouv n'exige pas l'égalité région-de-signature/région-cible.
   L'hypothèse « quasi certaine » de l'audit était fausse. **Aucun changement.** (Cas
   d'école du garde-fou « pas de correctif sans scan réel ».)
3. **`IpProtocol` numérique** (`F7`, moyen, Outscale) : `"6"`/`"17"`/`"1"` non mappés
   → SSH ouvert via protocole numérique = faux négatif (contournement CSPM classique).
   → étendre la table du transform protocol.
4. **`volume_in_use` casse Scaleway** (`lib.rego:87`) : le wire Scaleway est **`in_use`**
   (underscore), non reconnu → à ajouter avant tout câblage block Scaleway.

### §1 — Corrections d'opposabilité SANS scan ni changement de détection (livrables immédiats)

Pur « fond honnête », zéro risque :

| # | Correction | Fichier |
|---|-----------|---------|
| 1 | **Souveraineté Exoscale** `controle_capitalistique: a_verifier → extra_ue` (chaîne Akenes → A1 Digital → Telekom Austria → **América Móvil MX 60,8 %**) + sourcer ; le deny CLD-GVN-4 gagne le bon message | `providers/exoscale.yaml:9-15` |
| 2 | **Raison N/A KMS Exoscale périmée** : réécrire (SSE-SOS existe et est activé par défaut depuis 2025 ; la *conclusion* N/A tient, pas la *raison*) | `providers/exoscale.yaml:399-403` |
| 3 | **Retirer les faux-pass structurels** de `fournisseurs:` tant que l'attribut décisif n'est pas collecté : `iam_no_root_access_key` (outscale + scaleway : `root_owned` jamais dérivé) | `referentiel/controles.yaml:122` |
| 4 | **`loadbalancer_http_redirect_to_https` → `non_applicable` Outscale** (mécanisme inexistant : ni Listener ni ListenerRule n'ont de redirection) | `controles.yaml:669` + `providers/outscale.yaml` |
| 5 | **Requalifier `subnet`/`compute_image` Outscale** de `verifie` (TF-only trompeur) → état honnête tant que ReadSubnets/ReadImages ne sont pas câblés en live | `providers/outscale.yaml:431-448` |
| 6 | **`blockstorage_snapshot_not_public` Scaleway → `absent`** (aucun mécanisme de partage public de snapshot block) | `providers/scaleway.yaml` |
| 7 | Purges cosmétiques : alias de zones fantômes (`lib.rego` Exoscale ×5, Outscale `cn-southeast-1`), commentaires périmés (« cible 2025 » Scaleway) | `lib.rego`, `providers/*.yaml` |

> **Nuance flow logs** (corrige ma synthèse précédente) : `ReadFlowLogs` **n'existe
> pas** dans l'OAPI Outscale → absence de flow logs = **N/A légitime chez Outscale**,
> pas un trou pepin. Reste un écart structurel réel chez les autres → §4.

### §2 — Câblage de collecte live (corrige les faux verts ; souvent l'endpoint est déjà appelé)

**Scaleway** — top 5 (tout YAML pur sauf mention) :
1. `security_group` objet en live (`inbound_default_policy`) : **8 lignes**, l'endpoint
   est déjà appelé par le `for_each` ; défaut constructeur = `accept` (non conforme).
2. RDB live (`/rdb/v1/.../instances` + `for_each .../acls`) : corrige 3 faux pass,
   attrape l'ACL par défaut `0.0.0.0/0` invisible au plan TF.
3. IAM live : policies + rules + **security-settings** (`max_api_key_expiration_duration`
   ⇒ branche `iam_apiaccesspolicy_max_key_expiration` quasi gratuitement).
4. VPC v2 private-networks (SCW-Q1 résolu) : `network_documented` en live.
5. Kapsule : clusters + `/acls` (`kubernetes_cluster_not_publicly_accessible`, `auto_upgrade`).
   *Hors top 5* : `root_owned` via `User.type=owner` ; `user_data` (exige **code Go** :
   la réponse n'est pas du JSON).

**Exoscale** — top 5 :
1. **DBaaS** (trou majeur) : `/dbaas-service` + détail par moteur (`ip-filter`,
   `backup-schedule`, `termination-protection`) → mapping **quasi direct** sur ce que
   les règles `database_*` lisent déjà ; ajouter exoscale aux 3 contrôles DB.
2. Corriger `audit_enabled` : mapper `audit.enabled` (bool), pas la seule présence
   d'endpoint (faux PASS CLD-LOG-1).
3. Chiffrement **observé** : remplacer `const encrypted:true` par le champ réel
   `encrypted` du volume + `disk-encrypted` de l'instance.
4. **Labels en live** partout (instance/volume/SKS/rôle) → `required_tags` évalué en
   live comme en TF.
5. IAM Organization Policy (`default-service-strategy` org) ; check « SSE-SOS activé »
   via `GetBucketEncryption`.

**Outscale** — top 5 (après §0.1 pagination) :
1. **Pagination** (bloquant, cf. §0).
2. `ReadLoadBalancers` (type `load_balancer`, 0 obstacle) : `ssl_listeners`, `logging`.
3. **OKS `GET /clusters/all`** (spec publique) : `admin_whitelist`, `cp_multi_az`,
   `disable_api_termination`, `auto_maintenances` → 4 contrôles s'activent d'un coup
   (auth bi-en-têtes à supporter).
4. Profondeur EIM : ReadUsers → ReadAccessKeys par user + politiques **inline/liées**.
5. ReadSubnets + ReadImages en live (**`Filters.AccountIds=self` impératif**, sinon
   tempête de faux positifs sur les OMIs publiques) + `auth.region` substitué.

### §3 — Nouvelles familles / règles (extension de couverture, chacune sourcée + scan réel)

- **PaaS exposé par défaut** : serverless `privacy`/`http_option` (Scaleway),
  registry `is_public` (Scaleway), namespace d'images public.
- **Cycle de vie cryptographique** : Key Manager `rotation_policy` (Scaleway),
  rotation/expiration des clés, expiration de certificats (`ReadServerCertificates`
  Outscale).
- **Endpoints privés** : RDB endpoint public vs private-network, Redis `tls_enabled`.
- **TLS des LB** : `ssl_compatibility_level` (Scaleway ; N/A chez Exoscale L4 et
  Outscale sans politique TLS — cf. §4).
- **Journalisation/alerting d'organisation** : Audit Trail + Cockpit (Scaleway),
  `/event` + audit-trail (Exoscale), `ReadApiLogs` (Outscale).

### §4 — Écarts structurels à publier (`gaps.md`, l'atout d'opposabilité)

À documenter comme **écarts assumés et sourcés**, pas comme checks manquants :

| Écart | Statut par provider |
|-------|--------------------|
| Flow logs | **N/A Outscale** (mécanisme inexistant) ; absent Exoscale/Scaleway |
| IMDS hardening | absent des 3 |
| Password policy compte | Scaleway (security-settings) **existe** ; Outscale via ApiAccessPolicy ; à câbler |
| TLS ciphers par frontend LB | N/A Exoscale (L4), N/A Outscale ; ancrable Scaleway |
| S3 Block Public Access / GetBucketEncryption | lisibilité variable selon provider |
| Chiffrement volume observable | N/A Outscale (pas de champ) ; observable Exoscale (`encrypted`/`disk-encrypted`) ; N/A Scaleway |

---

## 4. Séquencement & dépendances entre axes

Les deux axes **convergent sur P1/P2** : chaque N/A encodé et chaque règle ajoutée
**alimente le site et l'OSCAL**. Donc **le fond doit être honnête AVANT de le publier**.

```
Axe 2 §1 (corrections sans scan)      ──┐
Axe 2 §0 (bugs moteur)                  │  fond honnête
Axe 2 §2 (câblage live, avec scans)   ──┘        │
                                                 ▼
Track S (scankit v0.1.0) → P0 → P1 → P2 (site consomme le fond) → P3 → P4/P5
                                                 ▲
Axe 2 §3/§4 (nouvelles règles + gaps.md) ────────┘  (en continu, alimente le site)
```

**Démarrage recommandé** : Axe 2 **§1** (immédiat, sans scan) → **§0.2** (région de
signature, débloque le scan de la région SecNumCloud) → **§2** provider par provider
(chaque câblage validé par un scan réel, cf. garde-fou). En parallèle, **Track S**
peut avancer (indépendant du fond).

---

## 5. Décisions

**Prises**
- ✅ Parité complète *avant* première publication ; site bilingue FR/EN ; docs repo EN.
- ✅ scankit v0.1.0 puis bascule pepin hors `replace`.
- ✅ **Renommage SecNumCloud** : `name → « ANSSI SecNumCloud 3.2 »` + description
  recadrée (vue pour le **client** cloud, sous-ensemble technique, pas une
  qualification du prestataire) ; `code`/slug intacts. *Appliqué, tests verts.*
- ✅ **Domaine du site** : **pepin.io** (→ `astro.config.mjs` `site: 'https://pepin.io'`,
  canonicals, `sitemap.xml`, `llms.txt`, `robots.txt`, badges).

**En attente**
- ⏳ **Scans réels** (comptes + teardown) pour débloquer §0 (pagination, région de
  signature) puis §2/§3.

---

## 6. Garde-fous projet (non négociables)

- **Aucun changement de règle/collecte sans scan réel** validant (comptes réels
  pepin ; *toute ressource provisionnée est détruite en fin de run* — CLAUDE.md §1.1).
  Privilégier `scan --tf` (terraform-plan, sans provisioning) quand c'est possible.
- **Pas de secret en dépôt** ; pas de `git push`/tag/PR sans accord explicite.
- **Aucune attribution Claude** dans les commits ; Conventional Commits.
- Le contrat `etat` par type est un **défaut structurel connu** : viser la
  granularité par attribut (`attrCollected`), ne pas ajouter de faux vert en
  attendant.
