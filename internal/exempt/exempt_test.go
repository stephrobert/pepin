package exempt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stephrobert/scankit/assessment"
)

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "exceptions.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("écriture de la fixture : %v", err)
	}
	return path
}

const validFile = `exceptions:
  - control: network_securitygroup_allow_ingress_from_internet_to_tcp_port_22
    resource: vm-bastion
    justification: "Bastion administré, accès restreint par IP source"
    expires_at: 2026-12-31
    owner: platform-security
    approved_by: security@example.org
`

func at(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("date de test %q : %v", s, err)
	}
	return v
}

func failing(control, subject string) assessment.Assessment {
	return assessment.Assessment{Results: []assessment.Result{
		{Control: control, Subject: subject, Status: assessment.Fail, Severity: "critical"},
	}}
}

func policyOf(t *testing.T, body string) Policy {
	t.Helper()
	pol, err := Load(write(t, body))
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}
	return pol
}

const sshControl = "network_securitygroup_allow_ingress_from_internet_to_tcp_port_22"

var knownSSH = map[string]bool{sshControl: true}
var subjectBastion = map[string]bool{"vm-bastion": true}

// TestAnExemptionNeverTurnsAFailIntoAPass est L'INVARIANT de ce lot, et il est
// écrit pour échouer si le statut produit devenait conforme de quelque manière que
// ce soit : par la valeur du statut, par le décompte, ou par la lecture que scankit
// en fait (Conformant()).
//
// Un contrôle exempté n'est pas conforme. Il est écarté, sciemment et de façon
// traçable. S'il pouvait compter comme conforme, on aurait donné un nom
// respectable au faux vert.
func TestAnExemptionNeverTurnsAFailIntoAPass(t *testing.T) {
	got, rep := Apply(failing(sshControl, "vm-bastion"), policyOf(t, validFile),
		at(t, "2026-08-19"), subjectBastion, knownSSH)

	r := got.Results[0]
	if r.Status == assessment.Pass {
		t.Fatal("une dérogation a produit un `pass` — c'est exactement le faux vert que le statut existe pour empêcher")
	}
	if r.Status != StatusExempted {
		t.Fatalf("statut %q, attendu %q", r.Status, StatusExempted)
	}
	// Le décompte par statut ne doit compter l'exempté nulle part ailleurs.
	sum := got.Summarize()
	if sum.ByStatus[assessment.Pass] != 0 {
		t.Errorf("%d résultat(s) comptés `pass` après exemption", sum.ByStatus[assessment.Pass])
	}
	if sum.ByStatus[StatusExempted] != 1 {
		t.Errorf("%d résultat(s) comptés `exempted`, attendu 1", sum.ByStatus[StatusExempted])
	}
	// La dérogation est TRAÇABLE : qui, pourquoi, jusqu'à quand.
	if r.Waiver == nil || r.Waiver.Justification == "" || r.Waiver.Until == "" {
		t.Fatalf("dérogation appliquée sans waiver traçable : %+v", r.Waiver)
	}
	for _, label := range []string{"exemption_owner", "exemption_approved_by", "exemption_expires_at"} {
		if r.Labels[label] == "" {
			t.Errorf("label %q absent : une dérogation sans responsable n'engage personne", label)
		}
	}
	if !rep.Applied() {
		t.Error("l'effet `applied` n'est pas rapporté")
	}
}

