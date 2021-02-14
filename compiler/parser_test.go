package compiler

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"
)

func printAST(w io.Writer, ast astNode, ruledLine string, childRuledLinePrefix string, withAttrs bool) {
	if ast == nil {
		return
	}
	fmt.Fprintf(w, ruledLine)
	fmt.Fprintf(w, "node: %v", ast)
	if withAttrs {
		fmt.Fprintf(w, ", nullable: %v, first: %v, last: %v", ast.nullable(), ast.first(), ast.last())
	}
	fmt.Fprintf(w, "\n")
	left, right := ast.children()
	children := []astNode{}
	if left != nil {
		children = append(children, left)
	}
	if right != nil {
		children = append(children, right)
	}
	num := len(children)
	for i, child := range children {
		line := "└─ "
		if num > 1 {
			if i == 0 {
				line = "├─ "
			} else if i < num-1 {
				line = "│  "
			}
		}
		prefix := "│  "
		if i >= num-1 {
			prefix = "    "
		}
		printAST(w, child, childRuledLinePrefix+line, childRuledLinePrefix+prefix, withAttrs)
	}
}

func TestParser(t *testing.T) {
	newCharTok := func(char rune) *token {
		return newToken(tokenKindChar, char)
	}

	rune2Byte := func(char rune, index int) byte {
		return []byte(string(char))[index]
	}

	symPos := func(n uint8) symbolPosition {
		return newSymbolPosition(n, false)
	}

	endPos := func(n uint8) symbolPosition {
		return newSymbolPosition(n, true)
	}

	root, symTab, err := parse(map[int][]byte{
		1: []byte("(a|b)*abb"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if root == nil {
		t.Fatal("root of AST is nil")
	}
	printAST(os.Stdout, root, "", "", false)

	{
		expectedAST := newConcatNode(
			newConcatNode(
				newConcatNode(
					newConcatNode(
						newRepeatNode(
							newAltNode(
								newSymbolNode(newCharTok('a'), rune2Byte('a', 0), symPos(1)),
								newSymbolNode(newCharTok('b'), rune2Byte('b', 0), symPos(2)),
							),
						),
						newSymbolNode(newCharTok('a'), rune2Byte('a', 0), symPos(3)),
					),
					newSymbolNode(newCharTok('b'), rune2Byte('b', 0), symPos(4)),
				),
				newSymbolNode(newCharTok('b'), rune2Byte('b', 0), symPos(5)),
			),
			newEndMarkerNode(1, endPos(6)),
		)
		testAST(t, expectedAST, root)
	}

	{
		followTab := genFollowTable(root)
		if followTab == nil {
			t.Fatal("follow table is nil")
		}
		expectedFollowTab := followTable{
			1: newSymbolPositionSet().add(symPos(1)).add(symPos(2)).add(symPos(3)),
			2: newSymbolPositionSet().add(symPos(1)).add(symPos(2)).add(symPos(3)),
			3: newSymbolPositionSet().add(symPos(4)),
			4: newSymbolPositionSet().add(symPos(5)),
			5: newSymbolPositionSet().add(endPos(6)),
		}
		testFollowTable(t, expectedFollowTab, followTab)
	}

	{
		entry := func(v byte) byteRange {
			return byteRange{
				from: v,
				to:   v,
			}
		}

		expectedSymTab := &symbolTable{
			symPos2Byte: map[symbolPosition]byteRange{
				symPos(1): entry(byte('a')),
				symPos(2): entry(byte('b')),
				symPos(3): entry(byte('a')),
				symPos(4): entry(byte('b')),
				symPos(5): entry(byte('b')),
			},
			endPos2ID: map[symbolPosition]int{
				endPos(6): 1,
			},
		}
		testSymbolTable(t, expectedSymTab, symTab)
	}
}

func testAST(t *testing.T, expected, actual astNode) {
	t.Helper()

	aTy := reflect.TypeOf(actual)
	eTy := reflect.TypeOf(expected)
	if eTy != aTy {
		t.Fatalf("AST node type is mismatched; want: %v, got: %v", eTy, aTy)
	}

	if actual == nil {
		return
	}

	switch e := expected.(type) {
	case *symbolNode:
		a := actual.(*symbolNode)
		if a.token.char != e.token.char {
			t.Fatalf("character is mismatched; want: '%v' (%v), got: '%v' (%v)", string(e.token.char), e.token.char, string(a.token.char), a.token.char)
		}
		if a.pos != e.pos {
			t.Fatalf("symbol position is mismatched; want: %v, got: %v", e.pos, a.pos)
		}
	case *endMarkerNode:
		a := actual.(*endMarkerNode)
		if a.pos != e.pos {
			t.Fatalf("symbol position is mismatched; want: %v, got: %v", e.pos, a.pos)
		}
	}
	eLeft, eRight := expected.children()
	aLeft, aRight := actual.children()
	testAST(t, eLeft, aLeft)
	testAST(t, eRight, aRight)
}

func testFollowTable(t *testing.T, expected, actual followTable) {
	if len(actual) != len(expected) {
		t.Errorf("unexpected number of the follow table entries; want: %v, got: %v", len(expected), len(actual))
	}
	for ePos, eSet := range expected {
		aSet, ok := actual[ePos]
		if !ok {
			t.Fatalf("follow entry is not found; position: %v, follow: %v", ePos, eSet)
		}
		if aSet.hash() != eSet.hash() {
			t.Fatalf("follow entry of position %v is mismatched; want: %v, got: %v", ePos, aSet, eSet)
		}
	}
}

func testSymbolTable(t *testing.T, expected, actual *symbolTable) {
	t.Helper()

	if len(actual.symPos2Byte) != len(expected.symPos2Byte) {
		t.Errorf("unexpected symPos2Byte entries; want: %v entries, got: %v entries", len(expected.symPos2Byte), len(actual.symPos2Byte))
	}
	for ePos, eByte := range expected.symPos2Byte {
		byte, ok := actual.symPos2Byte[ePos]
		if !ok {
			t.Errorf("a symbol position entry was not found: %v -> %v", ePos, eByte)
			continue
		}
		if byte.from != eByte.from || byte.to != eByte.to {
			t.Errorf("unexpected symbol position entry; want: %v -> %v, got: %v -> %v", ePos, eByte, ePos, byte)
		}
	}

	if len(actual.endPos2ID) != len(expected.endPos2ID) {
		t.Errorf("unexpected endPos2ID entries; want: %v entries, got: %v entries", len(expected.endPos2ID), len(actual.endPos2ID))
	}
	for ePos, eID := range expected.endPos2ID {
		id, ok := actual.endPos2ID[ePos]
		if !ok {
			t.Errorf("an end position entry was not found: %v -> %v", ePos, eID)
			continue
		}
		if id != eID {
			t.Errorf("unexpected end position entry; want: %v -> %v, got: %v -> %v", ePos, eID, ePos, id)
		}
	}
}
