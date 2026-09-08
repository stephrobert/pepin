package assess

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stephrobert/scankit/assessment"
)

// prov construit un index de provenance minimal : type -> attributs CHERCHÉS.
func prov(typ string, attrs ...string) ProvenanceIndex {
	byAttr := map[string]AttrOrigin{}
	for _, a := range attrs {
		byAttr[a] = AttrOrigin{Sources: []string{"api:ListAPIKeys"}, Total: 1}
	}
	return ProvenanceIndex{typ: byAttr}
}

func resultats() assessment.Assessment {
	return assessment.Assessment{Results: []assessment.Result{
		{Control: "iam_accesskey_expiration_set", Status: assessment.Fail, Subject: "SCW123"},
	}}
}

// TestAttestedAbsenceDistinguishesSoughtFromNeverSought est le cœur de l'issue #121.
//
// Les deux situations sont indiscernables pour une règle ; la provenance les sépare,
// et l'assessment en conclut (ADR-0017).
func TestAttestedAbsenceDistinguishesSoughtFromNeverSought(t *testing.T) {
	cas := []struct {
		nom  string
		idx  ProvenanceIndex
		veut assessment.Status
	}{
		{
			// Le champ a été demandé, la source ne l'expose pas : chez Scaleway
			// `ExpiresAt` est un *time.Time et un nil signifie « aucune expiration ».
			// L'absence EST l'observation, et l'écart est réel.
			nom:  "cherché et non exposé : l'écart reste",
			idx:  prov("access_key", "access_key_id", "expiration_date"),
			veut: assessment.Fail,
		},
		{
			// Le type a bien été collecté — d'autres attributs sont attestés — mais
			// celui-ci n'a jamais été cherché. L'écart était déduit de rien.
			nom:  "jamais cherché, alors que le type est attesté : l'écart tombe",
			idx:  prov("access_key", "access_key_id"),
			veut: assessment.NotEvaluated,
		},
		{
			// Aucune attestation sur ce type : Pépin n'a pas collecté cet inventaire,
			// il l'a reçu. Il n'a donc rien à en dire, ni dans un sens ni dans l'autre.
			nom:  "aucune attestation sur le type : inchangé",
			idx:  prov("autre_type", "champ"),
			veut: assessment.Fail,
		},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got := WithAttestedAbsence(resultats(), c.idx)
			if got.Results[0].Status != c.veut {
				t.Fatalf("statut %q, attendu %q", got.Results[0].Status, c.veut)
			}
			if c.veut == assessment.NotEvaluated && got.Results[0].Evidence.Observed == "" {
				t.Error("un « non évalué » muet n'est pas opposable : il doit nommer le champ jamais cherché")
			}
		})
	}
}

// TestInventoryWithoutProvenanceIsUntouched — invariant 3 de l'ADR-0017.
//
// Un inventaire reçu d'un tiers ne porte aucune provenance. Dégrader sur cette base
// remplacerait un faux positif par un faux vert, ce qui serait strictement pire.
func TestInventoryWithoutProvenanceIsUntouched(t *testing.T) {
	before := resultats()
	after := WithAttestedAbsence(before, ProvenanceIndex{})
	if after.Results[0].Status != before.Results[0].Status {
		t.Errorf("statut %q → %q sans aucune provenance : une absence d'attestation ne conclut rien",
			before.Results[0].Status, after.Results[0].Status)
	}
}

// TestAttestedAbsenceOnlyRemovesFindings — invariant 2 de l'ADR-0017.
//
// L'attestation ne peut que RETIRER une affirmation. Créer un écart, ou transformer
// un « non évalué » en « conforme » sur la seule foi d'une provenance, serait une
// porte ouverte que cet ADR n'accorde pas.
func TestAttestedAbsenceOnlyRemovesFindings(t *testing.T) {
	// `exempted` est le cinquième statut, propre à Pépin (internal/exempt) : il est
	// écrit en clair ici pour ne pas faire dépendre le paquet assess du paquet exempt.
	depart := []assessment.Status{
		assessment.Pass, assessment.NotEvaluated, assessment.NotApplicable,
		assessment.Error, assessment.Status("exempted"),
	}
	for _, st := range depart {
		a := assessment.Assessment{Results: []assessment.Result{
			{Control: "iam_accesskey_expiration_set", Status: st, Subject: "SCW123"},
		}}
		// L'index le plus « dégradant » possible : le type est attesté, le champ non.
		got := WithAttestedAbsence(a, prov("access_key", "access_key_id"))
		if got.Results[0].Status != st {
			t.Errorf("statut %q → %q : la passe a touché autre chose qu'un écart", st, got.Results[0].Status)
		}
	}
}

