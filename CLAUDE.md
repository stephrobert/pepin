# CLAUDE.md — règles de travail assisté par IA sur Pépin

Pépin est un **CSPM multi-cloud souverain** (Go). Il évalue la **posture** d'un
cloud (OVH, Scaleway, Exoscale, Outscale…) contre un **référentiel commun** de
contrôles (`referentiel/`), ancré sur **SCSL** (module posture cloud), **SecNumCloud**,
**CIS** et **ISO**. « Pépin trouve les pépins de votre cloud. »

> Né de la généralisation d'**osc-policy** (Outscale) et de la lignée **pitstop**
> (SCSL-R) / **plumber**. Le référentiel commun est la source de vérité des
> contrôles ; les règles Rego par provider en sont l'implémentation.

## 0. Règle d'architecture FONDAMENTALE — un seul jeu de règles commun

**Toutes les règles de posture sont COMMUNES à tous les providers.** Elles vivent
dans **`internal/commonrules/`** et s'évaluent sur le **modèle normalisé commun**
(`input.resources[]`, types et attributs agnostiques). **Aucune règle n'est propre
à un provider.** Ce qui change d'un cloud à l'autre, c'est uniquement la **SOURCE**
(le collecteur) : tout est **normalisé AVANT** de passer dans les règles.

- **Providers = COLLECTEURS Go compilés.** Un cloud = un package `providers/<nom>/`
  qui implémente `provider.Provider` (interface = `Name`/`Description`/`Collect`
  [+ `TerraformMapper`]). Son seul rôle : projeter sa source (API live via SDK
  natif, plan Terraform, S3) vers le **modèle normalisé commun**. Il ne porte
  **aucune** règle Rego. Ajout d'un provider ⇒ recompilation. Skill `nouveau-provider`.
- **Règles = Rego commun.** `internal/commonrules/rules/*.rego` (embarqué, chargé à
  chaque scan) + règles externes `--policy-dir` (hot-load), fusionnées et
  interrogées via `data.pepin.rules.deny`. `labels.provider` est **tiré de la
  ressource** (`provider_of(r)`), jamais codé en dur. Une règle ne se déclenche que
  si des ressources du type visé existent ; un provider qui n'a pas ce type ne la
  déclenche pas (pas de faux positif). Skill `nouvelle-regle`.

> Corollaire : pour couvrir un nouveau check, on **normalise** la donnée dans le
> collecteur/mapper vers le modèle commun, puis on écrit **une** règle commune —
> jamais une règle par provider.

## 1. Build, test, audit — toujours via `mise`

Le projet est piloté par **mise** (`mise.toml`, plus de Makefile) : `mise run <tâche>`
(`mise tasks ls` pour la liste ; mise fournit aussi les outils — opa, terraform).

```bash
mise run build   # compile (version injectée)
mise run test    # go test ./... -race + tests Rego (opa)
mise run audit   # vet + lint (golangci-lint) + sec (gosec) + vuln (govulncheck) + osv
```

Ne pas committer si `mise run test` ou `mise run audit` échoue.

### 1.1 Scans réels & provisionnement (comptes Outscale + Scaleway disponibles)

Des comptes **Outscale** et **Scaleway** réels sont disponibles pour valider une
règle par un scan live (`--live`), condition non négociable avant d'ACTIVER un
contrôle (`fournisseurs:` non vide, contrat `verifie`).

