package compiler

import (
	"sort"
)

type DFA struct {
	States               []string
	InitialState         string
	AcceptingStatesTable map[string]int
	TransitionTable      map[string][256]string
}

func genDFA(root astNode, symTab *symbolTable) *DFA {
	initialState := root.first()
	initialStateHash := initialState.hash()
	stateMap := map[string]symbolPositionSet{}
	tranTab := map[string][256]string{}
	{
		follow := genFollowTable(root)
		unmarkedStates := map[string]symbolPositionSet{
			initialStateHash: initialState,
		}
		for len(unmarkedStates) > 0 {
			nextUnmarkedStates := map[string]symbolPositionSet{}
			for hash, state := range unmarkedStates {
				tranTabOfState := [256]symbolPositionSet{}
				for _, pos := range state.sort() {
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

	accTab := map[string]int{}
	{
		for h, s := range stateMap {
			for pos := range s {
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

type TransitionTable struct {
	InitialState    int         `json:"initial_state"`
	AcceptingStates map[int]int `json:"accepting_states"`
	Transition      [][]int     `json:"transition"`
}

func GenTransitionTable(dfa *DFA) (*TransitionTable, error) {
	state2Num := map[string]int{}
	for i, s := range dfa.States {
		state2Num[s] = i + 1
	}

	acc := map[int]int{}
	for s, id := range dfa.AcceptingStatesTable {
		acc[state2Num[s]] = id
	}

	tran := make([][]int, len(dfa.States)+1)
	for s, tab := range dfa.TransitionTable {
		entry := make([]int, 256)
		for v, to := range tab {
			entry[v] = state2Num[to]
		}
		tran[state2Num[s]] = entry
	}

	return &TransitionTable{
		InitialState:    state2Num[dfa.InitialState],
		AcceptingStates: acc,
		Transition:      tran,
	}, nil
}
