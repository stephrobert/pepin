// Commande docgen régénère la documentation dérivée : la matrice de couverture
// (docs/coverage.md et sa version française) et toutes les régions générées des pages
// écrites à la main (sorties de commande réelles, tableaux tirés du référentiel).
//
// Usage :
//
//	mise run gen-docs             # depuis la racine du dépôt
//	go run ./tools/docgen -root . # équivalent
//
// La vérification, elle, n'est pas un drapeau de cette commande : elle est un TEST
// (internal/docgen.TestGeneratedDocsAreUpToDate), pour qu'une documentation périmée fasse
// échouer `mise run test` et donc la CI, sans dépendre d'une étape qu'on peut oublier.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stephrobert/pepin/internal/docgen"
)

func main() {
	root := flag.String("root", ".", "racine du dépôt")
	bin := flag.String("bin", "", "binaire pepin à interroger (défaut : $PEPIN_BIN, ./pepin, ou compilation jetable)")
	flag.Parse()

	abs, err := filepath.Abs(*root)
	if err != nil {
		fail(err)
	}
	tmp, err := os.MkdirTemp("", "pepin-docgen-bin-")
	if err != nil {
		fail(err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	resolved := *bin
	if resolved == "" {
		resolved, err = docgen.ResolveBinary(abs, tmp)
		if err != nil {
			fail(err)
		}
	}
	changed, err := docgen.Write(abs, resolved)
	if err != nil {
		fail(err)
	}
	if len(changed) == 0 {
		fmt.Println("docgen : documentation déjà à jour")
		return
	}
	for _, p := range changed {
		fmt.Println("docgen : régénéré " + p)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "docgen : "+err.Error())
	os.Exit(1)
}