**RÈGLE ABSOLUE — nettoyage : toute ressource provisionnée pour un test DOIT être
détruite à la fin** (`terraform destroy`, ou suppression via l'API/console). Ne
jamais laisser une ressource de test vivre : coût, surface d'exposition, dérive.
Tenir la liste de ce qui est créé et confirmer la destruction avant de conclure.

**Préférer le NON-provisionnement.** pepin sait auditer un **plan Terraform**
(`scan --tf plan.json`, chemin `tfmap`/`tfparse`) : pour valider une règle et le
mapping d'un nouveau type, écrire (ou réutiliser depuis GitHub) un `.tf` du/des
provider(s) souverain(s), générer `terraform plan -out` → `plan.json`, et scanner
CE plan — aucune ressource cloud créée. Le scan live ne sert qu'à confirmer le
**contrat d'API** (champs réels) quand le plan ne suffit pas ; il est alors suivi
d'un `destroy` immédiat.

### 1.2 Bilinguisme : docs, CLI et référentiel

Comme pavois, les docs du dépôt sont **bilingues FR/EN**. **L'anglais est la langue
PRIMAIRE** : `README.md`, `SECURITY.md`, `CONTRIBUTING.md` sont en anglais ; leur
contrepartie française est `*.fr.md`, reliée par un sélecteur de langue en tête
(`🇬🇧 English · [🇫🇷 Français](*.fr.md)`). Tenir les deux versions synchronisées à
chaque changement.

**La CLI et le RÉFÉRENTIEL sont eux aussi bilingues** (issue #37) : la langue est
DÉTECTÉE (`--lang` → `PEPIN_LANG` → `LC_ALL` → `LANG` → repli `en`, cf.
`internal/i18n`), et elle vaut pour tout ce que l'outil imprime (rapport, verdict,
aide, erreurs) comme pour les formats parsables (`json`, `sarif`, `oscal`,
`assessment`). Le **français reste la langue de RÉFÉRENCE du contenu normatif** :
c'est lui qui fait foi, l'anglais en est la traduction maintenue en parallèle.

Concrètement, tout texte destiné à l'utilisateur s'écrit DEUX fois, côte à côte :

- **Go** : `i18n.T(fr, en)` au point de rendu ; les chaînes d'aide de cobra, figées
  à l'init, sont réécrites par `cmd.localize()` après résolution de la langue.
- **Rego** : `message`/`remediation` en français, `labels.message_en` et
  `labels.remediation_en` en anglais (`finding.Finding.Labels` est extensible :
  **on ne modifie pas scankit**). Le scan consomme ces labels puis les RETIRE, car
  ce sont un transport, pas une donnée du rapport.
- **Référentiel** : `titre_en`, `description_en`, `remediation_en` à côté de leurs
  homologues français dans `referentiel/controles.yaml`.
- **Contrats providers** : `reason_en` à côté de `reason` (justifications de
  non-applicabilité, lues par un auditeur dans l'OSCAL).

Trois portes refusent une traduction manquante, et elles sont dans `mise run validate`
et `mise run test` : `TestEveryControlIsBilingual`, `TestEveryFindingCarriesRemediation`
et `TestEveryContractJustificationIsBilingual`. Une quatrième, `cmd/lang_test.go`,
exige qu'une sortie `LANG=en` ne porte aucun mot accenté qui ne vienne pas de
l'inventaire scanné, et que la sortie française n'ait pas bougé d'un octet.

Le CODE et les COMMENTAIRES restent en français ; les COMMITS restent en anglais
(Conventional Commits).

## 2. Ancrage sur le contrat de l'API — règle d'or (skill `ancrage-contrat`)

**Ne jamais inventer le modèle de ressources d'un provider.** Le modèle évalué
DOIT refléter le **contrat natif** du SDK/API (champs, types, tags JSON). Un champ
non vérifié dans le SDK ne s'emploie pas ; un champ **dérivé** (calculé par le
collecteur) est explicitement marqué « DÉRIVÉ » avec sa formule. Les fixtures
`examples/*.json` sont des réponses API réalistes, jamais fantaisistes. Citer la
source du contrat dans l'en-tête de chaque règle.

## 3. Règles Rego (skill `nouvelle-regle`)

- **Emplacement unique : `internal/commonrules/rules/*.rego`** (jamais dans
  `providers/`). Une règle = un contrôle commun à tous les providers (§0).
- `package pepin.rules` ; `import rego.v1` ; `deny contains f if { … }`.
- `f` suit le modèle partagé **`scankit/finding.Finding`** (cf. §9) : `code`,
  `severity` (`critical|high|medium|low`), `subject` (ressource fautive),
  `message` (FR actionnable), `remediation` (FR), et
  `labels: {"provider": provider_of(r), "category": "security|compliance",
  "message_en": …, "remediation_en": …}` : **`provider` tiré de la ressource via le
  helper `provider_of`, jamais en dur**, et les deux labels `_en` OBLIGATOIRES
  (§1.2 : la traduction voyage dans les labels, scankit n'est pas modifié).
- **`code` = identifiant de check agnostique, COMMUN à tous les providers**
  (ex. `network_securitygroup_allow_ingress_from_internet_to_tcp_port_22`,
  `objectstorage_bucket_public_access`). Convention `<service>_<resource>_<check>`
  avec préfixe service **neutre** (`network_`, `compute_`, `objectstorage_`,
  `blockstorage_`, `iam_`, `kubernetes_`, `loadbalancer_`, `governance_`). Le
  contenu (`message`/`remediation`/fixtures) reste, lui, propre au provider
  (vocabulaire natif, ancré SDK §2). **Pas d'id de règle propre au provider** (plus
  de `osc-xxx`/`scaleway-xxx`).
- **Le `code` doit exister dans `referentiel/controles.yaml`**, qui le mappe aux
  frameworks et à l'**index SCSL** : exigence `CLD-*` (domaine **CLD**, posture
  cloud) de l'API statique du framework `framework-scsl/api/v1/exigences.json` —
  l'ID canonique `SCSL-CLD-*` est lu **dépréfixé** en `CLD-*`. **L'index SCSL est
  GELÉ** : on **mappe sur une exigence `CLD-*` existante**, on n'en **crée/n'en
  invente jamais**. Si
  aucune exigence gelée ne couvre le besoin, le contrôle reste au **catalogue**
  (`statut: a_trier`) — hors périmètre tant que SCSL ne l'a pas figé. `mise run validate`
  refuse tout `scsl` absent de l'index gelé.
