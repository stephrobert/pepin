package i18n

import "testing"

// env construit un getenv de test à partir d'une carte.
func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

// TestParseNormalisesLocales : la forme POSIX complète doit donner la langue, pas
// une comparaison littérale. `LANG=fr_FR.UTF-8` est le cas nominal d'un poste
// francophone : le rater, c'est servir de l'anglais à toute la France.
func TestParseNormalisesLocales(t *testing.T) {
	cases := map[string]Lang{
		"fr":              FR,
		"FR":              FR,
		"fr_FR.UTF-8":     FR,
		"fr_BE@euro":      FR,
		"fr-CA":           FR,
		"  fr_FR.UTF-8  ": FR,
		"en":              EN,
		"en_US.UTF-8":     EN,
		"de_DE.UTF-8":     EN,
		"C.UTF-8":         EN,
		"POSIX":           EN,
		"":                EN,
		"français":        EN, // valeur inconnue : repli, jamais d'erreur
	}
	for in, want := range cases {
		if got := Parse(in); got != want {
			t.Errorf("Parse(%q) = %q, attendu %q", in, got, want)
		}
	}
}

// TestResolveOrderOfPrecedence : --lang l'emporte sur tout, puis PEPIN_LANG,
// puis LC_ALL, puis LANG, et le repli est l'anglais.
func TestResolveOrderOfPrecedence(t *testing.T) {
	cases := []struct {
		name string
		flag string
		env  map[string]string
		want Lang
	}{
		{"repli sans rien", "", nil, EN},
		{"LANG seul", "", map[string]string{"LANG": "fr_FR.UTF-8"}, FR},
		{"LC_ALL bat LANG", "", map[string]string{"LC_ALL": "en_US.UTF-8", "LANG": "fr_FR.UTF-8"}, EN},
		{"LC_ALL francophone", "", map[string]string{"LC_ALL": "fr_FR.UTF-8", "LANG": "en_US.UTF-8"}, FR},
		{"PEPIN_LANG bat LC_ALL", "", map[string]string{"PEPIN_LANG": "fr", "LC_ALL": "en_US.UTF-8"}, FR},
		{"--lang bat PEPIN_LANG", "en", map[string]string{"PEPIN_LANG": "fr", "LANG": "fr_FR.UTF-8"}, EN},
		{"--lang fr sur env anglais", "fr", map[string]string{"LC_ALL": "en_US.UTF-8"}, FR},
		{"--lang inconnu retombe sur en", "klingon", map[string]string{"LANG": "fr_FR.UTF-8"}, EN},
		{"variable vide : source suivante", "", map[string]string{"PEPIN_LANG": "", "LC_ALL": "  ", "LANG": "fr_FR.UTF-8"}, FR},
		// LC_ALL posé volontairement sur une locale inconnue N'EST PAS contourné
		// par un LANG résiduel : c'est la règle POSIX, et la seule qui ne surprenne pas.
		{"LC_ALL inconnu ne retombe pas sur LANG", "", map[string]string{"LC_ALL": "de_DE.UTF-8", "LANG": "fr_FR.UTF-8"}, EN},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Resolve(c.flag, env(c.env)); got != c.want {
				t.Errorf("Resolve(%q, %v) = %q, attendu %q", c.flag, c.env, got, c.want)
			}
		})
	}
}

// TestResolveWithoutEnvReader : un getenv nil ne doit pas paniquer (chemin des
// tests et des appels programmatiques).
func TestResolveWithoutEnvReader(t *testing.T) {
	if got := Resolve("", nil); got != EN {
		t.Errorf("Resolve(\"\", nil) = %q, attendu %q", got, EN)
	}
	if got := Resolve("fr", nil); got != FR {
		t.Errorf("Resolve(\"fr\", nil) = %q, attendu %q", got, FR)
	}
}

// TestCurrentDefaultsToEnglish : sans Set, la langue est l'anglais, jamais une
// valeur vide qui traverserait le rendu.
func TestCurrentDefaultsToEnglish(t *testing.T) {
	defer Set(Current())
	Set(Lang("")) // valeur illégale : normalisée
	if got := Current(); got != EN {
		t.Errorf("Current() après Set(\"\") = %q, attendu %q", got, EN)
	}
	Set(Lang("de"))
	if got := Current(); got != EN {
		t.Errorf("Current() après Set(\"de\") = %q, attendu %q", got, EN)
	}
	Set(FR)
	if got := Current(); got != FR {
		t.Errorf("Current() après Set(FR) = %q, attendu %q", got, FR)
	}
}

func TestTAndTIn(t *testing.T) {
	defer Set(Current())
	Set(FR)
	if got := T("chien", "dog"); got != "chien" {
		t.Errorf("T en français = %q", got)
	}
	Set(EN)
	if got := T("chien", "dog"); got != "dog" {
		t.Errorf("T en anglais = %q", got)
	}
	if got := TIn(FR, "chien", "dog"); got != "chien" {
		t.Errorf("TIn(FR) = %q", got)
	}
}

// TestPickDegradesToFrench : la traduction d'une DONNÉE peut manquer (règle
// externe --policy-dir, contrôle non traduit) ; le rendu doit alors montrer le
// français, jamais du vide. L'absence est refusée en CI, pas au runtime.
func TestPickDegradesToFrench(t *testing.T) {
	if got := PickIn(EN, "message français", ""); got != "message français" {
		t.Errorf("PickIn(EN, fr, \"\") = %q, attendu le repli français", got)
	}
	if got := PickIn(EN, "message français", "   "); got != "message français" {
		t.Errorf("PickIn(EN, fr, blancs) = %q, attendu le repli français", got)
	}
	if got := PickIn(EN, "message français", "English message"); got != "English message" {
		t.Errorf("PickIn(EN, fr, en) = %q", got)
	}
	if got := PickIn(FR, "message français", "English message"); got != "message français" {
		t.Errorf("PickIn(FR, …) = %q", got)
	}
}
