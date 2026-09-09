package docgen

import (
	"strings"
	"testing"
)

// La matrice DÉCLARAIT ce que le mapping nomme, au lieu de MESURER ce qu'un plan
// porte. Sur un plan Outscale réaliste, `public_ip` est calculé : Terraform ne le
// connaît qu'après `apply`, donc il est absent — et la cellule affichait ✅ parce que
// la spec le nomme. Les scénarios de véracité confirmaient la cellule sur un plan
// écrit à la main où l'attribut est un littéral : une fixture qui se confirme
// elle-même n'établit rien.
//
// Les tenants de référence sont générés depuis du HCL TIERS, pas écrit pour Pépin :
// c'est le témoin qui manquait.

// TestTheTerraformCoverageIsMeasuredNotDeclared : là où un plan de référence exerce un
// type, un attribut que le mapping déclare mais qu'aucun plan ne porte ne compte pas
// comme projeté.
func TestTheTerraformCoverageIsMeasuredNotDeclared(t *testing.T) {
	m, err := BuildMatrix(repoRoot, "fr")
	if err != nil {
		t.Fatalf("construction de la matrice : %v", err)
	}
	var vue bool
	for _, row := range m.Rows {
		if row.Code != "compute_instance_public_ip_with_open_securitygroup" {
			continue
		}
		vue = true
		c := row.Cells["outscale"][SourceTerraform]
		if c.Status != Partial {
			t.Errorf("outscale/terraform = %q, attendu %q : `public_ip` est calculé, aucun plan ne le porte",
				c.Status, Partial)
		}
		// Le MOTIF compte autant que le statut : « le mapping ne le nomme pas » se
		// corrige dans la spec, « le mapping le nomme mais aucun plan ne le porte »
		// ne se corrige pas ainsi. Donner l'un pour l'autre envoie corriger dans le vide.
		if !strings.Contains(c.Reason, "après `apply`") {
			t.Errorf("le motif ne dit pas que la valeur n'existe qu'après apply :\n  %s", c.Reason)
		}
	}
	if !vue {
		t.Fatal("contrôle absent de la matrice : la garde ne mesure rien")
	}
}

// LE CONTRE-EXEMPLE : la mesure ne doit pas tout dégrader. Un attribut que les plans
// de référence portent RÉELLEMENT laisse sa cellule intacte — sans quoi la matrice
// aurait simplement échangé un excès de promesse contre un excès de prudence.
func TestAnAttributeRealPlansDoCarryStaysSupported(t *testing.T) {
	m, err := BuildMatrix(repoRoot, "fr")
	if err != nil {
		t.Fatalf("construction de la matrice : %v", err)
	}
	var supportees int
	for _, row := range m.Rows {
		for _, prov := range m.CloudProviders {
			if row.Cells[prov][SourceTerraform].Status == Supported {
				supportees++
			}
		}
	}
	if supportees == 0 {
		t.Fatal("plus aucune cellule terraform `supported` : la mesure a tout dégradé, ce qui n'est pas une correction")
	}
}
