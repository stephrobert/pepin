package docgen

// blockStrings porte les libellés des régions injectées, dans une langue. Le CONTENU des
// tableaux vient toujours du dépôt ; seuls les en-têtes et les intitulés de situation sont
// traduits, pour que les deux versions d'une page disent exactement la même chose.
type blockStrings struct {
	colSituation, colCommand, colExit                 string
	colStatus, colCount, colWitness, colReason        string
	colControl, colType, colDeciding, colSeverity     string
	colProvider, colOnlyVia, colJustification         string
	colProofs, colFigure                              string
	exitCompliant, exitNonCompliance, exitError       string
	exitNothing, exitMediumPlain, exitMediumStrict    string
	exitPartial                                       string
	veracityPaths, veracityProven                     string
	veracityObligations, veracityRemaining            string
	veracityUnavailable                               string
	colAttributes                                     string
	exitExempted, exitExpired                         string
	orWord, none, totalWord, figControls, figDeclared string
	noTypeWord                                        string
	noDeviationFor, noResultFor, quotedPlaceholder    string
}

func blockText(lang string) blockStrings {
	if lang == "fr" {
		return blockStrings{
			colSituation: "Situation", colCommand: "Commande", colExit: "Code de sortie",
			colStatus: "Statut", colCount: "Nombre", colWitness: "Contrôle témoin", colReason: "Motif",
			colControl: "Contrôle", colType: "Type de ressource lu", colDeciding: "Attribut décisif",
			colSeverity: "Sévérité", colProvider: "Fournisseur", colOnlyVia: "Observable uniquement via",
			colJustification: "Justification consignée au contrat", colProofs: "Preuves de remédiation",
			colFigure:           "Chiffre",
			exitCompliant:       "Aucun écart sur le périmètre évalué",
			exitNonCompliance:   "Au moins un écart critical ou high",
			exitError:           "Erreur technique (fichier illisible, provider inconnu, API injoignable)",
			exitNothing:         "Rien n'a été mesuré (inventaire vide) : **sans avoir à demander `--strict`**",
			exitMediumPlain:     "Écarts medium/low seulement, sans `--strict`",
			exitMediumStrict:    "Écarts medium/low seulement, avec `--strict`",
			exitPartial:         "Aucun écart, mais une unité de collecte n'a pas pu être lue",
			veracityPaths:       "Chemins contrôle × fournisseur × source sur lesquels Pépin conclut",
			veracityProven:      "Chemins dont tous les verdicts atteignables sont prouvés de bout en bout",
			veracityObligations: "Verdicts à prouver au total",
			veracityRemaining:   "Verdicts restant à prouver",
			veracityUnavailable: "_Contrat de véracité indisponible : le calcul des obligations a échoué._",
			exitExempted:        "Tout écart critical/high est couvert par une dérogation valide",
			exitExpired:         "La même dérogation, échue : elle ne s'applique plus",
			colAttributes:       "Attributs communs",
			orWord:              "ou",
			none:                "_Aucun._",
			totalWord:           "Total",
			figControls:         "Contrôles au référentiel",
			figDeclared:         "Contrôles déclarés pour au moins un fournisseur",
			noTypeWord:          "(aucun : contrôle transverse)",
			noDeviationFor:      "(aucun écart %s sur ce scan)",
			noResultFor:         "(aucun résultat « %s » sur ce scan)",
			quotedPlaceholder:   "« … »",
		}
	}
	return blockStrings{
		colSituation: "Situation", colCommand: "Command", colExit: "Exit code",
		colStatus: "Status", colCount: "Count", colWitness: "Witness control", colReason: "Reason",
		colControl: "Control", colType: "Resource type read", colDeciding: "Deciding attribute",
		colSeverity: "Severity", colProvider: "Provider", colOnlyVia: "Observable only through",
		colJustification: "Justification recorded in the contract", colProofs: "Remediation proofs",
		colFigure:           "Figure",
		exitCompliant:       "No deviation in the evaluated scope",
		exitNonCompliance:   "At least one critical or high deviation",
		exitError:           "Technical error (unreadable file, unknown provider, unreachable API)",
		exitNothing:         "Nothing was measured (empty inventory): **without having to ask for `--strict`**",
		exitMediumPlain:     "Medium/low deviations only, without `--strict`",
		exitMediumStrict:    "Medium/low deviations only, with `--strict`",
		exitPartial:         "No deviation, but one collection unit could not be read",
		veracityPaths:       "Control x provider x source paths on which Pépin concludes",
		veracityProven:      "Paths whose every reachable verdict is proven end to end",
		veracityObligations: "Verdicts to prove in total",
		veracityRemaining:   "Verdicts left to prove",
		veracityUnavailable: "_Veracity contract unavailable: the obligation computation failed._",
		exitExempted:        "Every critical/high deviation is covered by a valid exemption",
		exitExpired:         "The same exemption, lapsed: it no longer applies",
		colAttributes:       "Common attributes",
		orWord:              "or",
		none:                "_None._",
		totalWord:           "Total",
		figControls:         "Controls in the reference",
		figDeclared:         "Controls declared for at least one provider",
		noTypeWord:          "(none: cross-cutting control)",
		noDeviationFor:      "(no %s deviation in this scan)",
		noResultFor:         "(no \"%s\" result in this scan)",
		quotedPlaceholder:   "\"…\"",
	}
}
