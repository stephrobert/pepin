package policy

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/referentiel"
)

// evaluable indique que le moteur sait lire ce réglage, sous l'une des trois
// formes qu'une contrainte peut comparer : un ordinal, un ensemble, ou un jeu
// d'exigences d'étiquetage.
func evaluable(r Resolved, param string) bool {
	if _, ok := ordinal(r, param); ok {
		return true
	}
	if _, ok := members(r, param); ok {
		return true
	}
	_, ok := requirements(r, param)
	return ok
}

// write écrit un fichier de politique jetable et rend son chemin.
func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "policy.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("écriture de la politique de test : %v", err)
	}
	return p
}

// TestEveryDeclaredParameterIsEvaluated est la garde ANTI-DÉRIVE entre les deux
// moitiés de l'issue #65 : le référentiel DÉCLARE les réglages qu'une contrainte
// peut nommer, le moteur les ÉVALUE. Un nom déclaré que le moteur ignore ne
// protégerait rien — la contrainte serait écrite, jamais appliquée —, et un
// réglage évalué que le référentiel ne nomme pas ne pourrait jamais faire tomber
// une correspondance. Les deux listes doivent donc coïncider exactement.
func TestEveryDeclaredParameterIsEvaluated(t *testing.T) {
	def := Defaults()
	for _, p := range referentiel.ConfigParameters {
		if !evaluable(def, p) {
			t.Errorf("le référentiel déclare le réglage %q, que le moteur de politique ne sait pas "+
				"évaluer : la contrainte serait écrite et jamais appliquée. Ajouter le cas dans "+
				"internal/policy/relaxation.go (ordinal ou members).", p)
		}
		if got := render(def, p); strings.Contains(got, "réglage inconnu") || strings.Contains(got, "unknown setting") {
			t.Errorf("le réglage %q n'a pas de rendu lisible : le rapport ne pourrait pas dire "+
				"ce qui a changé", p)
		}
	}
}

// TestNoEvaluatedParameterIsUndeclared est l'autre sens de la même garde.
func TestNoEvaluatedParameterIsUndeclared(t *testing.T) {
	declared := map[string]bool{}
	for _, p := range referentiel.ConfigParameters {
		declared[p] = true
	}
	// Les réglages que le moteur sait évaluer, énumérés à la main pour que l'ajout
	// d'un cas dans ordinal/members sans sa déclaration au référentiel se voie ici.
	evaluated := []string{
		"tagging.required_tags",
		"tagging.network_required_tags",
		"tagging.resource_types",
		"snapshots.max_age_days",
		"snapshots.accepted_states",
		"secrets.min_confidence",
	}
	def := Defaults()
	for _, p := range evaluated {
		if !evaluable(def, p) {
			t.Fatalf("le test énumère %q comme évaluable alors que le moteur l'ignore", p)
		}
		if !declared[p] {
			t.Errorf("le moteur évalue le réglage %q, qu'aucune `config_requise` ne peut nommer : "+
				"le desserrer ne ferait tomber aucune correspondance normative. L'ajouter à "+
				"referentiel.ConfigParameters et à la `config_requise` du contrôle qui le lit.", p)
		}
	}
}

// TestAnEmptyPolicyIsTheDefaultProfile : une section absente ne change rien. C'est
// la propriété qui garantit qu'un fichier de politique partiel n'assouplit que ce
// qu'il nomme.
func TestAnEmptyPolicyIsTheDefaultProfile(t *testing.T) {
	if got := Resolve(nil); !reflect.DeepEqual(got, Defaults()) {
		t.Errorf("une politique absente doit rendre le profil par défaut\n  attendu : %+v\n  obtenu  : %+v", Defaults(), got)
	}
	if got := Resolve(&Controls{}); !reflect.DeepEqual(got, Defaults()) {
		t.Errorf("une section `controls:` vide doit rendre le profil par défaut")
	}
}

// TestAnExplicitDefaultPolicyRelaxesNothing : écrire les valeurs par défaut n'est
// pas un assouplissement. Sans cette propriété, un utilisateur qui documente sa
// configuration verrait son rapport perdre ses correspondances normatives pour
// n'avoir rien changé.
func TestAnExplicitDefaultPolicyRelaxesNothing(t *testing.T) {
	seven := 7
	res := Resolve(&Controls{
		Tagging: &Tagging{
			RequiredTags:        defaultBillableTags,
			NetworkRequiredTags: defaultNetworkTags,
			Aliases:             defaultTagAliases,
			ResourceTypes:       defaultTaggedTypes,
		},
		Snapshots: &Snapshots{MaxAgeDays: &seven, AcceptedStates: defaultSnapshotStates},
		Secrets:   &Secrets{MinConfidence: "low"},
	})
	if !reflect.DeepEqual(res, Defaults()) {
		t.Fatalf("le profil par défaut écrit explicitement doit être identique au profil implicite\n  attendu : %+v\n  obtenu  : %+v", Defaults(), res)
	}
	if got := Relaxations(res, referentiel.ConfigConstraintsByControl(), nil); len(got) != 0 {
		t.Errorf("le profil par défaut ne doit produire AUCUN assouplissement, obtenu : %+v", got)
	}
}