// TestAnExpiredExemptionStopsApplyingAndSaysSo : sans date, une dérogation devient
// permanente par oubli. L'expiration la retire du jeu, et le dit.
func TestAnExpiredExemptionStopsApplyingAndSaysSo(t *testing.T) {
	got, rep := Apply(failing(sshControl, "vm-bastion"), policyOf(t, validFile),
		at(t, "2027-01-02"), subjectBastion, knownSSH)

	if got.Results[0].Status != assessment.Fail {
		t.Errorf("statut %q après expiration, attendu `fail` — l'écart redevient un écart", got.Results[0].Status)
	}
	if rep.Applied() {
		t.Error("une dérogation expirée compte comme appliquée")
	}
	if rep.Count(EffectExpired) != 1 {
		t.Fatalf("%d dérogation(s) rapportée(s) expirée(s), attendu 1", rep.Count(EffectExpired))
	}
	notices := rep.Notices()
	if len(notices) != 1 || !strings.Contains(strings.ToUpper(notices[0]), "EXPIR") {
		t.Errorf("l'expiration ne se dit pas : %v", notices)
	}
}

// TestTheLastDayOfValidityIsStillCovered : « jusqu'au 31 décembre » couvre le
// 31 décembre. Une borne qui exclut son dernier jour surprend tout le monde.
func TestTheLastDayOfValidityIsStillCovered(t *testing.T) {
	got, _ := Apply(failing(sshControl, "vm-bastion"), policyOf(t, validFile),
		at(t, "2026-12-31"), subjectBastion, knownSSH)
	if got.Results[0].Status != StatusExempted {
		t.Errorf("statut %q le jour même de l'échéance, attendu `exempted`", got.Results[0].Status)
	}
}

// TestAnOrphanExemptionIsReported : une dérogation qui ne correspond à aucun
// contrôle ni à aucune ressource est le symptôme d'une exception oubliée. On la
// signale ; on ne l'ignore jamais.
func TestAnOrphanExemptionIsReported(t *testing.T) {
	cases := map[string]struct {
		body     string
		subjects map[string]bool
		controls map[string]bool
	}{
		"contrôle inconnu": {
			body:     strings.Replace(validFile, sshControl, "controle_qui_nexiste_plus", 1),
			subjects: subjectBastion, controls: knownSSH,
		},
		"ressource inconnue": {
			body: validFile, subjects: map[string]bool{"vm-autre": true}, controls: knownSSH,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got, rep := Apply(failing(sshControl, "vm-bastion"), policyOf(t, c.body),
				at(t, "2026-08-19"), c.subjects, c.controls)
			if got.Results[0].Status != assessment.Fail {
				t.Errorf("statut %q, attendu `fail` — une dérogation orpheline n'écarte rien", got.Results[0].Status)
			}
			if rep.Count(EffectOrphan) != 1 {
				t.Fatalf("%d orpheline(s) rapportée(s), attendu 1", rep.Count(EffectOrphan))
			}
			if len(rep.Notices()) != 1 {
				t.Errorf("l'orpheline ne se dit pas : %v", rep.Notices())
			}
			if !rep.Stale() {
				t.Error("un fichier portant une orpheline n'est pas signalé comme à revoir")
			}
		})
	}
}

// TestOnlyAFailCanBeExempted : on ne déroge pas à un contrôle qu'on n'a pas su
// mesurer. Exempter un `not-evaluated` reviendrait à cacher une absence de mesure
// derrière une exception — le faux vert avec une étape de plus.
func TestOnlyAFailCanBeExempted(t *testing.T) {
	a := assessment.Assessment{Results: []assessment.Result{
		{Control: sshControl, Subject: "vm-bastion", Status: assessment.NotEvaluated},
		{Control: sshControl, Subject: "vm-bastion", Status: assessment.Pass},
		{Control: sshControl, Subject: "vm-bastion", Status: assessment.NotApplicable},
	}}
	got, rep := Apply(a, policyOf(t, validFile), at(t, "2026-08-19"), subjectBastion, knownSSH)
	for i, want := range []assessment.Status{assessment.NotEvaluated, assessment.Pass, assessment.NotApplicable} {
		if got.Results[i].Status != want {
			t.Errorf("statut %q déplacé en %q par une dérogation", want, got.Results[i].Status)
		}
	}
	if rep.Applied() {
		t.Error("une dérogation sans écart à écarter se dit appliquée")
	}
}

