package cmd

import (
	"reflect"
	"testing"

	"github.com/stephrobert/pepin/internal/policy"
)

func TestResolvedConfigOfPreservesDefaultsAndExplicitEmptyTypes(t *testing.T) {
	defaultTypes := policy.Defaults().Tagging.ResourceTypes
	for _, tt := range []struct {
		name      string
		input     any
		wantTypes []string
		wantError bool
	}{
		{"legacy input", map[string]any{}, defaultTypes, false},
		{"legacy partial config", map[string]any{"config": map[string]any{"tagging": map[string]any{}}}, defaultTypes, false},
		{"custom types", map[string]any{"config": map[string]any{"tagging": map[string]any{"resource_types": []string{"security_group"}}}}, []string{"security_group"}, false},
		{"explicit empty types", map[string]any{"config": map[string]any{"tagging": map[string]any{"resource_types": []string{}}}}, []string{}, false},
		{"null config", map[string]any{"config": nil}, nil, true},
		{"null tagging", map[string]any{"config": map[string]any{"tagging": nil}}, nil, true},
		{"null types", map[string]any{"config": map[string]any{"tagging": map[string]any{"resource_types": nil}}}, nil, true},
		{"invalid types", map[string]any{"config": map[string]any{"tagging": map[string]any{"resource_types": "bucket"}}}, nil, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvedConfigOf(tt.input)
			if (err != nil) != tt.wantError {
				t.Fatalf("erreur = %v ; attendue = %t", err, tt.wantError)
			}
			if !tt.wantError && !reflect.DeepEqual(got.Tagging.ResourceTypes, tt.wantTypes) {
				t.Fatalf("types = %v ; attendus = %v", got.Tagging.ResourceTypes, tt.wantTypes)
			}
		})
	}
}
