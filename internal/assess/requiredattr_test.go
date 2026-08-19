package assess

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRequiredAttrGuardsExist : le contrat annoncé au-dessus de requiredAttr était jusqu'ici
// FICTIF (le test n'existait pas), et la table avait dérivé — des contrôles lisaient leur
// attribut décisif avec un défaut CONFORME sans être gatés, produisant des « pass » sur une
// donnée jamais collectée. Ce test rend la synchronisation vérifiable : pour chaque entrée,
// le code du contrôle ET au moins un de ses attributs doivent apparaître dans les règles.
func TestRequiredAttrGuardsExist(t *testing.T) {
	dir := filepath.Join("..", "commonrules", "rules")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("règles indisponibles : %v", err)
	}
	var corpus strings.Builder
	byFile := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".rego") || strings.HasSuffix(e.Name(), "_test.rego") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("lecture %s : %v", e.Name(), err)
		}
		byFile[e.Name()] = string(b)
		corpus.WriteString(string(b))
	}
	all := corpus.String()

	for code, attrs := range requiredAttr {
		if !strings.Contains(all, `"`+code+`"`) {
			t.Errorf("requiredAttr référence %q : aucune règle n'émet ce code (entrée périmée)", code)
			continue
		}
		// L'attribut gaté doit être lu par le fichier qui émet ce code — ou par un
		// helper partagé de lib.rego qu'il appelle. Une règle a le droit de déléguer
		// la lecture d'un attribut (volume_in_use lit `state` pour le compte de
		// blockstorage_volume_snapshots_exist) : ne regarder que le fichier de la
		// règle ferait rejeter un gate pourtant légitime.
		var src string
		for name, body := range byFile {
			if name == "lib.rego" {
				continue
			}
			if strings.Contains(body, `"`+code+`"`) {
				src = body + byFile["lib.rego"]
				break
			}
		}
		found := false
		for _, a := range attrs {
			if strings.Contains(src, `"`+a+`"`) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("contrôle %q : aucun de ses attributs gatés %v n'est lu par sa règle — le gate ne protège rien", code, attrs)
		}
	}
}
