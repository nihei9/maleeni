package compiler

import (
	"fmt"
	"testing"
)

func TestASTNode(t *testing.T) {
	tests := []struct {
		root     astNode
		nullable bool
		first    symbolPositionSet
		last     symbolPositionSet
	}{
		{
			root:     newSymbolNodeWithPos(0, 1),
			nullable: false,
			first:    newSymbolPositionSet().add(1),
			last:     newSymbolPositionSet().add(1),
		},
		{
			root:     newEndMarkerNodeWithPos(1, 1),
			nullable: false,
			first:    newSymbolPositionSet().add(1),
			last:     newSymbolPositionSet().add(1),
		},
		{
			root: newConcatNode(
				newSymbolNodeWithPos(0, 1),
				newSymbolNodeWithPos(0, 2),
			),
			nullable: false,
			first:    newSymbolPositionSet().add(1),
			last:     newSymbolPositionSet().add(2),
		},
		{
			root: newConcatNode(
				newRepeatNode(newSymbolNodeWithPos(0, 1)),
				newSymbolNodeWithPos(0, 2),
			),
			nullable: false,
			first:    newSymbolPositionSet().add(1).add(2),
			last:     newSymbolPositionSet().add(2),
		},
		{
			root: newConcatNode(
				newSymbolNodeWithPos(0, 1),
				newRepeatNode(newSymbolNodeWithPos(0, 2)),
			),
			nullable: false,
			first:    newSymbolPositionSet().add(1),
			last:     newSymbolPositionSet().add(1).add(2),
		},
		{
			root: newConcatNode(
				newRepeatNode(newSymbolNodeWithPos(0, 1)),
				newRepeatNode(newSymbolNodeWithPos(0, 2)),
			),
			nullable: true,
			first:    newSymbolPositionSet().add(1).add(2),
			last:     newSymbolPositionSet().add(1).add(2),
		},
		{
			root: newAltNode(
				newSymbolNodeWithPos(0, 1),
				newSymbolNodeWithPos(0, 2),
			),
			nullable: false,
			first:    newSymbolPositionSet().add(1).add(2),
			last:     newSymbolPositionSet().add(1).add(2),
		},
		{
			root: newAltNode(
				newRepeatNode(newSymbolNodeWithPos(0, 1)),
				newSymbolNodeWithPos(0, 2),
			),
			nullable: true,
			first:    newSymbolPositionSet().add(1).add(2),
			last:     newSymbolPositionSet().add(1).add(2),
		},
		{
			root: newAltNode(
				newSymbolNodeWithPos(0, 1),
				newRepeatNode(newSymbolNodeWithPos(0, 2)),
			),
			nullable: true,
			first:    newSymbolPositionSet().add(1).add(2),
			last:     newSymbolPositionSet().add(1).add(2),
		},
		{
			root: newAltNode(
				newRepeatNode(newSymbolNodeWithPos(0, 1)),
				newRepeatNode(newSymbolNodeWithPos(0, 2)),
			),
			nullable: true,
			first:    newSymbolPositionSet().add(1).add(2),
			last:     newSymbolPositionSet().add(1).add(2),
		},
		{
			root:     newRepeatNode(newSymbolNodeWithPos(0, 1)),
			nullable: true,
			first:    newSymbolPositionSet().add(1),
			last:     newSymbolPositionSet().add(1),
		},
		{
			root:     newOptionNode(newSymbolNodeWithPos(0, 1)),
			nullable: true,
			first:    newSymbolPositionSet().add(1),
			last:     newSymbolPositionSet().add(1),
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%v", i), func(t *testing.T) {
			if tt.root.nullable() != tt.nullable {
				t.Errorf("unexpected nullable attribute; want: %v, got: %v", tt.nullable, tt.root.nullable())
			}
			if tt.first.hash() != tt.root.first().hash() {
				t.Errorf("unexpected first positions attribute; want: %v, got: %v", tt.first, tt.root.first())
			}
			if tt.last.hash() != tt.root.last().hash() {
				t.Errorf("unexpected last positions attribute; want: %v, got: %v", tt.last, tt.root.last())
			}
		})
	}
}
