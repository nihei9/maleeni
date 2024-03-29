package driver

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"text/template"

	"github.com/nihei9/maleeni/spec"
)

//go:embed lexer.go
var lexerCoreSrc string

func GenLexer(clspec *spec.CompiledLexSpec, pkgName string) ([]byte, error) {
	var lexerSrc string
	{
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "lexer.go", lexerCoreSrc, parser.ParseComments)
		if err != nil {
			return nil, err
		}

		var b strings.Builder
		err = format.Node(&b, fset, f)
		if err != nil {
			return nil, err
		}

		lexerSrc = b.String()
	}

	var modeIDsSrc string
	{
		var b strings.Builder
		fmt.Fprintf(&b, "const (\n")
		for i, k := range clspec.ModeNames {
			if i == spec.LexModeIDNil.Int() {
				fmt.Fprintf(&b, "    ModeIDNil ModeID = %v\n", i)
				continue
			}
			fmt.Fprintf(&b, "    ModeID%v ModeID = %v\n", spec.SnakeCaseToUpperCamelCase(k.String()), i)
		}
		fmt.Fprintf(&b, ")")

		modeIDsSrc = b.String()
	}

	var modeNamesSrc string
	{
		var b strings.Builder
		fmt.Fprintf(&b, "const (\n")
		for i, k := range clspec.ModeNames {
			if i == spec.LexModeIDNil.Int() {
				fmt.Fprintf(&b, "    ModeNameNil = %#v\n", "")
				continue
			}
			fmt.Fprintf(&b, "    ModeName%v = %#v\n", spec.SnakeCaseToUpperCamelCase(k.String()), k)
		}
		fmt.Fprintf(&b, ")")

		modeNamesSrc = b.String()
	}

	var modeIDToNameSrc string
	{
		var b strings.Builder
		fmt.Fprintf(&b, `
// ModeIDToName converts a mode ID to a name.
func ModeIDToName(id ModeID) string {
    switch id {`)
		for i, k := range clspec.ModeNames {
			if i == spec.LexModeIDNil.Int() {
				fmt.Fprintf(&b, `
    case ModeIDNil:
        return ModeNameNil`)
				continue
			}
			name := spec.SnakeCaseToUpperCamelCase(k.String())
			fmt.Fprintf(&b, `
    case ModeID%v:
        return ModeName%v`, name, name)
		}
		fmt.Fprintf(&b, `
    }
    return ""
}
`)

		modeIDToNameSrc = b.String()
	}

	var kindIDsSrc string
	{
		var b strings.Builder
		fmt.Fprintf(&b, "const (\n")
		for i, k := range clspec.KindNames {
			if i == spec.LexKindIDNil.Int() {
				fmt.Fprintf(&b, "    KindIDNil KindID = %v\n", i)
				continue
			}
			fmt.Fprintf(&b, "    KindID%v KindID = %v\n", spec.SnakeCaseToUpperCamelCase(k.String()), i)
		}
		fmt.Fprintf(&b, ")")

		kindIDsSrc = b.String()
	}

	var kindNamesSrc string
	{
		var b strings.Builder
		fmt.Fprintf(&b, "const (\n")
		fmt.Fprintf(&b, "    KindNameNil = %#v\n", "")
		for _, k := range clspec.KindNames[1:] {
			fmt.Fprintf(&b, "    KindName%v = %#v\n", spec.SnakeCaseToUpperCamelCase(k.String()), k)
		}
		fmt.Fprintf(&b, ")")

		kindNamesSrc = b.String()
	}

	var kindIDToNameSrc string
	{
		var b strings.Builder
		fmt.Fprintf(&b, `
// KindIDToName converts a kind ID to a name.
func KindIDToName(id KindID) string {
    switch id {`)
		for i, k := range clspec.KindNames {
			if i == spec.LexModeIDNil.Int() {
				fmt.Fprintf(&b, `
    case KindIDNil:
        return KindNameNil`)
				continue
			}
			name := spec.SnakeCaseToUpperCamelCase(k.String())
			fmt.Fprintf(&b, `
    case KindID%v:
        return KindName%v`, name, name)
		}
		fmt.Fprintf(&b, `
    }
    return ""
}
`)

		kindIDToNameSrc = b.String()
	}

	var specSrc string
	{
		t, err := template.New("").Funcs(genTemplateFuncs(clspec)).Parse(lexSpecTemplate)
		if err != nil {
			return nil, err
		}

		var b strings.Builder
		err = t.Execute(&b, map[string]interface{}{
			"initialModeID":    "ModeID" + spec.SnakeCaseToUpperCamelCase(clspec.ModeNames[clspec.InitialModeID].String()),
			"modeIDNil":        "ModeIDNil",
			"modeKindIDNil":    spec.LexModeKindIDNil,
			"stateIDNil":       spec.StateIDNil,
			"compressionLevel": clspec.CompressionLevel,
		})
		if err != nil {
			return nil, err
		}

		specSrc = b.String()
	}

	var src string
	{
		tmpl := `// Code generated by maleeni-go. DO NOT EDIT.
{{ .lexerSrc }}

{{ .modeIDsSrc }}

{{ .modeNamesSrc }}

{{ .modeIDToNameSrc }}

{{ .kindIDsSrc }}

{{ .kindNamesSrc }}

{{ .kindIDToNameSrc }}

{{ .specSrc }}
`

		t, err := template.New("").Parse(tmpl)
		if err != nil {
			return nil, err
		}

		var b strings.Builder
		err = t.Execute(&b, map[string]string{
			"lexerSrc":        lexerSrc,
			"modeIDsSrc":      modeIDsSrc,
			"modeNamesSrc":    modeNamesSrc,
			"modeIDToNameSrc": modeIDToNameSrc,
			"kindIDsSrc":      kindIDsSrc,
			"kindNamesSrc":    kindNamesSrc,
			"kindIDToNameSrc": kindIDToNameSrc,
			"specSrc":         specSrc,
		})
		if err != nil {
			return nil, err
		}

		src = b.String()
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	f.Name = ast.NewIdent(pkgName)

	var b bytes.Buffer
	err = format.Node(&b, fset, f)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

const lexSpecTemplate = `
type lexSpec struct {
	pop           [][]bool
	push          [][]ModeID
	modeNames     []string
	initialStates []StateID
	acceptances   [][]ModeKindID
	kindIDs       [][]KindID
	kindNames     []string
	initialModeID ModeID
	modeIDNil     ModeID
	modeKindIDNil ModeKindID
	stateIDNil    StateID

	rowNums           [][]int
	rowDisplacements  [][]int
	bounds            [][]int
	entries           [][]StateID
	originalColCounts []int
}

func NewLexSpec() *lexSpec {
	return &lexSpec{
		pop: {{ genPopTable }},
		push: {{ genPushTable }},
		modeNames: {{ genModeNameTable }},
		initialStates: {{ genInitialStateTable }},
		acceptances: {{ genAcceptTable }},
		kindIDs: {{ genKindIDTable }},
		kindNames: {{ genKindNameTable }},
		initialModeID: {{ .initialModeID }},
		modeIDNil: {{ .modeIDNil }},
		modeKindIDNil: {{ .modeKindIDNil }},
		stateIDNil: {{ .stateIDNil }},

		rowNums: {{ genRowNums }},
		rowDisplacements: {{ genRowDisplacements }},
		bounds: {{ genBounds }},
		entries: {{ genEntries }},
		originalColCounts: {{ genOriginalColCounts }},
	}
}

func (s *lexSpec) InitialMode() ModeID {
	return s.initialModeID
}

func (s *lexSpec) Pop(mode ModeID, modeKind ModeKindID) bool {
	return s.pop[mode][modeKind]
}

func (s *lexSpec) Push(mode ModeID, modeKind ModeKindID) (ModeID, bool) {
	id := s.push[mode][modeKind]
	return id, id != s.modeIDNil
}

func (s *lexSpec) ModeName(mode ModeID) string {
	return s.modeNames[mode]
}

func (s *lexSpec) InitialState(mode ModeID) StateID {
	return s.initialStates[mode]
}

func (s *lexSpec) NextState(mode ModeID, state StateID, v int) (StateID, bool) {
{{ if eq .compressionLevel 2 -}}
	rowNum := s.rowNums[mode][state]
	d := s.rowDisplacements[mode][rowNum]
	if s.bounds[mode][d+v] != rowNum {
		return s.stateIDNil, false
	}
	return s.entries[mode][d+v], true
{{ else if eq .compressionLevel 1 -}}
	rowNum := s.rowNums[mode][state]
	colCount := s.originalColCounts[mode]
	next := s.entries[mode][rowNum*colCount+v]
	if next == s.stateIDNil {
		return s.stateIDNil, false
	}
	return next, true
{{ else -}}
	colCount := s.originalColCounts[mode]
	next := s.entries[mode][int(state)*colCount+v]
	if next == s.stateIDNil {
		return s.stateIDNil, false
	}
	return next, true
{{ end -}}
}

func (s *lexSpec) Accept(mode ModeID, state StateID) (ModeKindID, bool) {
	id := s.acceptances[mode][state]
	return id, id != s.modeKindIDNil
}

func (s *lexSpec) KindIDAndName(mode ModeID, modeKind ModeKindID) (KindID, string) {
	id := s.kindIDs[mode][modeKind]
	return id, s.kindNames[id]
}
`

func genTemplateFuncs(clspec *spec.CompiledLexSpec) template.FuncMap {
	fns := template.FuncMap{
		"genPopTable": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[][]bool{\n")
			for i, s := range clspec.Specs {
				if i == spec.LexModeIDNil.Int() {
					fmt.Fprintf(&b, "nil,\n")
					continue
				}

				c := 1
				fmt.Fprintf(&b, "{\n")
				for _, v := range s.Pop {
					fmt.Fprintf(&b, "%v, ", v != 0)

					if c == 20 {
						fmt.Fprintf(&b, "\n")
						c = 1
					} else {
						c++
					}
				}
				if c > 1 {
					fmt.Fprintf(&b, "\n")
				}
				fmt.Fprintf(&b, "},\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
		"genPushTable": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[][]ModeID{\n")
			for i, s := range clspec.Specs {
				if i == spec.LexModeIDNil.Int() {
					fmt.Fprintf(&b, "nil,\n")
					continue
				}

				c := 1
				fmt.Fprintf(&b, "{\n")
				for _, v := range s.Push {
					fmt.Fprintf(&b, "%v,", v)

					if c == 20 {
						fmt.Fprintf(&b, "\n")
						c = 1
					} else {
						c++
					}
				}
				if c > 1 {
					fmt.Fprintf(&b, "\n")
				}
				fmt.Fprintf(&b, "},\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
		"genModeNameTable": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[]string{\n")
			for i, name := range clspec.ModeNames {
				if i == spec.LexModeIDNil.Int() {
					fmt.Fprintf(&b, "ModeNameNil,\n")
					continue
				}
				fmt.Fprintf(&b, "ModeName%v,\n", spec.SnakeCaseToUpperCamelCase(name.String()))
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
		"genInitialStateTable": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[]StateID{\n")
			for i, s := range clspec.Specs {
				if i == spec.LexModeIDNil.Int() {
					fmt.Fprintf(&b, "%v,\n", spec.StateIDNil)
					continue
				}

				fmt.Fprintf(&b, "%v,\n", s.DFA.InitialStateID)
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
		"genAcceptTable": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[][]ModeKindID{\n")
			for i, s := range clspec.Specs {
				if i == spec.LexModeIDNil.Int() {
					fmt.Fprintf(&b, "nil,\n")
					continue
				}

				c := 1
				fmt.Fprintf(&b, "{\n")
				for _, v := range s.DFA.AcceptingStates {
					fmt.Fprintf(&b, "%v,", v)

					if c == 20 {
						fmt.Fprintf(&b, "\n")
						c = 1
					} else {
						c++
					}
				}
				if c > 1 {
					fmt.Fprintf(&b, "\n")
				}
				fmt.Fprintf(&b, "},\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
		"genKindIDTable": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[][]KindID{\n")
			for i, ids := range clspec.KindIDs {
				if i == spec.LexModeIDNil.Int() {
					fmt.Fprintf(&b, "nil,\n")
					continue
				}

				fmt.Fprintf(&b, "{\n")
				for j, id := range ids {
					if j == spec.LexModeKindIDNil.Int() {
						fmt.Fprintf(&b, "KindIDNil,\n")
						continue
					}
					fmt.Fprintf(&b, "KindID%v,\n", spec.SnakeCaseToUpperCamelCase(string(clspec.KindNames[id].String())))
				}
				fmt.Fprintf(&b, "},\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
		"genKindNameTable": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[]string{\n")
			for i, name := range clspec.KindNames {
				if i == spec.LexKindIDNil.Int() {
					fmt.Fprintf(&b, "KindNameNil,\n")
					continue
				}
				fmt.Fprintf(&b, "KindName%v,\n", spec.SnakeCaseToUpperCamelCase(name.String()))
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
	}

	switch clspec.CompressionLevel {
	case 2:
		fns["genRowNums"] = func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[][]int{\n")
			for i, s := range clspec.Specs {
				if i == spec.LexModeIDNil.Int() {
					fmt.Fprintf(&b, "nil,\n")
					continue
				}

				c := 1
				fmt.Fprintf(&b, "{\n")
				for _, v := range s.DFA.Transition.RowNums {
					fmt.Fprintf(&b, "%v,", v)

					if c == 20 {
						fmt.Fprintf(&b, "\n")
						c = 1
					} else {
						c++
					}
				}
				if c > 1 {
					fmt.Fprintf(&b, "\n")
				}
				fmt.Fprintf(&b, "},\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		}

		fns["genRowDisplacements"] = func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[][]int{\n")
			for i, s := range clspec.Specs {
				if i == spec.LexModeIDNil.Int() {
					fmt.Fprintf(&b, "nil,\n")
					continue
				}

				c := 1
				fmt.Fprintf(&b, "{\n")
				for _, d := range s.DFA.Transition.UniqueEntries.RowDisplacement {
					fmt.Fprintf(&b, "%v,", d)

					if c == 20 {
						fmt.Fprintf(&b, "\n")
						c = 1
					} else {
						c++
					}
				}
				if c > 1 {
					fmt.Fprintf(&b, "\n")
				}
				fmt.Fprintf(&b, "},\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		}

		fns["genBounds"] = func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[][]int{\n")
			for i, s := range clspec.Specs {
				if i == spec.LexModeIDNil.Int() {
					fmt.Fprintf(&b, "nil,\n")
					continue
				}

				c := 1
				fmt.Fprintf(&b, "{\n")
				for _, v := range s.DFA.Transition.UniqueEntries.Bounds {
					fmt.Fprintf(&b, "%v,", v)

					if c == 20 {
						fmt.Fprintf(&b, "\n")
						c = 1
					} else {
						c++
					}
				}
				if c > 1 {
					fmt.Fprintf(&b, "\n")
				}
				fmt.Fprintf(&b, "},\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		}

		fns["genEntries"] = func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[][]StateID{\n")
			for i, s := range clspec.Specs {
				if i == spec.LexModeIDNil.Int() {
					fmt.Fprintf(&b, "nil,\n")
					continue
				}

				c := 1
				fmt.Fprintf(&b, "{\n")
				for _, v := range s.DFA.Transition.UniqueEntries.Entries {
					fmt.Fprintf(&b, "%v,", v)

					if c == 20 {
						fmt.Fprintf(&b, "\n")
						c = 1
					} else {
						c++
					}
				}
				if c > 1 {
					fmt.Fprintf(&b, "\n")
				}
				fmt.Fprintf(&b, "},\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		}

		fns["genOriginalColCounts"] = func() string {
			return "nil"
		}
	case 1:
		fns["genRowNums"] = func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[][]int{\n")
			for i, s := range clspec.Specs {
				if i == spec.LexModeIDNil.Int() {
					fmt.Fprintf(&b, "nil,\n")
					continue
				}

				c := 1
				fmt.Fprintf(&b, "{\n")
				for _, v := range s.DFA.Transition.RowNums {
					fmt.Fprintf(&b, "%v,", v)

					if c == 20 {
						fmt.Fprintf(&b, "\n")
						c = 1
					} else {
						c++
					}
				}
				if c > 1 {
					fmt.Fprintf(&b, "\n")
				}
				fmt.Fprintf(&b, "},\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		}

		fns["genRowDisplacements"] = func() string {
			return "nil"
		}

		fns["genBounds"] = func() string {
			return "nil"
		}

		fns["genEntries"] = func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[][]StateID{\n")
			for i, s := range clspec.Specs {
				if i == spec.LexModeIDNil.Int() {
					fmt.Fprintf(&b, "nil,\n")
					continue
				}

				c := 1
				fmt.Fprintf(&b, "{\n")
				for _, v := range s.DFA.Transition.UncompressedUniqueEntries {
					fmt.Fprintf(&b, "%v,", v)

					if c == 20 {
						fmt.Fprintf(&b, "\n")
						c = 1
					} else {
						c++
					}
				}
				if c > 1 {
					fmt.Fprintf(&b, "\n")
				}
				fmt.Fprintf(&b, "},\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		}

		fns["genOriginalColCounts"] = func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[]int{\n")
			for i, s := range clspec.Specs {
				if i == spec.LexModeIDNil.Int() {
					fmt.Fprintf(&b, "0,\n")
					continue
				}

				fmt.Fprintf(&b, "%v,\n", s.DFA.Transition.OriginalColCount)
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		}
	default:
		fns["genRowNums"] = func() string {
			return "nil"
		}

		fns["genRowDisplacements"] = func() string {
			return "nil"
		}

		fns["genBounds"] = func() string {
			return "nil"
		}

		fns["genEntries"] = func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[][]StateID{\n")
			for i, s := range clspec.Specs {
				if i == spec.LexModeIDNil.Int() {
					fmt.Fprintf(&b, "nil,\n")
					continue
				}

				c := 1
				fmt.Fprintf(&b, "{\n")
				for _, v := range s.DFA.UncompressedTransition {
					fmt.Fprintf(&b, "%v,", v)

					if c == 20 {
						fmt.Fprintf(&b, "\n")
						c = 1
					} else {
						c++
					}
				}
				if c > 1 {
					fmt.Fprintf(&b, "\n")
				}
				fmt.Fprintf(&b, "},\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		}

		fns["genOriginalColCounts"] = func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[]int{\n")
			for i, s := range clspec.Specs {
				if i == spec.LexModeIDNil.Int() {
					fmt.Fprintf(&b, "0,\n")
					continue
				}

				fmt.Fprintf(&b, "%v,\n", s.DFA.ColCount)
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		}
	}

	return fns
}
