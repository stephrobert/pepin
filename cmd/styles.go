package cmd

import "github.com/charmbracelet/lipgloss"

// Tokens de la charte SCSL (fond sombre, accent rouge). Le rendu des findings
// (sévérités, verdict) vient de scankit/report ; ne restent ici que les styles
// propres au CLI de Pépin (bandeau providers, message d'erreur racine).
var (
	colRouge = lipgloss.Color("#f04438")
	colTexte = lipgloss.Color("#f3f5f9")
	colMuted = lipgloss.Color("#a3acbd")

	eyebrow  = lipgloss.NewStyle().Foreground(colRouge).Bold(true)
	titre    = lipgloss.NewStyle().Foreground(colTexte).Bold(true)
	muted    = lipgloss.NewStyle().Foreground(colMuted)
	errStyle = lipgloss.NewStyle().Foreground(colRouge)
)
