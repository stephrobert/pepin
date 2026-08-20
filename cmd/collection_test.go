package cmd

// Les critères d'acceptation de la complétude de collecte, mesurés sur le PRODUIT :
// on compile le binaire, on lui donne un inventaire qui déclare une unité de
// collecte incomplète, et on lit ce qu'il imprime et ce qu'il rend comme code.
//
// Pourquoi passer par un inventaire porteur de son état de collecte plutôt que par
// une fausse API : c'est EXACTEMENT le chemin qu'emprunte le rejeu d'un bundle
// scellé (`verify --re-derive` relit l'input.json, `collection` compris). Éprouver
// ce chemin éprouve donc les deux — le scan qui produit l'état, et le vérificateur
// qui doit en retrouver le même verdict. Les chemins de collecte eux-mêmes (403,
// timeout, pagination) sont mesurés contre un serveur httptest dans
// internal/collect.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cleanInventory : l'inventaire d'exemple SANS écart, celui qui rend 0. Le point de
// départ obligé — on ne peut prouver qu'une incomplétude retire un « conforme »
// qu'en partant d'un scan qui, sans elle, serait conforme.
const cleanInventory = "examples/scaleway/inventory-ok.json"

// withIncompleteUnit rend le chemin d'une copie de l'inventaire propre déclarant
// `security_group_rule` incomplet pour privilège insuffisant.
func withIncompleteUnit(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot, cleanInventory)) // #nosec G304 -- fixture du dépôt.
	if err != nil {
		t.Fatalf("lecture de %s : %v", cleanInventory, err)
	}
	var inv map[string]any
	if err := json.Unmarshal(raw, &inv); err != nil {
		t.Fatalf("inventaire d'exemple illisible : %v", err)
	}
	inv["collection"] = map[string]any{
		"units": []any{
			map[string]any{"unit": "object_storage_bucket", "types": []string{"object_storage_bucket"},
				"attempted": true, "complete": true},
			map[string]any{"unit": "security_group_rule", "types": []string{"security_group_rule"},
				"attempted": true, "complete": false, "error": "permission_denied",
				"detail": "HTTP 403 GET https://api.example/security-groups"},
		},
	}
	out, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		t.Fatalf("sérialisation de l'inventaire de test : %v", err)
	}
	path := filepath.Join(t.TempDir(), "inventory-incomplete.json")
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatalf("écriture de l'inventaire de test : %v", err)
	}
	return path
}

// TestAnIncompleteCollectionNeverProducesAZeroExitCode est le point qui décide de
// tout : « aucun écart » sur un périmètre partiellement lu n'est pas « conforme ».
func TestAnIncompleteCollectionNeverProducesAZeroExitCode(t *testing.T) {
	bin := buildPepin(t)

	// Témoin : le même inventaire, sans état de collecte, rend bien 0. Sans lui, le
	// test pourrait passer sur un inventaire qui n'était de toute façon pas conforme,
	// et ne mesurerait alors que lui-même.
	if _, _, code := runScan(t, bin, "scan", "scaleway", cleanInventory); code != exitConforme {
		t.Fatalf("l'inventaire témoin doit rendre %d, il rend %d — le test ne mesure plus rien", exitConforme, code)
	}

	_, _, code := runScan(t, bin, "scan", "scaleway", withIncompleteUnit(t))
	if code == exitConforme {
		t.Fatal("un scan dont une unité de collecte a échoué a rendu 0 — jamais 0 sur un périmètre non lu")
	}
	if code != exitStrict {
		t.Errorf("code de sortie %d, attendu %d (périmètre partiellement lu)", code, exitStrict)
	}
}

