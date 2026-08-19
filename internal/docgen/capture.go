package docgen

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Capture est le résultat OBSERVÉ d'une exécution de `pepin` : la ligne de commande telle
// qu'un lecteur la retapera, les deux flux, et le code de sortie. Rien de ce qui apparaît
// dans docs/ sous forme de sortie ne vient d'ailleurs que d'ici.
type Capture struct {
	Args   []string
	Stdout string
	Stderr string
	Exit   int
}

// Command rend la commande telle qu'elle est écrite dans la documentation.
func (c Capture) Command() string { return "./pepin " + strings.Join(c.Args, " ") }

// Runner exécute le binaire Pépin depuis la racine du dépôt, dans un environnement neutralisé
// (ni couleur, ni terminal), seule façon d'obtenir une sortie reproductible d'une machine à
// l'autre.
type Runner struct {
	Bin  string // chemin du binaire pepin
	Root string // racine du dépôt (répertoire de travail des commandes)
}

// Run exécute `pepin <args>` et retourne la capture. Une erreur n'est remontée que si le
// processus n'a pas pu être lancé : un code de sortie non nul est un RÉSULTAT attendu ici
// (c'est même ce que la documentation explique).
func (r Runner) Run(args ...string) (Capture, error) {
	cmd := exec.Command(r.Bin, args...) // #nosec G204 -- binaire et arguments fixés par le paquet, jamais par une entrée externe.
	cmd.Dir = r.Root
	// Environnement minimal et explicite : NO_COLOR retire les séquences ANSI, TERM=dumb
	// évite toute adaptation au terminal, LC_ALL fige le tri et le formatage.
	cmd.Env = []string{"NO_COLOR=1", "TERM=dumb", "LC_ALL=C.UTF-8", "HOME=" + os.TempDir()}
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			return Capture{}, fmt.Errorf("exécution de %s %s : %w", r.Bin, strings.Join(args, " "), err)
		}
		code = ee.ExitCode()
	}
	return Capture{Args: args, Stdout: stdout.String(), Stderr: stderr.String(), Exit: code}, nil
}

// ResolveBinary trouve le binaire à interroger : PEPIN_BIN s'il est posé, sinon ./pepin à la
// racine (celui que `mise run build` produit, et que la CI construit avant les tests), sinon
// une compilation jetable. Un binaire absent ne doit pas rendre la documentation
// « inchangée » : ce serait un contrôle qui se déclare vert parce qu'il n'a rien regardé.
func ResolveBinary(root, tmpDir string) (string, error) {
	if bin := os.Getenv("PEPIN_BIN"); bin != "" {
		return bin, nil
	}
	built := filepath.Join(root, "pepin")
	if st, err := os.Stat(built); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
		return built, nil
	}
	out := filepath.Join(tmpDir, "pepin")
	cmd := exec.Command("go", "build", "-o", out, ".") // #nosec G204 -- arguments constants.
	cmd.Dir = root
	if b, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("compilation du binaire de capture : %w\n%s", err, b)
	}
	return out, nil
}

// Fence enveloppe une sortie capturée dans un bloc de code Markdown. Les sorties de Pépin ne
// contiennent pas de clôture de bloc, mais on le vérifie plutôt que de le supposer.
func Fence(lang, body string) string {
	body = strings.TrimRight(body, "\n")
	if strings.Contains(body, "\n```") {
		body = strings.ReplaceAll(body, "\n```", "\n'''")
	}
	return "```" + lang + "\n" + body + "\n```"
}

// Head rend les n premières lignes d'une sortie, suivies d'un marqueur d'élision explicite :
// une sortie tronquée doit se voir comme tronquée.
func Head(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[:n], "\n") + "\n[…]"
}

// Tail rend les n dernières lignes d'une sortie, précédées du marqueur d'élision.
func Tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}
	return "[…]\n" + strings.Join(lines[len(lines)-n:], "\n")
}
