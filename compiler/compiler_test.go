package compiler

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/nihei9/maleeni/spec"
)

func TestCompile(t *testing.T) {
	tests := []struct {
		Caption string
		Spec    string
		Err     bool
	}{
		{
			Caption: "allow duplicates names between fragments and non-fragments",
			Spec: `
{
    "entries": [
        {
            "kind": "a2z",
            "pattern": "\\f{a2z}"
        },
        {
            "fragment": true,
            "kind": "a2z",
            "pattern": "[a-z]"
        }
    ]
}
`,
		},
		{
			Caption: "don't allow duplicates names in non-fragments",
			Spec: `
{
    "entries": [
        {
            "kind": "a2z",
            "pattern": "a|b|c|d|e|f|g|h|i|j|k|l|m|n|o|p|q|r|s|t|u|v|w|x|y|z"
        },
        {
            "kind": "a2z",
            "pattern": "[a-z]"
        }
    ]
}
`,
			Err: true,
		},
		{
			Caption: "don't allow duplicates names in fragments",
			Spec: `
{
    "entries": [
        {
            "kind": "a2z",
            "pattern": "\\f{a2z}"
        },
        {
            "fragments": true,
            "kind": "a2z",
            "pattern": "a|b|c|d|e|f|g|h|i|j|k|l|m|n|o|p|q|r|s|t|u|v|w|x|y|z"
        },
        {
            "fragments": true,
            "kind": "a2z",
            "pattern": "[a-z]"
        }
    ]
}
`,
			Err: true,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%v %s", i, tt.Caption), func(t *testing.T) {
			lspec := &spec.LexSpec{}
			err := json.Unmarshal([]byte(tt.Spec), lspec)
			if err != nil {
				t.Fatalf("%v", err)
			}
			clspec, err := Compile(lspec)
			if tt.Err {
				if err == nil {
					t.Fatalf("expected an error")
				}
				if clspec != nil {
					t.Fatalf("Compile function mustn't return a compiled specification")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error")
				}
				if clspec == nil {
					t.Fatalf("Compile function must return a compiled specification")
				}
			}
		})
	}
}
