package tfparse

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Ce que ces tests protègent : la résolution de références est ce qui rend la
// corrélation possible sur un plan, et c'est aussi la mécanique la plus facile à
// rendre APPROXIMATIVE. Chaque cas muet vaut autant que chaque cas résolu.

func planAvec(t *testing.T, doc map[string]any) []Resource {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("sérialisation du plan : %v", err)
	}
	p := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(p, raw, 0o600); err != nil {
		t.Fatalf("écriture du plan : %v", err)
	}
	rs, err := ParsePlan(p)
	if err != nil {
		t.Fatalf("lecture du plan : %v", err)
	}
	return rs
}

func refsDe(t *testing.T, rs []Resource, adresse string) map[string][]string {
	t.Helper()
	for _, r := range rs {
		if r.Address == adresse {
			return r.References
		}
	}
	t.Fatalf("ressource %q absente du plan", adresse)
	return nil
}

func ressources(vals ...map[string]any) map[string]any {
	return map[string]any{"root_module": map[string]any{"resources": vals}}
}

func res(addr, typ string, values map[string]any) map[string]any {
	return map[string]any{"address": addr, "type": typ, "name": "x", "values": values}
}

func expr(refs ...string) map[string]any {
	out := make([]any, len(refs))
	for i, r := range refs {
		out[i] = r
	}
	return map[string]any{"references": out}
}

// ── Le cas nominal : une référence directe, à plat ─────────────────────────────

func TestADeclaredReferenceResolvesToTheReferencedAddress(t *testing.T) {
	rs := planAvec(t, map[string]any{
		"planned_values": ressources(
			res("outscale_vm.web", "outscale_vm", map[string]any{"state": "running"}),
			res("outscale_security_group.web", "outscale_security_group", map[string]any{}),
		),
		"configuration": map[string]any{"root_module": map[string]any{
			"resources": []any{map[string]any{
				"address": "outscale_vm.web",
				"type":    "outscale_vm",
				// Terraform émet la traversée complète ET l'adresse : c'est la seconde
				// qui identifie la ressource, et c'est elle qu'on doit retenir.
				"expressions": map[string]any{
					"security_group_ids": expr("outscale_security_group.web.security_group_id", "outscale_security_group.web"),
				},
			}},
		}},
	})
	got := refsDe(t, rs, "outscale_vm.web")["security_group_ids"]
	if len(got) != 1 || got[0] != "outscale_security_group.web" {
		t.Fatalf("référence résolue = %v, attendu [outscale_security_group.web]", got)
	}
}

// ── LES CONTRE-EXEMPLES : ce qui ne doit RIEN résoudre ─────────────────────────

// Une variable de racine, un local, une source de données : aucun ne désigne une
// ressource du plan. Les projeter donnerait une valeur plausible et fausse.
func TestAReferenceToSomethingThatIsNotAResourceResolvesToNothing(t *testing.T) {
	for _, cible := range []string{"var.sg", "local.sg", "data.outscale_security_group.x", "outscale_security_group.absente"} {
		rs := planAvec(t, map[string]any{
			"planned_values": ressources(res("outscale_vm.web", "outscale_vm", map[string]any{})),
			"configuration": map[string]any{"root_module": map[string]any{
				"resources": []any{map[string]any{
					"address":     "outscale_vm.web",
					"type":        "outscale_vm",
					"expressions": map[string]any{"security_group_ids": expr(cible)},
				}},
			}},
		})
		if got := refsDe(t, rs, "outscale_vm.web"); len(got) != 0 {
			t.Errorf("cible %q : résolu %v, attendu rien", cible, got)
		}
	}
}

// LE contre-exemple qui compte. Une déclaration démultipliée par `count` produit N
// instances que la configuration ne distingue pas. Joindre la VM à l'une d'elles
// serait un écart posé sur une ressource qui ne le porte pas — la classe de faux
// positif la plus coûteuse du dépôt.
func TestAnAmbiguousReferenceResolvesToNothing(t *testing.T) {
	rs := planAvec(t, map[string]any{
		"planned_values": ressources(
			res("outscale_vm.web", "outscale_vm", map[string]any{}),
			res("outscale_security_group.pool[0]", "outscale_security_group", map[string]any{}),
			res("outscale_security_group.pool[1]", "outscale_security_group", map[string]any{}),
		),
		"configuration": map[string]any{"root_module": map[string]any{
			"resources": []any{map[string]any{
				"address":     "outscale_vm.web",
				"type":        "outscale_vm",
				"expressions": map[string]any{"security_group_ids": expr("outscale_security_group.pool")},
			}},
		}},
	})
	if got := refsDe(t, rs, "outscale_vm.web"); len(got) != 0 {
		t.Fatalf("résolu %v, attendu rien : deux instances ne se départagent pas", got)
	}
}

