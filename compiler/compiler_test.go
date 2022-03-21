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
    "name": "test",
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
    "name": "test",
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
    "name": "test",
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
		{
			Caption: "don't allow kind names in the same mode to contain spelling inconsistencies",
			Spec: `
{
    "name": "test",
    "entries": [
        {
            "kind": "foo_1",
            "pattern": "foo_1"
        },
        {
            "kind": "foo1",
            "pattern": "foo1"
        }
    ]
}
`,
			Err: true,
		},
		{
			Caption: "don't allow kind names across modes to contain spelling inconsistencies",
			Spec: `
{
    "name": "test",
    "entries": [
        {
            "modes": ["default"],
            "kind": "foo_1",
            "pattern": "foo_1"
        },
        {
            "modes": ["other_mode"],
            "kind": "foo1",
            "pattern": "foo1"
        }
    ]
}
`,
			Err: true,
		},
		{
			Caption: "don't allow mode names to contain spelling inconsistencies",
			Spec: `
{
    "name": "test",
    "entries": [
        {
            "modes": ["foo_1"],
            "kind": "a",
            "pattern": "a"
        },
        {
            "modes": ["foo1"],
            "kind": "b",
            "pattern": "b"
        }
    ]
}
`,
			Err: true,
		},
		{
			Caption: "allow fragment names in the same mode to contain spelling inconsistencies because fragments will not appear in output files",
			Spec: `
{
    "name": "test",
    "entries": [
        {
            "kind": "a",
            "pattern": "a"
        },
        {
            "fragment": true,
            "kind": "foo_1",
            "pattern": "foo_1"
        },
        {
            "fragment": true,
            "kind": "foo1",
            "pattern": "foo1"
        }
    ]
}
`,
		},
		{
			Caption: "allow fragment names across modes to contain spelling inconsistencies because fragments will not appear in output files",
			Spec: `
{
    "name": "test",
    "entries": [
        {
            "modes": ["default"],
            "kind": "a",
            "pattern": "a"
        },
        {
            "modes": ["default"],
            "fragment": true,
            "kind": "foo_1",
            "pattern": "foo_1"
        },
        {
            "modes": ["other_mode"],
            "fragment": true,
            "kind": "foo1",
            "pattern": "foo1"
        }
    ]
}
`,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%v %s", i, tt.Caption), func(t *testing.T) {
			lspec := &spec.LexSpec{}
			err := json.Unmarshal([]byte(tt.Spec), lspec)
			if err != nil {
				t.Fatalf("%v", err)
			}
			clspec, err, _ := Compile(lspec)
			if tt.Err {
				if err == nil {
					t.Fatalf("expected an error")
				}
				if clspec != nil {
					t.Fatalf("Compile function mustn't return a compiled specification")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if clspec == nil {
					t.Fatalf("Compile function must return a compiled specification")
				}
			}
		})
	}
}
