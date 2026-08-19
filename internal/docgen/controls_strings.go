package docgen

// Les libellés des pages du catalogue des contrôles, dans une langue.
//
// Seuls les EN-TÊTES et les intitulés de situation vivent ici. Le contenu (titres de
// contrôles, descriptions, remédiations, motifs de non-couverture) vient toujours du
// référentiel et des contrats de fournisseurs, qui sont eux-mêmes bilingues : les deux
// versions d'une page disent donc exactement la même chose.
type controlStrings struct {
	indexTitle, indexIntro                     string
	figuresTitle, readingTitle, readingBody    string
	dormantTitle, dormantIntro, noDormant      string
	dormant, none                              string
	colControl, colSeverity, colActiveFor      string
	colProofs, colFamily, colFigure, colCount  string
	figTotal, figActive, figDormant, figProofs string

	backToIndex                                string
	colField, colValue                         string
	rowCode, rowFamily, rowSeverity, rowSCSL   string
	rowType, rowAttrs, rowState, rowDeclared   string
	rowProofs, noType, noAttr                  string
	stateActive, stateDormant                  string
	whyTitle, whyNote                          string
	mappingTitle, mappingIntro                 string
	colFramework, colRefs                      string
	whereTitle, whereIntro, reasonsIntro       string
	colProvider, colTerraform, colLive         string
	colSource, colStatus, colReason, noReason  string
	concludeTitle, colMeans, colWhere          string
	meansFail, meansPass, meansNA, meansNE     string
	concludeNote                               string
	investigateTitle, investType, investNoType string
	investAttrs, investNoAttrs, investGuard    string
	investDescriptors, investRule              string
	remediateTitle, colProof, proofMissing     string
	proofDormant, proofNote                    string
	verifyTitle, verifyTF, verifyLive          string
	verifyBodyFmt, verifyNone, verifyPartial   string
	noSource                                   string
	seeAlsoTitle, seeAlso                      string
}

