package compiler

import (
	"sort"

	"github.com/nihei9/maleeni/spec"
)

type DFA struct {
	States               []string
	InitialState         string
	AcceptingStatesTable map[string]spec.LexModeKindID
	TransitionTable      map[string][256]string
}

func genDFA(root astNode, symTab *symbolTable) *DFA {
	initialState := root.first()
	initialStateHash := initialState.hash()
	stateMap := map[string]*symbolPositionSet{
		initialStateHash: initialState,
	}
	tranTab := map[string][256]string{}
	{
		follow := genFollowTable(root)
		unmarkedStates := map[string]*symbolPositionSet{
			initialStateHash: initialState,
		}
		for len(unmarkedStates) > 0 {
			nextUnmarkedStates := map[string]*symbolPositionSet{}
			for hash, state := range unmarkedStates {
				tranTabOfState := [256]*symbolPositionSet{}
				for _, pos := range state.set() {
					if pos.isEndMark() {
						continue
					}
					valRange := symTab.symPos2Byte[pos]
					for symVal := valRange.from; symVal <= valRange.to; symVal++ {
						if tranTabOfState[symVal] == nil {
							tranTabOfState[symVal] = newSymbolPositionSet()
						}
						tranTabOfState[symVal].merge(follow[pos])
					}
				}
				for _, t := range tranTabOfState {
					if t == nil {
						continue
					}
					h := t.hash()
					if _, ok := stateMap[h]; ok {
						continue
					}
					stateMap[h] = t
					nextUnmarkedStates[h] = t
				}
				tabOfState := [256]string{}
				for v, t := range tranTabOfState {
					if t == nil {
						continue
					}
					tabOfState[v] = t.hash()
				}
				tranTab[hash] = tabOfState
			}
			unmarkedStates = nextUnmarkedStates
		}
	}

	accTab := map[string]spec.LexModeKindID{}
	{
		for h, s := range stateMap {
			for _, pos := range s.set() {
				if !pos.isEndMark() {
					continue
				}
				priorID, ok := accTab[h]
				if !ok {
					accTab[h] = symTab.endPos2ID[pos]
				} else {
					id := symTab.endPos2ID[pos]
					if id < priorID {
						accTab[h] = id
					}
				}
			}
		}
	}

	var states []string
	{
		for s := range stateMap {
			states = append(states, s)
		}
		sort.Slice(states, func(i, j int) bool {
			return states[i] < states[j]
		})
	}

	return &DFA{
		States:               states,
		InitialState:         initialStateHash,
		AcceptingStatesTable: accTab,
		TransitionTable:      tranTab,
	}
}

func genTransitionTable(dfa *DFA) (*spec.TransitionTable, error) {
	stateHash2ID := map[string]spec.StateID{}
	for i, s := range dfa.States {
		// Since 0 represents an invalid value in a transition table,
		// assign a number greater than or equal to 1 to states.
		stateHash2ID[s] = spec.StateID(i + spec.StateIDMin.Int())
	}

	acc := make([]spec.LexModeKindID, len(dfa.States)+1)
	for _, s := range dfa.States {
		id, ok := dfa.AcceptingStatesTable[s]
		if !ok {
			continue
		}
		acc[stateHash2ID[s]] = id
	}

	rowCount := len(dfa.States) + 1
	colCount := 256
	tran := make([]spec.StateID, rowCount*colCount)
	for s, tab := range dfa.TransitionTable {
		for v, to := range tab {
			tran[stateHash2ID[s].Int()*256+v] = stateHash2ID[to]
		}
	}

	return &spec.TransitionTable{
		InitialStateID:         stateHash2ID[dfa.InitialState],
		AcceptingStates:        acc,
		UncompressedTransition: tran,
		RowCount:               rowCount,
		ColCount:               colCount,
	}, nil
}
