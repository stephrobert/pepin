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
	orWord, none, totalWord, figControls, figDeclared string
	noTypeWord                                        string
}

func blockText(lang string) blockStrings {
	if lang == "fr" {
		return blockStrings{
			colSituation: "Situation", colCommand: "Commande", colExit: "Code de sortie",
			colStatus: "Statut", colCount: "Nombre", colWitness: "Contrôle témoin", colReason: "Motif",
			colControl: "Contrôle", colType: "Type de ressource lu", colDeciding: "Attribut décisif",
			colSeverity: "Sévérité", colProvider: "Fournisseur", colOnlyVia: "Observable uniquement via",
			colJustification: "Justification consignée au contrat", colProofs: "Preuves de remédiation",
			colFigure:         "Chiffre",
			exitCompliant:     "Aucun écart sur le périmètre évalué",
			exitNonCompliance: "Au moins un écart critical ou high",
			exitError:         "Erreur technique (fichier illisible, provider inconnu, API injoignable)",
			exitNothing:       "Rien n'a été mesuré (inventaire vide) : **sans avoir à demander `--strict`**",
			exitMediumPlain:   "Écarts medium/low seulement, sans `--strict`",
			exitMediumStrict:  "Écarts medium/low seulement, avec `--strict`",
			orWord:            "ou",
			none:              "_Aucun._",
			totalWord:         "Total",
			figControls:       "Contrôles au référentiel",
			figDeclared:       "Contrôles déclarés pour au moins un fournisseur",
			noTypeWord:        "(aucun : contrôle transverse)",
		}
	}
	return blockStrings{
		colSituation: "Situation", colCommand: "Command", colExit: "Exit code",
		colStatus: "Status", colCount: "Count", colWitness: "Witness control", colReason: "Reason",
		colControl: "Control", colType: "Resource type read", colDeciding: "Deciding attribute",
		colSeverity: "Severity", colProvider: "Provider", colOnlyVia: "Observable only through",
		colJustification: "Justification recorded in the contract", colProofs: "Remediation proofs",
		colFigure:         "Figure",
		exitCompliant:     "No deviation in the evaluated scope",
		exitNonCompliance: "At least one critical or high deviation",
		exitError:         "Technical error (unreadable file, unknown provider, unreachable API)",
		exitNothing:       "Nothing was measured (empty inventory): **without having to ask for `--strict`**",
		exitMediumPlain:   "Medium/low deviations only, without `--strict`",
		exitMediumStrict:  "Medium/low deviations only, with `--strict`",
		orWord:            "or",
		none:              "_None._",
		totalWord:         "Total",
		figControls:       "Controls in the reference",
		figDeclared:       "Controls declared for at least one provider",
		noTypeWord:        "(none: cross-cutting control)",
	}
}