// L'instance UNIQUE d'une déclaration indexée, elle, se désigne sans ambiguïté.
func TestASingleIndexedInstanceStillResolves(t *testing.T) {
	rs := planAvec(t, map[string]any{
		"planned_values": ressources(
			res("outscale_vm.web", "outscale_vm", map[string]any{}),
			res("outscale_security_group.pool[0]", "outscale_security_group", map[string]any{}),
		),
		"configuration": map[string]any{"root_module": map[string]any{
			"resources": []any{map[string]any{
				"address":     "outscale_vm.web",
				"type":        "outscale_vm",
				"expressions": map[string]any{"security_group_ids": expr("outscale_security_group.pool")},
			}},
		}},
	})
	got := refsDe(t, rs, "outscale_vm.web")["security_group_ids"]
	if len(got) != 1 || got[0] != "outscale_security_group.pool[0]" {
		t.Fatalf("résolu %v, attendu [outscale_security_group.pool[0]]", got)
	}
}

// ── La frontière de module : sans elle, la fonctionnalité n'existe pas ─────────

// Mesuré sur le tenant de référence ztiac-two-tier : à l'intérieur d'un module,
// `security_group_ids` référence `var.security_group_ids`. Sans franchir la
// frontière, aucun plan modulaire — donc aucun plan réel — ne joint quoi que ce soit.
func TestAReferenceCrossesAModuleBoundaryThroughItsArgument(t *testing.T) {
	rs := planAvec(t, map[string]any{
		"planned_values": map[string]any{"root_module": map[string]any{
			"resources": []any{res("outscale_security_group.sg", "outscale_security_group", map[string]any{})},
			"child_modules": []any{map[string]any{
				"address":   "module.vm",
				"resources": []any{res("module.vm.outscale_vm.this", "outscale_vm", map[string]any{})},
			}},
		}},
		"configuration": map[string]any{"root_module": map[string]any{
			"module_calls": map[string]any{"vm": map[string]any{
				"source":      "./modules/vm",
				"expressions": map[string]any{"security_group_ids": expr("outscale_security_group.sg.id", "outscale_security_group.sg")},
				"module": map[string]any{"resources": []any{map[string]any{
					"address":     "outscale_vm.this",
					"type":        "outscale_vm",
					"expressions": map[string]any{"security_group_ids": expr("var.security_group_ids")},
				}}},
			}},
		}},
	})
	got := refsDe(t, rs, "module.vm.outscale_vm.this")["security_group_ids"]
	if len(got) != 1 || got[0] != "outscale_security_group.sg" {
		t.Fatalf("résolu %v, attendu [outscale_security_group.sg]", got)
	}
}

// ── Robustesse : c'est une entrée de TIERS ────────────────────────────────────

// `expressions` porte un TABLEAU pour un bloc répété (`block_device_mappings`). Une
// structure stricte y échouait, et l'échec n'était pas local : `json.Unmarshal` rend
// une erreur pour TOUT le plan, donc un plan valide devenait illisible à cause d'un
// bloc dont on n'a que faire. Mesuré sur la fixture du dépôt.
func TestARepeatedBlockDoesNotMakeTheWholePlanUnreadable(t *testing.T) {
	rs := planAvec(t, map[string]any{
		"planned_values": ressources(
			res("outscale_vm.web", "outscale_vm", map[string]any{}),
			res("outscale_security_group.web", "outscale_security_group", map[string]any{}),
		),
		"configuration": map[string]any{"root_module": map[string]any{
			"resources": []any{map[string]any{
				"address": "outscale_vm.web",
				"type":    "outscale_vm",
				"expressions": map[string]any{
					"block_device_mappings": []any{map[string]any{"expressions": map[string]any{}}},
					"security_group_ids":    expr("outscale_security_group.web"),
				},
			}},
		}},
	})
	got := refsDe(t, rs, "outscale_vm.web")["security_group_ids"]
	if len(got) != 1 {
		t.Fatalf("le bloc répété a fait perdre la référence : %v", got)
	}
}

// Un plan SANS bloc `configuration` reste parfaitement lisible : la résolution est un
// supplément, jamais une condition.
func TestAPlanWithoutConfigurationStillParses(t *testing.T) {
	rs := planAvec(t, map[string]any{
		"planned_values": ressources(res("outscale_vm.web", "outscale_vm", map[string]any{"state": "running"})),
	})
	if len(rs) != 1 || rs[0].References != nil {
		t.Fatalf("plan sans configuration : %d ressource(s), refs=%v", len(rs), rs[0].References)
	}
}
