// Package scoring résume les findings d'un scan : comptes par sévérité et
// verdict de conformité. Le verdict est NON CONFORME dès qu'un finding
// critical ou high est présent (gouverne le code de sortie 1).
package scoring

import "github.com/stephrobert/scankit/finding"

// Result agrège les findings d'un scan.
type Result struct {
	Total    int  `json:"total"`
	Critical int  `json:"critical"`
	High     int  `json:"high"`
	Medium   int  `json:"medium"`
	Low      int  `json:"low"`
	Conforme bool `json:"conforme"`
}

// Summarize compte les findings par sévérité et calcule le verdict.
func Summarize(findings []finding.Finding) Result {
	r := Result{Total: len(findings)}
	for _, f := range findings {
		switch f.Severity {
		case "critical":
			r.Critical++
		case "high":
			r.High++
		case "medium":
			r.Medium++
		case "low":
			r.Low++
		}
	}
	r.Conforme = r.Critical == 0 && r.High == 0
	return r
}