// TestTighteningIsNotARelaxation : la contrainte ne dit pas « ne touche à rien »,
// elle dit de quel côté du défaut la promesse survit. Durcir doit rester
// silencieux, sinon la seule configuration confortable serait le défaut, et la
// configurabilité n'aurait servi à personne.
func TestTighteningIsNotARelaxation(t *testing.T) {
	one := 1
	res := Resolve(&Controls{
		Tagging:   &Tagging{RequiredTags: append(append([]string{}, defaultBillableTags...), "data-classification")},
		Snapshots: &Snapshots{MaxAgeDays: &one, AcceptedStates: []string{"completed"}},
		Secrets:   &Secrets{MinConfidence: "low"},
	})
	if got := Relaxations(res, referentiel.ConfigConstraintsByControl(), nil); len(got) != 0 {
		for code, rs := range got {
			for _, r := range rs {
				t.Errorf("durcissement signalé à tort : %s / %s (%s → %s)", code, r.Parameter, r.Default, r.Effective)
			}
		}
	}
}

// TestAnotherWritingConventionIsNotARelaxation : le faux positif que l'issue #61
// décrit, appliqué à la contrainte elle-même. Une organisation qui écrit
// `cost-center, project, environment, owner` exige MOINS d'écritures que le profil
// par défaut (qui accepte aussi `cc`, `app`, `env`, `team`…) : elle a resserré sa
// convention, pas desserré son exigence. La signaler comme assouplie lui ferait
// perdre sa correspondance normative pour avoir bien fait — le faux positif le plus
// coûteux qui soit, puisqu'il punit le bon comportement.
func TestAnotherWritingConventionIsNotARelaxation(t *testing.T) {
	res := Resolve(&Controls{Tagging: &Tagging{
		RequiredTags:        []string{"cost-center", "project", "environment", "owner"},
		NetworkRequiredTags: []string{"owner", "project", "environment"},
	}})
	if got := Relaxations(res, referentiel.ConfigConstraintsByControl(), nil); len(got) != 0 {
		for code, rs := range got {
			for _, r := range rs {
				t.Errorf("convention d'écriture signalée comme assouplissement : %s / %s (%s → %s)",
					code, r.Parameter, r.Default, r.Effective)
			}
		}
	}
}

// TestRelaxationsAreDetected éprouve chacun des sens de contrainte, un par un, sur
// le réglage qui le porte réellement.
func TestRelaxationsAreDetected(t *testing.T) {
	thirty := 30
	cases := []struct {
		name    string
		ctrl    *Controls
		control string
		param   string
	}{
		{
			name:    "délai de fraîcheur allongé (au_plus_le_defaut)",
			ctrl:    &Controls{Snapshots: &Snapshots{MaxAgeDays: &thirty}},
			control: "blockstorage_volume_snapshots_exist",
			param:   "snapshots.max_age_days",
		},
		{
			name:    "états acceptés élargis (sous_ensemble_du_defaut)",
			ctrl:    &Controls{Snapshots: &Snapshots{AcceptedStates: []string{"completed", "created", "creating"}}},
			control: "blockstorage_volume_snapshots_exist",
			param:   "snapshots.accepted_states",
		},
		{
			name:    "étiquette retirée du profil (au_moins_aussi_strict_que_le_defaut)",
			ctrl:    &Controls{Tagging: &Tagging{RequiredTags: []string{"Owner", "Project", "Env"}}},
			control: "governance_resource_required_tags",
			param:   "tagging.required_tags",
		},
		{
			name:    "seuil de confiance monté (au_plus_le_defaut)",
			ctrl:    &Controls{Secrets: &Secrets{MinConfidence: "high"}},
			control: "compute_instance_no_secrets_in_user_data",
			param:   "secrets.min_confidence",
		},
		{
			name: "alias élargi (au_moins_aussi_strict_que_le_defaut)",
			ctrl: &Controls{Tagging: &Tagging{Aliases: map[string][]string{
				"CostCenter": {"CostCenter", "cost-center", "cc", "billing-code", "billing", "n-importe-quoi"},
				"Project":    {"Project", "app", "application", "service"},
				"Env":        {"Env", "environment", "stage"},
				"Owner":      {"Owner", "team", "responsible", "contact"},
			}}},
			control: "governance_resource_required_tags",
			param:   "tagging.required_tags",
		},
		{
			name:    "périmètre de types rétréci (superset_du_defaut)",
			ctrl:    &Controls{Tagging: &Tagging{ResourceTypes: []string{"compute_instance"}}},
			control: "governance_resource_required_tags",
			param:   "tagging.resource_types",
		},
	}
	constraints := referentiel.ConfigConstraintsByControl()
	refs := map[string][]string{
		"blockstorage_volume_snapshots_exist":      {"scsl:CLD-STO-3"},
		"governance_resource_required_tags":        {"scsl:CLD-GVN-1"},
		"compute_instance_no_secrets_in_user_data": {"scsl:CLD-CMP-9"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Relaxations(Resolve(tc.ctrl), constraints, refs)
			found := false
			for _, r := range got[tc.control] {
				if r.Parameter == tc.param {
					found = true
					if len(r.DroppedReferences) == 0 {
						t.Errorf("l'assouplissement doit nommer les correspondances abandonnées")
					}
					if r.Default == r.Effective {
						t.Errorf("l'assouplissement doit rendre deux valeurs distinctes, obtenu %q des deux côtés", r.Default)
					}
					if s := r.Sentence(); !strings.Contains(s, tc.param) {
						t.Errorf("la phrase doit nommer le réglage : %q", s)
					}
				}
			}
			if !found {
				t.Errorf("assouplissement non détecté sur %s / %s ; obtenu : %+v", tc.control, tc.param, got)
			}
		})
	}
}

