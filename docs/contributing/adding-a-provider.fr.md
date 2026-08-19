> [🇬🇧 English](adding-a-provider.md) · 🇫🇷 Français

# Ajouter un fournisseur, de bout en bout

Un fournisseur, dans Pépin, est **une source, pas un jeu de règles**. Toutes les règles
de posture sont communes à tous les clouds et s'évaluent sur le modèle normalisé ; ce
qu'un fournisseur apporte, c'est la projection de son API et de ses ressources Terraform
vers ce modèle. Ajouter un cloud revient donc à écrire **un descripteur**,
`providers/<nom>.yaml`, et **zéro règle**.

Ce n'est pas un slogan, c'est le critère d'acceptation : si votre changement ajoute un
fichier `.rego`, c'est la normalisation qui a manqué. Voir
[l'architecture](../project/architecture.fr.md) pour la raison.

Voici ce que le binaire connaît des fournisseurs enregistrés aujourd'hui :

<!-- pepin:gen provider-list -->
```text

// pépin  providers enregistrés
  exoscale  Exoscale (CH) — instances, security groups, block storage, SKS, SOS
  kubernetes  Kubernetes (in-cluster) — RBAC, Pod Security Standards, NetworkPolicy
  outscale  Outscale (3DS) — VM, BSU, OOS, EIM, security groups, OKS, LBU
  scaleway  Scaleway — object storage, instances, IAM, security groups
```
<!-- /pepin:gen provider-list -->

## La règle d'or : ne jamais inventer le modèle de ressources

Le modèle que Pépin évalue **doit** refléter le contrat natif du fournisseur : champs
réels, types réels, tags JSON réels, lus dans le SDK ou dans la documentation officielle
de l'API. Un champ que vous n'avez pas vérifié ne s'emploie pas. Un champ que vous
**dérivez** (calculé par le collecteur ou par le mapping) est marqué comme dérivé, avec
sa formule.

La raison n'est pas le purisme. Un CSPM qui lit un champ inventé n'échoue pas bruyamment :
il rend « conforme » en silence, parce que l'attribut n'est tout simplement jamais là.
C'est un faux vert, et un faux vert est invisible par construction.

Deux habitudes rendent la chose praticable :

- Mettre en cache les pages sur lesquelles vous vous appuyez : `mise run fetch-docs`
  stocke la documentation officielle en Markdown sous `references/docs/<fournisseur>/`,
  et une entrée de contrat cite alors un fichier plutôt qu'un souvenir.
- Consigner les questions ouvertes dans
  `references/questions-providers/<fournisseur>.yaml` au lieu de deviner. Une question
  sans réponse vaut `a_verifier`, pas `verifie`.

## Préférer ne rien provisionner

Pépin sait auditer un **plan Terraform** (`terraform show -json`), ce qui suffit à
valider un mapping et une règle **sans créer la moindre ressource** :

```bash
terraform plan -out tfplan && terraform show -json tfplan > plan.json
./pepin scan <fournisseur> --terraform plan.json
```

Le scan live ne sert qu'à confirmer le **contrat d'API** (les champs que l'API rend
réellement) quand le plan ne suffit pas. Et la règle est alors absolue : **toute
ressource provisionnée pour un test doit être détruite** (`terraform destroy`, ou
suppression via l'API ou la console). Tenir la liste de ce que vous créez, et confirmer
la destruction avant de conclure. Le coût, la surface d'exposition et la dérive plaident
dans le même sens.

---

## 1. Identité et portée

```yaml
name: acme                       # l'identifiant que prend la CLI : pepin scan acme
description: "ACME (FR) — instances, security groups, stockage objet"
scope: cloud                     # « cloud » (défaut) | « in-cluster » pour une autre portée
region_key: region               # la clé logique alimentée par --region (Exoscale : « zone »)
```

`scope` n'est pas cosmétique : un fournisseur dont la portée n'est pas le plan de
contrôle cloud (le fournisseur `kubernetes` audite l'intérieur d'un cluster) est tenu
hors de la matrice de parité cloud, aucune des deux portées ne pouvant couvrir l'autre.

## 2. Les faits de souveraineté

Ces champs alimentent la ressource synthétique `governance_provider`, qu'évalue le
contrôle `CLD-GVN-4`. Ce sont des **faits sourcés**, pas des impressions :

```yaml
souverainete:
  eu_etabli: true                 # le siège du fournisseur est-il établi dans l'UE
  juridiction: FR                 # pays du siège
  controle_capitalistique: FR     # juridiction du contrôle ultime : FR | UE | extra_ue | a_verifier
  secnumcloud: non                # qualifie | en_cours | non
  exposition_extraterritoriale: false
  sources: "les URL qui établissent la chaîne capitalistique"
```

Remonter la chaîne capitalistique jusqu'au bout. Le descripteur d'Exoscale en est
l'exemple travaillé : société suisse, contrôle ultime hors UE, chaîne écrite et sourcée.

## 3. Authentification et résolution des identifiants

Le descripteur déclare comment une requête est signée et d'où viennent les identifiants.
Trois sources, dans l'ordre : l'environnement, le fichier de configuration natif du
fournisseur, puis les valeurs par défaut.

```yaml
auth:
  type: header                   # header | sigv4 | exoscale-hmac
  header: X-Auth-Token
  value: "{secret_key}"
credentials:
  env:
    access_key: ACME_ACCESS_KEY
    secret_key: ACME_SECRET_KEY
    region: ACME_REGION
  file: { path: "~/.config/acme/config.yaml", format: scw }   # scw | osc | exoscale
  defaults: { region: fr-par }
```

Lire le fichier de configuration natif compte plus qu'il n'y paraît : quelqu'un qui a
déjà configuré la CLI du fournisseur s'attend à ce que Pépin fonctionne sans rien
réexporter.

**Aucun secret n'entre jamais dans le dépôt.** Les identifiants viennent de
l'environnement ou de ce fichier ; `mise run secrets` (gitleaks) scanne l'historique à la
recherche des motifs de clés souveraines.

## 4. Projeter vers le modèle normalisé

Les règles lisent des types normalisés (`compute_instance`, `security_group`,
`security_group_rule`, `object_storage_bucket`, `managed_database`, `iam_policy`,
`network`, `kubernetes_cluster`…) et des attributs normalisés. Votre travail consiste à
y projeter le vocabulaire natif du fournisseur.

Partir des types existants. Lire
[`internal/commonrules/rules/`](../../internal/commonrules/rules) pour voir quels
attributs les règles lisent réellement, et
[le catalogue des contrôles](../controls/index.fr.md), où chaque page nomme le type lu et
l'attribut dont dépend sa décision.

Si votre fournisseur expose un mécanisme qu'aucun type normalisé ne couvre, c'est un
changement de normalisation (un type, un attribut) suivi d'**une règle commune**, jamais
d'une règle propre au fournisseur.

## 5. Le mapping Terraform, ancré sur le schéma réel

```yaml
mapping_terraform:
  resources:
    - tf_type: acme_security_group_rule
      type: security_group_rule
      id: security_group_id
      map:
        security_group_id: security_group_id
        direction: direction
        protocol: protocol
        port_from: from_port
        port_to: to_port
        cidrs: cidr_blocks
      transforms:
        protocol: lower
        cidrs: list
```

C'est l'endroit où l'invention est attrapée automatiquement.
`TestProviderMappingsMatchSchema` exécute `terraform providers schema -json` dans
`examples/<nom>/terraform/` et vérifie que **chaque attribut source nommé par la spec
existe dans le schéma réel du provider**. Un `tf_type` qui n'existe pas, ou un attribut
qui n'existe pas, fait échouer le test. Quand Terraform ou le schéma sont indisponibles,
le contrôle est ignoré plutôt que déclaré vert : lancez-le donc au moins une fois sur une
machine qui a les deux.

Les blocs imbriqués passent par `items` (le bloc répété est éclaté, `_parent.*` désignant
le conteneur), et `for_each` exprime une liste parente qui alimente un appel par élément.

## 6. La spec de collecte live

Le même descripteur décrit l'API live, pilotée par le moteur de collecte commun
(`internal/collect`) : aucun code Go pour une API REST/JSON.

```yaml
collecte:
  base_url: https://api.{region}.acme.example/v1
  resources:
    - type: compute_instance
      path: /instances
      items: instances
      id: vm_id
      map:
        vm_id: id
        state: state
        public_ip: public_ip
        security_group_ids: security_groups
```

Les points qui décident de la fiabilité du résultat :

- **La pagination.** Une liste tronquée en silence produit un rapport qui rate des
  ressources. Le moteur implémente quatre styles (`page`, `token`, `offset-body`,
  `token-body`) et refuse de tronquer sans le dire : il rend une erreur quand il atteint
  sa borne de pages. Déclarer le style de votre API plutôt que d'accepter la première
  page.
- **Les jointures.** `for_each` couvre le « lister les parents, puis appeler le détail
  par élément », qui est la façon d'atteindre les user-data, les clés par utilisateur ou
  le détail par instance.
- **La région est posée sur chaque ressource** par le moteur de collecte, ce qui rend la
  localisation observable en live pour tout fournisseur qui collecte quoi que ce soit.

Deux collecteurs sont du **code Go partagé**, pas de la spec : le stockage objet
(compatible S3) et le Kubernetes managé. On les active en déclarant leur endpoint :

```yaml
s3:
  endpoint: "https://s3.{region}.acme.example"
  region: "{region}"
  sse_kms: false        # true seulement si le fournisseur expose une clé client par bucket
oks:
  endpoint: "https://api.{region}.oks.acme.example"
```

## 7. Le contrat : ce qui est vérifié, ce qui ne l'est pas, ce qui n'existe pas

Le contrat est la couche d'honnêteté, et c'est lui qui pilote l'assessment. Pour chaque
type normalisé, l'un des trois états :

```yaml
contrat:
  types:
    security_group_rule:
      etat: verifie          # lu dans le SDK/l'API et réellement projeté
      source: "GET /security-groups — SecurityGroupRule.{direction,protocol,from_port}"
      mapping:
        protocol: Protocol (tcp|udp|icmp)
    blockstorage_volume:
      etat: a_verifier       # plausible, non lu : aucun « pass » ne sera jamais affirmé
      source: "à confirmer dans le SDK"
  non_applicable:
    - control: loadbalancer_http_redirect_to_https
      reason: "aucun mécanisme de redirection HTTP→HTTPS dans l'API des load balancers"
      reason_en: "no HTTP→HTTPS redirection mechanism in the load balancer API"
```

Trois invariants sont tenus par des tests :

- `TestContractVerifiedTypesAreCollected` : un type déclaré `verifie` **doit** être
  collecté ou mappé. Sinon les règles qui le lisent sont mortes, chargées mais jamais
  alimentées.
- `TestEveryContractJustificationIsBilingual` : toute justification `non_applicable`
  porte sa contrepartie anglaise, et la version anglaise ne porte aucun caractère
  accentué. C'est ce motif qu'un auditeur lit en face d'un `not-applicable` ; un N/A sans
  motif n'est pas opposable, et un N/A dont le motif bascule de langue ne l'est pas
  davantage.
- `TestProvidersValid` : identité, auth, identifiants et sources ne sont pas vides.

`a_verifier` n'est pas un échec, c'est un état honnête : il met le `pass` hors d'atteinte
et fait dire `partial` à la matrice de couverture, avec le motif. Déclarer `verifie` pour
verdir une case est la seule chose qui ne doit jamais arriver.

## 8. Les fixtures : ce qui est simulé, et ce qui est mesuré

Les fixtures sont des réponses d'API et des plans **réalistes**, dans le vocabulaire
natif du fournisseur :

```
examples/<nom>/terraform/         # un plan volontairement non conforme (plan.json)
examples/<nom>/inventory.json     # un inventaire normalisé, pour un scan hors ligne
```

Soyez explicite sur les deux registres, car ils n'ont pas le même poids :

- **Simulé** : une fixture committée prouve le *mapping* et la *règle*. Elle ne prouve
  rien du comportement de l'API.
- **Mesuré** : une exécution réelle (un plan produit depuis du vrai Terraform, ou un scan
  live) prouve le *contrat*, c'est-à-dire que l'API rend bien ce champ, sous cette forme.

Une entrée de contrat passe à `verifie` sur la foi du second, jamais du premier. Dites
lequel des deux vous avez fait dans la pull request : la différence est toute la valeur
de l'outil.

## 9. Régions et zones

`region_key` nomme la clé logique qu'alimente `--region`, parce que les fournisseurs ne
s'accordent pas sur le mot : Exoscale raisonne en zones, d'autres en régions. Elle
substitue `{region}` ou `{zone}` dans `base_url`, dans l'endpoint S3 et dans les chemins.
Déclarez une valeur `defaults` raisonnable pour qu'un premier scan fonctionne sans
cérémonie.

## 10. Les permissions de moindre privilège

Un scan Pépin est en **lecture seule**. La liste exacte des appels d'API que la collecte
live effectue est dérivée de votre descripteur et publiée sur la page du fournisseur :
c'est donc la liste des droits que doit porter une clé de lecture. Ne maintenez pas cette
liste à la main, elle est générée.

Pour donner sa page à votre fournisseur, l'ajouter à `documentedProviders` dans
[`internal/docgen/providers.go`](../../internal/docgen/providers.go) et créer
`docs/providers/<nom>.md` et `docs/providers/<nom>.fr.md` avec les régions générées
(`provider-<nom>-identity`, `-credentials`, `-live`, `-terraform`, `-coverage`, `-na`,
`-onesource`, `-scan`). Copier une page existante : tout ce qu'elle affirme de factuel y
est injecté.

## 11. Enregistrer, compiler, vérifier

Les descripteurs sont **embarqués** dans le binaire (`embed.go`,
`//go:embed all:providers`) : ajouter un cloud demande donc une recompilation, mais aucun
code Go.

```bash
mise run build
./pepin provider list
./pepin scan <nom> --terraform examples/<nom>/terraform/plan.json
```

Puis les portes :

```bash
mise run validate   # référentiel ↔ règles ↔ index SCSL gelé ↔ catalogue ↔ bilinguisme
mise run test       # tests Go (-race) + tests Rego + fraîcheur de la documentation
mise run audit      # vet + lint + gosec + govulncheck + osv
```

La documentation n'est pas dans cette liste parce qu'elle n'est pas seulement une porte :
c'est l'étape 13.

## 12. Valider sur un vrai compte, si vous en avez un

Seulement si le scan live est la seule façon de confirmer le contrat :

1. Scanner d'abord en **lecture seule**, sans rien provisionner. La plupart des questions
   de contrat trouvent leur réponse dans ce qu'un tenant existant rend déjà.
2. Si une ressource doit exister pour observer un champ, créer le strict minimum, dans un
   projet isolé, et noter ce que vous avez créé.
3. Observer, puis consigner le champ au contrat avec sa source.
4. **Tout détruire**, et confirmer la destruction avant de conclure.

Ne jamais laisser vivre une ressource de test. Et ne jamais annoncer une couverture
qu'une exécution n'a pas montrée : un contrôle non validé reste `fournisseurs: []`,
écrit, testé, activation gelée.

## 13. Documenter le fournisseur, la septième étape de la procédure

La procédure canonique (`CLAUDE.md` §10, et la même exigence dans
[CONTRIBUTING.fr.md](../../CONTRIBUTING.fr.md)) se termine sur **documenter**, et c'est
là qu'un fournisseur nouveau fait le plus de dégâts si on saute l'étape : un cloud qui
apparaît dans le binaire mais pas dans la documentation est un cloud dont personne ne
peut vérifier la couverture.

**Régénérer**, et committer ce qui change :

```bash
mise run gen-docs
```

La matrice de couverture gagne deux colonnes (votre source Terraform et votre source
live), les pages du catalogue gagnent votre fournisseur dans leurs listes « actif pour »,
les chiffres bougent, et les régions générées de votre page fournisseur se remplissent.
`TestGeneratedDocsAreUpToDate` échoue sur tout ce qui reste en arrière.

**Relire ce que le nouveau fournisseur rend faux** : le générateur ne possède que les
régions entre ses marqueurs.

- [`docs/known-limitations.fr.md`](../known-limitations.fr.md) : si une limite y était
  formulée comme « aucun cloud souverain de Pépin ne mesure X », votre fournisseur vient
  peut-être de la lever.
- la prose de [`docs/coverage.fr.md`](../coverage.fr.md), et sa clé de lecture si vous
  avez introduit une source qui se comporte autrement.
- votre propre page fournisseur : tout ce qui est hors des régions générées est écrit à
  la main, et c'est là que l'ancrage se dit, les appels d'API, les permissions minimales
  en lecture seule, ce qui est vérifié et ce qui ne l'est pas.

Dans les **deux langues**, toujours. Une page fournisseur qui n'existe qu'en anglais est
une page que la moitié des lecteurs de ce projet ne peut pas utiliser.

**Ajouter une ligne au CHANGELOG**, dans les deux langues. Un fournisseur nouveau ajoute
une **surface analysable** : un scan jusque-là impossible rend désormais des findings,
des statuts et un code de sortie. C'est exactement le genre de changement que le
CHANGELOG existe pour consigner, quelle que soit sa taille dans le diff.

---

## Checklist de release du fournisseur

- [ ] `providers/<nom>.yaml` existe, et **aucun** fichier `.rego` n'a été ajouté.
- [ ] Identité, `scope` et `region_key` sont posés ; `pepin provider list` montre le
      fournisseur.
- [ ] Les faits de souveraineté sont remplis, avec la chaîne capitalistique **sourcée**.
- [ ] L'auth et la résolution des identifiants fonctionnent depuis l'environnement **et**
      depuis le fichier de configuration natif.
- [ ] Chaque attribut mappé est ancré : lu dans le SDK ou la doc officielle, et cité (une
      page de `references/docs/<nom>/`, ou la `source` du contrat).
- [ ] Les attributs dérivés sont marqués comme dérivés, avec leur formule.
- [ ] `TestProviderMappingsMatchSchema` a été lancé **sur une machine avec Terraform**, et
      le mapping correspond au schéma réel du provider.
- [ ] Tout type marqué `verifie` est réellement collecté ou mappé ; ce qui n'a pas été lu
      reste `a_verifier`.
- [ ] Tout `non_applicable` porte sa justification bilingue.
- [ ] Les fixtures sont réalistes et **ne contiennent aucun secret**.
- [ ] La pull request dit explicitement ce qui a été **simulé** et ce qui a été **mesuré**.
- [ ] Toute ressource provisionnée pour un test a été **détruite**, et la destruction est
      confirmée.
- [ ] La page du fournisseur existe dans les deux langues, avec ses régions générées.
- [ ] `mise run gen-docs` a été lancé et le résultat committé.
- [ ] Les pages écrites à la main que le fournisseur rend fausses ont été relues, dans
      les deux langues : limites connues, prose de la matrice, page du fournisseur.
- [ ] Une ligne au CHANGELOG, dans les deux langues : un fournisseur nouveau ajoute une
      surface analysable.
- [ ] `mise run validate`, `mise run test` et `mise run audit` sont au vert.

## Voir aussi

- [Architecture](../project/architecture.fr.md) : c'est la source qui change, pas la
  règle.
- [Ajouter un contrôle](adding-a-control.fr.md) : l'autre moitié, la règle.
- [Plan Terraform ou scan live](../concepts/terraform-vs-live.fr.md) : ce que chaque
  source peut montrer, et ce qu'elle ne peut pas.
- [Matrice de couverture](../coverage.fr.md) : où votre fournisseur apparaîtra, et
  pourquoi une case n'est pas verte.
- Les descripteurs existants, qui sont la vraie référence :
  [`providers/exoscale.yaml`](../../providers/exoscale.yaml),
  [`providers/outscale.yaml`](../../providers/outscale.yaml),
  [`providers/scaleway.yaml`](../../providers/scaleway.yaml).
