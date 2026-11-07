package supplychain_test

import (
	"regexp"
	"strings"
	"testing"
)

func TestReleaseUsesDefaultKeylessCosignKeys(t *testing.T) {
	body := workflows(t)["release.yml"]
	commands := regexp.MustCompile(`(?m)^\s*(?:run:\s*)?cosign\s+sign(?:-blob)?\s+([^\n]+)`).FindAllStringSubmatch(body, -1)
	if len(commands) == 0 {
		t.Fatal("aucune signature de release exercée par la garde")
	}
	for _, command := range commands {
		if strings.Contains(command[1], "--key") || strings.Contains(command[1], "--signing-algorithm") {
			t.Fatal("la release ne doit pas substituer une clé ou un algorithme au défaut keyless P-256 de Cosign")
		}
	}
}