func controlText(lang string) controlStrings {
	if lang == "fr" {
		return controlStrings{
			indexTitle: "Catalogue des contrôles",
			indexIntro: "Un contrôle par page, calculé depuis `referentiel/controles.yaml` (la source de vérité),\n" +
				"les descripteurs `providers/*.yaml` et le verrou du « pass » de `internal/assess`.\n" +
				"Aucune métadonnée n'est recopiée ici : ajouter un contrôle, changer une sévérité ou\n" +
				"retirer un fournisseur réécrit ces pages, et la CI refuse une documentation en retard.\n\n" +
				"Ce catalogue dit ce que Pépin **peut conclure**, pas le résultat d'un scan. Pour la\n" +
				"vue d'ensemble par fournisseur et par source, voir la [matrice de couverture](../coverage.md).",
			figuresTitle: "Chiffres",
			readingTitle: "Comment lire ce catalogue",
			readingBody: "- **Contrôle actif** : déclaré pour au moins un fournisseur (`fournisseurs:` non vide).\n" +
				"  Un scan de ce fournisseur peut lui donner un statut.\n" +
				"- **Contrôle dormant** : écrit et testé, déclaré pour aucun fournisseur. Il n'est jamais\n" +
				"  évalué, et il ne compte dans aucune couverture. La liste complète est en bas de page.\n" +
				"- **Preuves de remédiation** : les montages déployables présents sous\n" +
				"  [`references/remediation/`](../../references/remediation/README.md). Le compte est\n" +
				"  celui des couples (contrôle, fournisseur) déclarés, et il est aujourd'hui partiel :\n" +
				"  la remédiation *textuelle*, elle, est portée par chaque écart émis.\n" +
				"- Les contrôles encore au triage, non retenus dans le référentiel actif, vivent au\n" +
				"  [catalogue de triage](../../referentiel/catalogue.yaml).",
			dormantTitle: "Contrôles dormants",
			dormantIntro: "Ces contrôles existent au référentiel mais ne sont déclarés pour aucun fournisseur :\n" +
				"aucun scan ne les évalue aujourd'hui. Ils apparaissent ici pour que le catalogue ne se\n" +
				"lise pas comme une couverture.",
			noDormant:  "_Aucun : tout contrôle du référentiel est déclaré pour au moins un fournisseur._",
			dormant:    "dormant",
			none:       "aucun",
			colControl: "Contrôle", colSeverity: "Sévérité", colActiveFor: "Actif pour",
			colProofs: "Preuves", colFamily: "Famille", colFigure: "Chiffre", colCount: "Nombre",
			figTotal: "Contrôles au référentiel", figActive: "Contrôles actifs",
			figDormant: "Contrôles dormants", figProofs: "Preuves de remédiation déployables",

			backToIndex: "Retour au catalogue",
			colField:    "Champ", colValue: "Valeur",
			rowCode: "Code", rowFamily: "Famille", rowSeverity: "Sévérité",
			rowSCSL: "Exigence SCSL (index gelé)", rowType: "Type de ressource lu",
			rowAttrs: "Attribut décisif", rowState: "État", rowDeclared: "Déclaré pour",
			rowProofs: "Preuves de remédiation",
			noType:    "aucun : contrôle transverse", noAttr: "aucun : jugé à la présence d'un écart",
			stateActive: "actif", stateDormant: "dormant (déclaré pour aucun fournisseur)",
			whyTitle: "Le risque",
			whyNote: "Cette description vient du référentiel : c'est le texte que le rapport cite, dans la\n" +
				"langue du lecteur.",
			mappingTitle: "Correspondances normatives",
			mappingIntro: "Reprises telles quelles du référentiel. L'exigence SCSL provient de l'index **gelé** :\n" +
				"un contrôle se rattache à une exigence existante, jamais à une exigence créée pour lui.\n" +
				"Ces correspondances sont **indicatives** : un rapport Pépin n'est pas une preuve de\n" +
				"qualification.",
			whereTitle: "Où Pépin sait le mesurer",
			whereIntro: "Une case ✅ signifie que la source produit le type visé, que le contrat du fournisseur\n" +
				"le déclare `verifie` et que l'attribut décisif est projeté. ◐ signifie « Pépin ne peut\n" +
				"pas décider depuis cette source », ∅ « non testable, avec justification », ✗ « non\n" +
				"déclaré, ou type absent de cette source ».",
			reasonsIntro: "Chaque case qui n'est pas ✅, **alors que le contrôle est déclaré pour ce fournisseur**,\n" +
				"porte son motif :",
			colProvider: "Fournisseur", colTerraform: "Plan Terraform", colLive: "Collecte live",
			colFramework: "Cadre", colRefs: "Références",
			colSource: "Source", colStatus: "Statut", colReason: "Motif",
			noReason:      "_Aucune : toutes les cases déclarées sont pleinement observables._",
			concludeTitle: "Ce que Pépin peut conclure",
			colMeans:      "Ce que le statut affirme", colWhere: "Atteignable depuis",
			meansFail: "un écart a été détecté sur une ressource réelle",
			meansPass: "la donnée décisive a été collectée, et elle est conforme",
			meansNA:   "le contrat du fournisseur déclare le contrôle non testable, avec sa justification",
			meansNE:   "le contrôle est implémenté, mais la donnée dont il dépend n'a pas été confirmée",
			concludeNote: "Un contrôle observable rend tout de même `not-evaluated` sur un inventaire qui ne\n" +
				"contient aucune ressource du type visé : « rien à voir » n'est pas « conforme ».",
			investigateTitle: "Comment enquêter",
			investType:       "Type de ressource normalisé lu par la règle :",
			investNoType:     "Contrôle transverse : il ne lit pas un type de ressource particulier.",
			investAttrs:      "Attribut dont la décision dépend :",
			investNoAttrs: "Aucun verrou d'attribut : le contrôle se juge à la présence d'un écart, " +
				"l'absence de mauvaise configuration valant conformité.",
			investGuard: "Sans cet attribut sur une ressource du type visé, le scan rend `not-evaluated` " +
				"et non `pass` (`internal/assess`, table `requiredAttr`).",
			investDescriptors: "Ce que chaque source projette se lit dans le descripteur :",
			investRule: "La règle qui émet ce code vit dans " +
				"[`internal/commonrules/rules/`](../../internal/commonrules/rules) : elle est **commune** " +
				"à tous les fournisseurs, seule la source change.",
			remediateTitle: "Comment corriger",
			colProof:       "Montage déployable",
			proofMissing:   "aucune preuve déposée à ce jour",
			proofDormant:   "_Contrôle dormant : aucun fournisseur déclaré, donc aucune preuve attendue._",
			proofNote: "Une preuve de remédiation est un module Terraform autonome, **conforme**, qui se déploie\n" +
				"tel quel, ou une note ancrée sur la documentation officielle. Voir\n" +
				"[le guide de remédiation](../guides/remediation.md).",
			verifyTitle: "Comment vérifier la correction",
			verifyTF:    "depuis un plan Terraform : aucune ressource n'est créée",
			verifyLive:  "depuis l'API du fournisseur : configuration effective",
			verifyBodyFmt: "Dans la sortie `assessment`, chercher `\"control\": \"%s\"` : son `status` doit être\n" +
				"`pass`. S'il reste `not-evaluated`, la donnée décisive n'a pas été collectée, et la\n" +
				"correction n'est **pas** démontrée : le tableau des motifs ci-dessus dit pourquoi.",
			verifyNone: "Aucune source ne sait aujourd'hui conclure `pass` sur ce contrôle : un scan\n" +
				"peut faire disparaître l'écart, il ne peut pas **démontrer** la conformité. Le tableau\n" +
				"des motifs ci-dessus dit ce qui manque.",
			verifyPartial: "**Une des deux sources ne sait pas lever le verrou du « pass »** pour ce contrôle :\n" +
				"le fournisseur cité y produit bien le type visé, mais le scan y rendra `not-evaluated`.\n" +
				"Le tableau des motifs dit laquelle, et pourquoi.",
			noSource:     "sans objet",
			seeAlsoTitle: "Voir aussi",
			seeAlso: "- [Le modèle d'assessment](../concepts/assessment-model.md) : ce que chaque statut affirme.\n" +
				"- [Matrice de couverture](../coverage.md) : la même information, tous contrôles confondus.\n" +
				"- [Plan Terraform ou scan live](../concepts/terraform-vs-live.md) : choisir la source.\n" +
				"- [Ajouter un contrôle](../contributing/adding-a-control.md) : la procédure de bout en bout.",
		}
	}
	return controlStrings{
		indexTitle: "Control catalogue",
		indexIntro: "One page per control, computed from `referentiel/controles.yaml` (the source of truth),\n" +
			"the `providers/*.yaml` descriptors and the `pass` lock in `internal/assess`.\n" +
			"No metadata is copied here: adding a control, changing a severity or removing a provider\n" +
			"rewrites these pages, and CI rejects documentation that lags behind.\n\n" +
			"This catalogue states what Pépin **can conclude**, not the result of any scan. For the\n" +
			"overview per provider and per source, see the [coverage matrix](../coverage.md).",
		figuresTitle: "Figures",
		readingTitle: "How to read this catalogue",
		readingBody: "- **Active control**: declared for at least one provider (non-empty `fournisseurs:`).\n" +
			"  A scan of that provider can give it a status.\n" +
			"- **Dormant control**: written and tested, declared for no provider. It is never evaluated,\n" +
			"  and it counts towards no coverage. The full list is at the bottom of this page.\n" +
			"- **Remediation proofs**: the deployable setups present under\n" +
			"  [`references/remediation/`](../../references/remediation/README.md). The count is over\n" +
			"  declared (control, provider) pairs, and it is partial today: the *textual* remediation,\n" +
			"  on the other hand, is carried by every deviation reported.\n" +
			"- Controls still being triaged, not retained in the active reference, live in the\n" +
			"  [triage catalogue](../../referentiel/catalogue.yaml).",
		dormantTitle: "Dormant controls",
		dormantIntro: "These controls exist in the reference but are declared for no provider: no scan evaluates\n" +
			"them today. They appear here so that the catalogue is not read as coverage.",
		noDormant:  "_None: every control in the reference is declared for at least one provider._",
		dormant:    "dormant",
		none:       "—",
		colControl: "Control", colSeverity: "Severity", colActiveFor: "Active for",
		colProofs: "Proofs", colFamily: "Family", colFigure: "Figure", colCount: "Count",
		figTotal: "Controls in the reference", figActive: "Active controls",
		figDormant: "Dormant controls", figProofs: "Deployable remediation proofs",

		backToIndex: "Back to the catalogue",
		colField:    "Field", colValue: "Value",
		rowCode: "Code", rowFamily: "Family", rowSeverity: "Severity",
		rowSCSL: "SCSL requirement (frozen index)", rowType: "Resource type read",
		rowAttrs: "Deciding attribute", rowState: "State", rowDeclared: "Declared for",
		rowProofs: "Remediation proofs",
		noType:    "none: cross-cutting control", noAttr: "none: judged on the presence of a deviation",
		stateActive: "active", stateDormant: "dormant (declared for no provider)",
		whyTitle: "The risk",
		whyNote: "This description comes from the reference: it is the text the report quotes, in the\n" +
			"reader's language.",
		mappingTitle: "Normative mappings",
		mappingIntro: "Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:\n" +
			"a control maps onto an existing requirement, never onto one created for it. These mappings\n" +
			"are **indicative**: a Pépin report is not proof of qualification.",
		whereTitle: "Where Pépin can measure it",
		whereIntro: "A ✅ cell means the source produces the targeted type, the provider contract marks it\n" +
			"`verifie`, and the deciding attribute is projected. ◐ means \"Pépin cannot decide from this\n" +
			"source\", ∅ \"not testable, with justification\", ✗ \"not declared, or type absent from this\n" +
			"source\".",
		reasonsIntro: "Every cell that is not ✅ **while the control is declared for that provider** carries its\n" +
			"reason:",
		colProvider: "Provider", colTerraform: "Terraform plan", colLive: "Live collection",
		colFramework: "Framework", colRefs: "References",
		colSource: "Source", colStatus: "Status", colReason: "Reason",
		noReason:      "_None: every declared cell is fully observable._",
		concludeTitle: "What Pépin can conclude",
		colMeans:      "What the status asserts", colWhere: "Reachable from",
		meansFail: "a deviation was detected on a real resource",
		meansPass: "the deciding data was collected, and it is compliant",
		meansNA:   "the provider contract declares the control untestable, with its justification",
		meansNE:   "the control is implemented, but the data it depends on was not confirmed",
		concludeNote: "An observable control still returns `not-evaluated` on an inventory that contains no\n" +
			"resource of the targeted type: \"nothing to look at\" is not \"compliant\".",
		investigateTitle: "How to investigate",
		investType:       "Normalized resource type the rule reads:",
		investNoType:     "Cross-cutting control: it does not read one particular resource type.",
		investAttrs:      "Attribute the decision depends on:",
		investNoAttrs: "No attribute lock: the control is judged on the presence of a deviation, " +
			"absence of a bad configuration counting as compliance.",
		investGuard: "Without that attribute on a resource of the targeted type, the scan returns " +
			"`not-evaluated` rather than `pass` (`internal/assess`, `requiredAttr` table).",
		investDescriptors: "What each source projects is readable in the descriptor:",
		investRule: "The rule that emits this code lives in " +
			"[`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every " +
			"provider, only the source changes.",
		remediateTitle: "How to remediate",
		colProof:       "Deployable setup",
		proofMissing:   "no proof filed yet",
		proofDormant:   "_Dormant control: no provider declared, so no proof expected._",
		proofNote: "A remediation proof is a self-contained, **compliant** Terraform module that deploys as\n" +
			"is, or a note anchored on the official documentation. See\n" +
			"[the remediation guide](../guides/remediation.md).",
		verifyTitle: "How to verify the fix",
		verifyTF:    "from a Terraform plan: nothing is provisioned",
		verifyLive:  "from the provider API: effective configuration",
		verifyBodyFmt: "In the `assessment` output, look for `\"control\": \"%s\"`: its `status` must be `pass`.\n" +
			"If it stays `not-evaluated`, the deciding data was not collected and the fix is **not**\n" +
			"demonstrated: the reasons table above says why.",
		verifyNone: "No source can conclude `pass` on this control today: a scan can make the deviation\n" +
			"disappear, it cannot **demonstrate** compliance. The reasons table above says what is\n" +
			"missing.",
		verifyPartial: "**One of the two sources cannot lift the `pass` lock** for this control: the provider\n" +
			"quoted does produce the targeted type there, but the scan will return `not-evaluated`.\n" +
			"The reasons table says which one, and why.",
		noSource:     "n/a",
		seeAlsoTitle: "See also",
		seeAlso: "- [The assessment model](../concepts/assessment-model.md): what each status asserts.\n" +
			"- [Coverage matrix](../coverage.md): the same information, across every control.\n" +
			"- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.\n" +
			"- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.",
	}
}