// TestAnIncompleteCollectionWithdrawsThePassesItCannotJustify : le code de sortie ne
// suffit pas, c'est le VERDICT par contrôle qui doit bouger. Les contrôles qui lisent
// le type non collecté perdent leur « pass » ; les autres le gardent.
func TestAnIncompleteCollectionWithdrawsThePassesItCannotJustify(t *testing.T) {
	bin := buildPepin(t)
	stdout, _, _ := runScan(t, bin, "scan", "scaleway", withIncompleteUnit(t), "--format", "assessment")

	var doc struct {
		Results []struct {
			Control  string `json:"control"`
			Status   string `json:"status"`
			Evidence struct {
				Observed string `json:"observed"`
			} `json:"evidence"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("assessment illisible : %v", err)
	}

	var degradedPass, keptPass bool
	for _, r := range doc.Results {
		switch r.Control {
		case "network_securitygroup_allow_ingress_from_internet_to_tcp_port_22":
			if r.Status != "not-evaluated" {
				t.Errorf("un contrôle qui lit le type non collecté doit être non évalué, obtenu %q", r.Status)
			}
			if !strings.Contains(r.Evidence.Observed, "security_group_rule") {
				t.Errorf("le motif doit NOMMER l'unité qui a manqué, obtenu %q", r.Evidence.Observed)
			}
			degradedPass = true
		case "objectstorage_bucket_public_access":
			// Le stockage objet a répondu : son verdict ne doit pas bouger, sinon un
			// seul droit manquant effacerait tout le reste de la mesure.
			if r.Status != "pass" {
				t.Errorf("un contrôle dont l'unité a répondu garde son verdict, obtenu %q", r.Status)
			}
			keptPass = true
		}
	}
	if !degradedPass || !keptPass {
		t.Fatal("les deux contrôles témoins doivent figurer dans l'assessment")
	}
}

// TestTheCapabilityReportSaysWhatCouldNotBeObserved : l'exigence de l'issue #45.
// Le relevé sort sur STDERR, avant le rapport, et il nomme l'unité, la raison, et
// le nombre de contrôles que cela coûte.
func TestTheCapabilityReportSaysWhatCouldNotBeObserved(t *testing.T) {
	bin := buildPepin(t)
	_, stderr, _ := runScan(t, bin, "scan", "scaleway", withIncompleteUnit(t))

	for _, want := range []string{
		"Collector capability report",
		"security_group_rule",
		"insufficient privilege",
		"HTTP 403",
		"cannot be evaluated",
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("le relevé de capacités ne dit pas %q :\n%s", want, stderr)
		}
	}
}

// TestTheCollectionStateIsAParsableSurface : un pipeline qui accepte le code d'une
// collecte incomplète doit pouvoir lire CE QUI a manqué. Journaliser ne suffit pas ;
// l'issue demande un enregistrement.
func TestTheCollectionStateIsAParsableSurface(t *testing.T) {
	bin := buildPepin(t)
	stdout, _, _ := runScan(t, bin, "scan", "scaleway", withIncompleteUnit(t), "--format", "json")

	var doc struct {
		Collection struct {
			Units []struct {
				Unit     string `json:"unit"`
				Complete bool   `json:"complete"`
				Error    string `json:"error"`
			} `json:"units"`
		} `json:"collection"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("sortie JSON illisible : %v", err)
	}
	found := false
	for _, u := range doc.Collection.Units {
		if u.Unit != "security_group_rule" {
			continue
		}
		found = true
		if u.Complete {
			t.Error("l'unité en échec ne doit pas être publiée comme complète")
		}
		if u.Error != "permission_denied" {
			t.Errorf("classe d'échec publiée %q, attendue permission_denied", u.Error)
		}
	}
	if !found {
		t.Errorf("`collection` doit publier l'unité en échec :\n%s", stdout)
	}
}

// TestACompleteScanCarriesNoCapabilityReport : la contrepartie. Un scan qui n'a
// rien à signaler ne doit rien imprimer de plus — un relevé qui apparaît sur chaque
// exécution est un relevé que personne ne lit, et l'avertissement se dévalue.
func TestACompleteScanCarriesNoCapabilityReport(t *testing.T) {
	bin := buildPepin(t)
	_, stderr, _ := runScan(t, bin, "scan", "scaleway", "-t", exemptFixture)
	if strings.Contains(stderr, "capability report") {
		t.Errorf("un scan sans anomalie de collecte ne doit pas imprimer de relevé :\n%s", stderr)
	}
}