// TestTagComparisonIgnoresCaseAndSeparators : `cost-center` ≡ `CostCenter`
// (critère d'acceptation de l'issue #61).
func TestTagComparisonIgnoresCaseAndSeparators(t *testing.T) {
	for _, form := range []string{"CostCenter", "cost-center", "cost_center", "Cost Center", "COST.CENTER"} {
		if got := normalizeTagKey(form); got != "costcenter" {
			t.Errorf("normalizeTagKey(%q) = %q, attendu \"costcenter\"", form, got)
		}
	}
	// Et la normalisation ne fusionne pas deux noms réellement distincts.
	if normalizeTagKey("owner") == normalizeTagKey("ownership") {
		t.Error("la normalisation ne doit pas confondre deux noms distincts")
	}
}

// TestLoadRefusesAnUnreadablePolicy : une valeur qu'on ne sait pas interpréter
// arrête le scan, elle ne s'applique pas à moitié.
func TestLoadRefusesAnUnreadablePolicy(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"fichier vide", "controls:\n", "exceptions"},
		{"seuil inconnu", "controls:\n  secrets:\n    min_confidence: paranoid\n", "min_confidence"},
		{"délai nul", "controls:\n  snapshots:\n    max_age_days: 0\n", "max_age_days"},
		{"délai négatif", "controls:\n  snapshots:\n    max_age_days: -3\n", "max_age_days"},
		{"étiquette vide", "controls:\n  tagging:\n    required_tags: [\"\"]\n", "required_tags"},
		{"dérogation sans date", "exceptions:\n  - control: c\n    justification: j\n    owner: o\n    approved_by: a\n", "expires_at"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(write(t, tc.body))
			if err == nil {
				t.Fatalf("politique invalide acceptée")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("le message doit nommer ce qui cloche (%q) ; obtenu : %v", tc.want, err)
			}
		})
	}
}

// TestLoadAcceptsAnExemptionsOnlyFile : un fichier `--exceptions` existant est un
// fichier de politique valide. Sans cette propriété, le passage à un fichier
// unique casserait toutes les invocations en place.
func TestLoadAcceptsAnExemptionsOnlyFile(t *testing.T) {
	f, err := Load(write(t, "exceptions:\n  - control: objectstorage_bucket_public_access\n"+
		"    justification: bucket public assumé, contenu déjà public\n"+
		"    expires_at: 2099-12-31\n    owner: sre\n    approved_by: rssi\n"))
	if err != nil {
		t.Fatalf("fichier de dérogations refusé : %v", err)
	}
	if len(f.Exemptions().Exemptions) != 1 {
		t.Errorf("la dérogation doit être lue")
	}
	if !reflect.DeepEqual(Resolve(f.Controls), Defaults()) {
		t.Errorf("un fichier sans section `controls:` doit laisser le profil par défaut intact")
	}
}

// TestLoadReadsBothSections : les deux sections coexistent dans le même fichier.
func TestLoadReadsBothSections(t *testing.T) {
	f, err := Load(write(t, "controls:\n  snapshots:\n    max_age_days: 14\n"+
		"exceptions:\n  - control: c\n    justification: j\n"+
		"    expires_at: 2099-01-01\n    owner: o\n    approved_by: a\n"))
	if err != nil {
		t.Fatalf("politique à deux sections refusée : %v", err)
	}
	if len(f.Exemptions().Exemptions) != 1 {
		t.Errorf("la section `exceptions:` doit être lue")
	}
	if got := Resolve(f.Controls).Snapshots.MaxAgeDays; got != 14 {
		t.Errorf("la section `controls:` doit être lue, max_age_days = %d", got)
	}
}

// TestDigestSeparatesTwoConfigurations : deux politiques différentes ne peuvent
// pas porter la même empreinte sous le même résultat.
func TestDigestSeparatesTwoConfigurations(t *testing.T) {
	thirty := 30
	a := Defaults().Digest()
	b := Resolve(&Controls{Snapshots: &Snapshots{MaxAgeDays: &thirty}}).Digest()
	if a == "" || a == b {
		t.Errorf("deux configurations distinctes doivent avoir deux empreintes distinctes (%q vs %q)", a, b)
	}
	if a != Defaults().Digest() {
		t.Error("l'empreinte doit être stable pour une même configuration")
	}
}