- Accès défensif (`object.get`). Toute règle ⇒ test `*_test.rego` (cas ✓ et ✗),
  exécuté par `mise run test-rego` (`opa test`).

## 4. Conventions Go (barre minimale)

- `gofmt -w` sur chaque fichier (hook automatique).
- Erreurs enveloppées avec contexte : `fmt.Errorf("lecture de %s : %w", path, err)`.
- Ne pas ignorer une erreur via `_` sauf si documenté jetable (`_, _ = fmt.Fprintln`).
- Tout symbole exporté reçoit un commentaire de doc commençant par son nom.
- Pas d'abstraction spéculative : on ajoute quand un besoin concret apparaît.

## 5. Sécurité du code (l'outil audite la sécurité — il doit être exemplaire)

- Le mode `scan` lit un export en **lecture seule** ; pas d'exécution de l'entrée.
- `mise run sec` (gosec) et `mise run vuln` (govulncheck) doivent passer ; traiter un
  finding, ne pas le supprimer sans justification écrite (`//nolint:` motivé).
- Entrées non fiables (export d'un tiers) : parser défensivement, jamais de panique.

## 6. Codes de sortie (portes de CI)

`0` conforme · `1` non-conformité (≥ 1 finding critical/high) · `2` erreur
technique · `3` **le scan n'établit pas la conformité** : rien de mesuré (inventaire
vide), collecte incomplète (une unité n'a pas pu être lue, cf. `model.Collection`),
ou écarts medium/low avec `--strict` · `4` tout écart critical/high est couvert par
une dérogation valide. `3` et `4` ne sont PAS des `0` : ni le vide, ni le partiel, ni
la dérogation ne valent conformité. **Il n'y a délibérément pas de cinquième code**
pour l'incomplétude : elle ne peut jamais primer sur `1` (un écart observé reste un
écart), donc elle n'occuperait que la place que `3` occupe déjà. Ce qui distingue les
cas est LISIBLE ailleurs — relevé de capacités, motif de chaque `not-evaluated`, clé
`collection` de `--format json`.
Ne pas changer cette sémantique sans mettre à jour README, README.fr,
`docs/reference/exit-codes.md` (généré, + `.fr.md`) et les tests.

## 7. Commits & branches

- Conventional Commits : `<type>(<scope>): <desc>` (scopes `engine`, `provider`,
  `policies`, `referentiel`, `cmd`, `scoring`…). Impératif, < 72 caractères.
- Branches `feat/<slug>`, `fix/<slug>`. **Jamais de push/PR sans demande
  explicite** ; pas de `--no-verify`/`--force`. Proposer le message, attendre l'accord.

## 8. Périmètre

Pas de feature hors tâche ; pas de « nettoyage » du code voisin dans le même
changement. Si une « petite » modification touche plus de trois fichiers,
s'arrêter et demander.

Module Go : `github.com/stephrobert/pepin`.

## 9. Découplage en modules — `scankit`

Le **moteur OPA, le modèle `Finding`, le rendu (terminal/SARIF), le scoring et
l'assessment (statuts/bundle/OSCAL)** viennent du module partagé
**`github.com/stephrobert/scankit`** (publié, consommé **en ligne** — épinglé par
version dans `go.mod`, **plus de `replace` local**), commun à Pépin et **pitstop**.

- Reste **dans Pépin** : les règles `.rego` (passées à `scankit/engine.Evaluate`),
  les providers/collecteurs, le référentiel, la marque (bandeau PEPIN), le verdict
  Conforme.
- Vient de **scankit** : `engine`, `finding.Finding` (cœur + `labels`), `report`
  (`Banner`/`Terminal`/`SARIF`, marque via `report.Options`), `scoring`.