// TestMandatoryFieldsAreValidatedAtLoadTime : les champs obligatoires se vérifient
// au CHARGEMENT. Un fichier incomplet arrête le scan, il ne produit pas un rapport
// qui tait la moitié de ses exceptions.
func TestMandatoryFieldsAreValidatedAtLoadTime(t *testing.T) {
	for name, body := range map[string]string{
		"sans justification": strings.Replace(validFile, `    justification: "Bastion administré, accès restreint par IP source"`+"\n", "", 1),
		"sans date":          strings.Replace(validFile, "    expires_at: 2026-12-31\n", "", 1),
		"sans responsable":   strings.Replace(validFile, "    owner: platform-security\n", "", 1),
		"sans approbateur":   strings.Replace(validFile, "    approved_by: security@example.org\n", "", 1),
		"sans contrôle":      strings.Replace(validFile, "  - control: "+sshControl+"\n", "  - resource: x\n", 1),
		"date illisible":     strings.Replace(validFile, "2026-12-31", "le mois prochain", 1),
		"fichier vide":       "exceptions: []\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(write(t, body)); err == nil {
				t.Error("chargement accepté — un champ obligatoire manquant doit arrêter le scan, pas l'appliquer à moitié")
			}
		})
	}
}

// TestLoadAcceptsBothDateForms : la date seule est la forme naturelle d'une revue,
// RFC3339 celle d'un outil. Les deux se chargent.
func TestLoadAcceptsBothDateForms(t *testing.T) {
	for _, form := range []string{"2026-12-31", `"2026-12-31T23:59:59Z"`} {
		if _, err := Load(write(t, strings.Replace(validFile, "2026-12-31", form, 1))); err != nil {
			t.Errorf("forme de date %s refusée : %v", form, err)
		}
	}
}

// TestAnExemptionWithoutResourceCoversEverySubject : une entrée sans `resource`
// porte sur tout le contrôle. Portée large, donc à assumer — mais elle doit
// fonctionner, sinon les équipes écrivent une entrée par ressource et la revue
// devient impraticable.
func TestAnExemptionWithoutResourceCoversEverySubject(t *testing.T) {
	body := strings.Replace(validFile, "    resource: vm-bastion\n", "", 1)
	a := assessment.Assessment{Results: []assessment.Result{
		{Control: sshControl, Subject: "vm-a", Status: assessment.Fail},
		{Control: sshControl, Subject: "vm-b", Status: assessment.Fail},
	}}
	got, rep := Apply(a, policyOf(t, body), at(t, "2026-08-19"), map[string]bool{}, knownSSH)
	for _, r := range got.Results {
		if r.Status != StatusExempted {
			t.Errorf("%s : statut %q, attendu `exempted`", r.Subject, r.Status)
		}
	}
	if n := len(rep.Records[0].Subjects); n != 2 {
		t.Errorf("%d sujet(s) rapporté(s) écarté(s), attendu 2", n)
	}
}

// TestTheDigestFollowsTheContentNotTheFormatting : l'empreinte prouve QUELLE
// politique a été appliquée. Elle doit suivre le contenu, pas l'indentation.
func TestTheDigestFollowsTheContentNotTheFormatting(t *testing.T) {
	a := policyOf(t, validFile)
	reordered := policyOf(t, validFile+strings.Replace(validFile, "exceptions:\n", "", 1))
	if a.Digest() == reordered.Digest() {
		t.Error("deux politiques différentes portent la même empreinte")
	}
	// Les mêmes entrées, clés dans un autre ordre : même politique, même empreinte.
	const reformatted = `exceptions:
  - owner: platform-security
    approved_by: security@example.org
    expires_at: 2026-12-31
    resource: vm-bastion
    justification: "Bastion administré, accès restreint par IP source"
    control: ` + sshControl + `
`
	if a.Digest() != policyOf(t, reformatted).Digest() {
		t.Error("l'empreinte a bougé pour un simple réordonnancement des clés")
	}
}
