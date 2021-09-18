package spec

import (
	"fmt"
	"testing"
)

var idTests = []struct {
	id      string
	invalid bool
}{
	{
		id: "foo",
	},
	{
		id: "foo2",
	},
	{
		id: "foo_bar_baz",
	},
	{
		id: "f_o_o",
	},
	{
		id:      "2foo",
		invalid: true,
	},
	{
		id:      "_foo",
		invalid: true,
	},
	{
		id:      "foo_",
		invalid: true,
	},
	{
		id:      "foo__bar",
		invalid: true,
	},
}

func TestValidateIdentifier(t *testing.T) {
	for _, tt := range idTests {
		t.Run(tt.id, func(t *testing.T) {
			err := validateIdentifier(tt.id)
			if tt.invalid {
				if err == nil {
					t.Errorf("expected error didn't occur")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error occurred: %v", err)
				}
			}
		})
	}
}

func TestLexKindName_validate(t *testing.T) {
	for _, tt := range idTests {
		t.Run(tt.id, func(t *testing.T) {
			err := LexKindName(tt.id).validate()
			if tt.invalid {
				if err == nil {
					t.Errorf("expected error didn't occur")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error occurred: %v", err)
				}
			}
		})
	}
}

func TestLexModeName_validate(t *testing.T) {
	for _, tt := range idTests {
		t.Run(tt.id, func(t *testing.T) {
			err := LexModeName(tt.id).validate()
			if tt.invalid {
				if err == nil {
					t.Errorf("expected error didn't occur")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error occurred: %v", err)
				}
			}
		})
	}
}

func TestSnakeCaseToUpperCamelCase(t *testing.T) {
	tests := []struct {
		snake string
		camel string
	}{
		{
			snake: "foo",
			camel: "Foo",
		},
		{
			snake: "foo_bar",
			camel: "FooBar",
		},
		{
			snake: "foo_bar_baz",
			camel: "FooBarBaz",
		},
		{
			snake: "Foo",
			camel: "Foo",
		},
		{
			snake: "fooBar",
			camel: "FooBar",
		},
		{
			snake: "FOO",
			camel: "FOO",
		},
		{
			snake: "FOO_BAR",
			camel: "FOOBAR",
		},
		{
			snake: "_foo_bar_",
			camel: "FooBar",
		},
		{
			snake: "___foo___bar___",
			camel: "FooBar",
		},
	}
	for _, tt := range tests {
		c := SnakeCaseToUpperCamelCase(tt.snake)
		if c != tt.camel {
			t.Errorf("unexpected string; want: %v, got: %v", tt.camel, c)
		}
	}
}

func TestFindSpellingInconsistencies(t *testing.T) {
	tests := []struct {
		ids        []string
		duplicated [][]string
	}{
		{
			ids:        []string{"foo", "foo"},
			duplicated: nil,
		},
		{
			ids:        []string{"foo", "Foo"},
			duplicated: [][]string{{"Foo", "foo"}},
		},
		{
			ids:        []string{"foo", "foo", "Foo"},
			duplicated: [][]string{{"Foo", "foo"}},
		},
		{
			ids:        []string{"foo_bar_baz", "FooBarBaz"},
			duplicated: [][]string{{"FooBarBaz", "foo_bar_baz"}},
		},
		{
			ids:        []string{"foo", "Foo", "bar", "Bar"},
			duplicated: [][]string{{"Bar", "bar"}, {"Foo", "foo"}},
		},
		{
			ids:        []string{"foo", "Foo", "bar", "Bar", "baz", "bra"},
			duplicated: [][]string{{"Bar", "bar"}, {"Foo", "foo"}},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%v", i), func(t *testing.T) {
			duplicated := FindSpellingInconsistencies(tt.ids)
			if len(duplicated) != len(tt.duplicated) {
				t.Fatalf("unexpected IDs; want: %#v, got: %#v", tt.duplicated, duplicated)
			}
			for i, dupIDs := range duplicated {
				if len(dupIDs) != len(tt.duplicated[i]) {
					t.Fatalf("unexpected IDs; want: %#v, got: %#v", tt.duplicated[i], dupIDs)
				}
				for j, id := range dupIDs {
					if id != tt.duplicated[i][j] {
						t.Fatalf("unexpected IDs; want: %#v, got: %#v", tt.duplicated[i], dupIDs)
					}
				}
			}
		})
	}
}

func TestLexSpec_Validate(t *testing.T) {
	// We expect that the spelling inconsistency error will occur.
	spec := &LexSpec{
		Entries: []*LexEntry{
			{
				Modes: []LexModeName{
					// 'Default' is the spelling inconsistency because 'default' is predefined.
					"Default",
				},
				Kind:    "foo",
				Pattern: "foo",
			},
		},
	}
	err := spec.Validate()
	if err == nil {
		t.Fatalf("expected error didn't occur")
	}
}