- Toute évolution moteur/rendu/scoring se fait **dans scankit** (profite aux deux) ;
  pas de moteur/rendu local à Pépin. Le rendu terminal est **identique à pitstop**.

## 10. Procédure canonique — ajouter/couvrir un contrôle (GRAVÉE)

Les artefacts pilotent tout : `catalogue.yaml` (QUOI), `contrats/<provider>.yaml`
(COMMENT, vérifié), `controles.yaml` (contrôle actif + SCSL gelé), `commonrules`
(règles communes). Pour couvrir un contrôle, suivre **exactement** ces étapes :

1. **Exigence SCSL d'abord (gel).** Vérifier qu'une exigence `CLD-*` **existe déjà**
   dans l'index gelé (`framework-scsl`, module SCSL-C). **Ne jamais en créer.** Si
   aucune ne couvre le besoin ⇒ laisser le contrôle au catalogue (`a_trier`), stop.
2. **Contrat par provider.** Pour CHAQUE provider visé, vérifier le champ natif dans
   son SDK et le consigner `etat: verifie` + mapping dans `referentiel/contrats/<provider>.yaml`.
   Jamais supposer (« absent » se prouve en lisant le SDK).
3. **Référentiel.** Ajouter le contrôle à `referentiel/controles.yaml` : `code`
   agnostique, `severite`, `scsl: [CLD-*]` (gelé), `frameworks`, `fournisseurs`,
   et les trois champs BILINGUES `titre`/`titre_en`, `description`/`description_en`,
   `remediation`/`remediation_en` (§1.2 ; `mise run validate` refuse une absence).
4. **Collecteur.** Normaliser le champ natif → attribut commun dans
   `providers/<provider>/` (collecteur live ET mapper Terraform). Le squelette se
   génère : `python3 scripts/gen-collector.py <provider>`.
5. **Règle commune.** UNE règle dans `internal/commonrules/rules/` (`package
   pepin.rules`, `labels.provider: provider_of(r)`, plus `labels.message_en` et
   `labels.remediation_en`, cf. §1.2) + test `*_test.rego` (✓ et ✗).
   Passer l'entrée du catalogue en `statut: implemente`.
6. **Contrat de véracité.** Un test Rego ne prouve qu'un maillon. Écrire les
   scénarios de bout en bout dans `internal/veracity/testdata/scenarios/` : un
   fichier par chemin contrôle × fournisseur × source, prouvant les verdicts que
   ce chemin peut atteindre (`fail`/`pass`/`not-evaluated` s'il sait conclure,
   `not-applicable` si le contrat le déclare tel). Un scénario `live` dont le type
   est produit par la spec `collecte` entre par `api:` — des réponses canned
   servies au collecteur RÉEL —, jamais par `inventory:` : c'est le collecteur qui
   a laissé passer l'incident fondateur, pas la règle. Ce qui n'est pas prouvé se
   CONSIGNE (`mise run veracity-update`), jamais ne se masque ; le registre est une
   porte dans les deux sens, et un contrôle ajouté sans ses scénarios casse la CI.
7. **Valider.** `mise run validate` (codes↔règles↔SCSL gelé↔catalogue) **puis**
   `mise run test` + `mise run audit` au vert. Aucun commit si l'un échoue (§1, §7).
8. **Documenter.** `mise run gen-docs`, et committer ce qu'il régénère : la matrice
   de couverture et les sorties capturées sont GÉNÉRÉES depuis le binaire et le
   référentiel, et `TestGeneratedDocsAreUpToDate` casse la CI si elles dérivent.
   Puis relire ce que le changement rend FAUX ailleurs — `docs/known-limitations.md`
   si un angle mort se comble, la page du provider concerné, `docs/coverage.md` —
   dans les DEUX langues. Un changement qui déplace un verdict sur un tenant
   inchangé, une surface analysable ou un code de sortie a sa ligne au CHANGELOG.

   La documentation n'est pas une étape de suivi : une page qui décrit un produit
   disparu est pire qu'une page absente, parce qu'elle inspire confiance. La
   question qui tranche : *quelqu'un qui lit la documentation sans lire le code
   serait-il induit en erreur par ce changement ?*

Invariants vérifiés par `mise run validate` : tout contrôle actif a une règle ; tout
code émis est catalogué ; tout `scsl` existe dans l'index **gelé** ; le catalogue
est cohérent. **Un nouveau provider = 1 contrat + 1 collecteur, zéro règle.**