// TestAbsenceDecidersReallyReadAnAbsence confronte la table aux règles.
//
// Un contrôle n'a sa place dans `absenceMustBeAttested` que si sa règle conclut
// VRAIMENT depuis l'absence d'un attribut — la forme `object.get(…, "attr", "") == ""`
// ou son équivalent. Sans cette garde, la table deviendrait une liste d'intentions,
// et un contrôle y resterait après que sa règle a changé de forme.
func TestAbsenceDecidersReallyReadAnAbsence(t *testing.T) {
	dir := filepath.Join("..", "commonrules", "rules")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("règles indisponibles : %v", err)
	}
	byCode := map[string]string{}
	for _, e := range entries {
		n := e.Name()
		if !strings.HasSuffix(n, ".rego") || strings.HasSuffix(n, "_test.rego") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, n)) //nolint:gosec // chemin du dépôt
		if err != nil {
			t.Fatalf("lecture de %s : %v", n, err)
		}
		byCode[strings.TrimSuffix(n, ".rego")] = string(b)
	}

	for code, parType := range absenceMustBeAttested {
		src, ok := byCode[code]
		if !ok {
			t.Errorf("%q est déclaré comme concluant depuis une absence, mais aucune règle ne porte ce nom", code)
			continue
		}
		// Un contrôle ne peut pas à la fois EXIGER un attribut et conclure de son
		// absence : les deux tables ensemble le rendraient muet.
		if _, exige := requiredAttr[code]; exige {
			t.Errorf("%q figure dans requiredAttr ET dans absenceMustBeAttested.\n"+
				"  Le verrou exigerait la présence de l'attribut, tandis que la règle conclut\n"+
				"  de son absence : le contrôle ne pourrait plus rien dire.", code)
		}
		if inconnus := absenceDeciderTypesAreKnown(code, parType); len(inconnus) > 0 {
			t.Errorf("%q déclare le(s) type(s) %v que ce contrôle ne lit pas", code, inconnus)
		}
		for _, attrs := range parType {
			for _, attr := range attrs {
				if !strings.Contains(src, `"`+attr+`"`) {
					t.Errorf("%q : l'attribut %q n'apparaît pas dans sa règle — la table décrit une règle qui n'existe plus",
						code, attr)
					continue
				}
				// La règle doit COMPARER cet attribut à un vide, sinon elle ne conclut
				// pas d'une absence et n'a rien à faire dans cette table.
				if !strings.Contains(src, `"`+attr+`", "") == ""`) &&
					!strings.Contains(src, `object.get(k.attributes, "`+attr+`", "")`) {
					t.Errorf("%q : la règle ne semble pas conclure de l'ABSENCE de %q.\n"+
						"  Cette table ne vaut que pour les règles dont l'écart naît d'un champ vide ou absent.",
						code, attr)
				}
			}
		}
	}
}

// TestNoRuleReadsProvenance — invariant 1 de l'ADR-0017, et la borne qui rend la
// révision de l'ADR-0007 acceptable.
//
// L'index parallèle existe pour que l'entrée des règles reste identique octet pour
// octet : une règle réécrite est une règle qui peut changer de verdict. La provenance
// se lit dans l'assessment, jamais dans le Rego. Cette garde le vérifie plutôt que de
// l'espérer — c'est précisément parce que l'invariant précédent n'était gardé qu'à
// moitié qu'il a pu devenir faux sans que personne ne le voie.
func TestNoRuleReadsProvenance(t *testing.T) {
	dir := filepath.Join("..", "commonrules", "rules")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("règles indisponibles : %v", err)
	}
	var lues int
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".rego") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // chemin du dépôt
		if err != nil {
			t.Fatalf("lecture de %s : %v", e.Name(), err)
		}
		lues++
		if strings.Contains(string(b), "provenance") {
			t.Errorf("%s lit `provenance`.\n"+
				"  L'ADR-0007 a choisi un index PARALLÈLE pour que l'entrée des règles ne bouge\n"+
				"  pas, et l'ADR-0017 n'a levé cette borne pour personne : la distinction\n"+
				"  « cherché » / « jamais cherché » se traite dans l'assessment, où les verdicts\n"+
				"  se statuent déjà.", e.Name())
		}
	}
	if lues == 0 {
		t.Fatal("aucune règle lue : la garde ne mesure rien")
	}
	t.Logf("%d fichier(s) de règles vérifié(s)", lues)
}
