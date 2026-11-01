package walle

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCanonicalNumericEnumPreservesReachableValues(t *testing.T) {
	cases := []struct {
		name         string
		property     string
		want         string
		alreadyValid bool
	}{
		{"nullable integer", `{"type":["integer","null"],"enum":[1,null]}`, `{"type":["integer","null"],"enum":[1,null]}`, true},
		{"null first integer", `{"type":["null","integer"],"enum":[1,null]}`, `{"type":["null","integer"],"enum":[1,null]}`, true},
		{"nullable number", `{"type":["number","null"],"enum":[1.5,null]}`, `{"type":["number","null"],"enum":[1.5,null]}`, true},
		{"null first number", `{"type":["null","number"],"enum":[1.5,null]}`, `{"type":["null","number"],"enum":[1.5,null]}`, true},
		{"integer excludes fractional values", `{"type":"integer","enum":[1,1.5,2]}`, `{"type":"integer","enum":[1,2]}`, false},
		{"integer excludes strings", `{"type":"integer","enum":[1,"x",2]}`, `{"type":"integer","enum":[1,2]}`, false},
		{"number excludes strings", `{"type":"number","enum":[1,"x",2]}`, `{"type":"number","enum":[1,2]}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := `{"type":"object","properties":{"value":` + tc.property + `},"required":["value"]}`
			schema, err := ParseSchema(input)
			if err != nil {
				t.Fatal(err)
			}
			if tc.alreadyValid {
				if err := schema.Validate(); err != nil {
					t.Errorf("valid numeric enum rejected: %v", err)
				}
			}
			canonical, warning := schema.Canonical()
			if tc.alreadyValid && warning != nil {
				t.Errorf("valid numeric enum produced warning: %v", warning)
			}
			if !tc.alreadyValid && warning == nil {
				t.Error("expected warning for unreachable enum values")
			}
			var got, want any
			if err := json.Unmarshal([]byte(canonical), &got); err != nil {
				t.Fatal(err)
			}
			expected := `{"type":"object","properties":{"value":` + tc.want + `},"required":["value"]}`
			if err := json.Unmarshal([]byte(expected), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("canonical schema lost reachable enum values: got %s; want %s", canonical, expected)
			}
		})
	}
}
