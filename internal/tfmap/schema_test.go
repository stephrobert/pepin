package tfmap

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDefaultTransformRequiresOptionalNonComputedAttribute(t *testing.T) {
	ref := tfBlock{Attributes: map[string]json.RawMessage{
		"optional":     json.RawMessage(`{"optional":true}`),
		"required":     json.RawMessage(`{"required":true}`),
		"computed":     json.RawMessage(`{"computed":true,"optional":true}`),
		"unclassified": json.RawMessage(`{}`),
		"parent_value": json.RawMessage(`{"optional":true}`),
	}}
	root := tfBlock{Attributes: map[string]json.RawMessage{
		"parent_value":    json.RawMessage(`{"required":true}`),
		"parent_computed": json.RawMessage(`{"computed":true}`),
	}}
	tests := []struct {
		name      string
		path      string
		transform any
		want      string
	}{
		{"optional", "optional", "default_null:false", ""},
		{"required", "required", []any{"default_null:false", "to_bool"}, "REQUIRED"},
		{"computed", "computed", "default:false", "COMPUTED"},
		{"coalesced computed", "optional||computed", "default:false", "COMPUTED"},
		{"parent required", "_parent.parent_value", "default:false", "REQUIRED"},
		{"parent computed", "_parent.parent_computed", "default:false", "COMPUTED"},
		{"unknown parent", "_parent.missing", "default:false", "drapeaux"},
		{"not optional", "unclassified", "default:false", "OPTIONAL"},
		{"no default", "required", "lower", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := ResourceSpec{
				TFType:     "example_resource",
				Map:        map[string]string{"value": tt.path},
				Transforms: map[string]any{"value": tt.transform},
			}
			got := defautsInterdits(rs, ref, root)
			if tt.want == "" {
				if len(got) != 0 {
					t.Errorf("unexpected errors: %v", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tt.want) {
				t.Errorf("errors = %v; want one containing %q", got, tt.want)
			}
		})
	}
}
