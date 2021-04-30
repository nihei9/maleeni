package compiler

import (
	"encoding/binary"
	"fmt"
	"strings"
)

type symbolPosition uint16

const (
	symbolPositionNil = symbolPosition(0x0000) // 0000 0000 0000 0000

	symbolPositionMin = uint16(0x0001) // 0000 0000 0000 0001
	symbolPositionMax = uint16(0x7fff) // 0111 1111 1111 1111

	symbolPositionMaskSymbol  = uint16(0x0000) // 0000 0000 0000 0000
	symbolPositionMaskEndMark = uint16(0x8000) // 1000 0000 0000 0000

	symbolPositionMaskValue = uint16(0x7fff) // 0111 1111 1111 1111
)

func newSymbolPosition(n uint16, endMark bool) (symbolPosition, error) {
	if n < symbolPositionMin || n > symbolPositionMax {
		return symbolPositionNil, fmt.Errorf("symbol position must be within %v to %v; n: %v, endMark: %v", symbolPositionMin, symbolPositionMax, n, endMark)
	}
	if endMark {
		return symbolPosition(n | symbolPositionMaskEndMark), nil
	}
	return symbolPosition(n | symbolPositionMaskSymbol), nil
}

func (p symbolPosition) String() string {
	if p.isEndMark() {
		return fmt.Sprintf("end#%v", uint16(p)&symbolPositionMaskValue)
	}
	return fmt.Sprintf("sym#%v", uint16(p)&symbolPositionMaskValue)
}

func (p symbolPosition) isEndMark() bool {
	if uint16(p)&symbolPositionMaskEndMark > 1 {
		return true
	}
	return false
}

func (p symbolPosition) describe() (uint16, bool) {
	v := uint16(p) & symbolPositionMaskValue
	if p.isEndMark() {
		return v, true
	}
	return v, false
}

type symbolPositionSet map[symbolPosition]struct{}

func newSymbolPositionSet() symbolPositionSet {
	return map[symbolPosition]struct{}{}
}

func (s symbolPositionSet) String() string {
	if len(s) <= 0 {
		return "{}"
	}
	ps := s.sort()
	var b strings.Builder
	fmt.Fprintf(&b, "{")
	for i, p := range ps {
		if i <= 0 {
			fmt.Fprintf(&b, "%v", p)
			continue
		}
		fmt.Fprintf(&b, ", %v", p)
	}
	fmt.Fprintf(&b, "}")
	return b.String()
}

func (s symbolPositionSet) add(pos symbolPosition) symbolPositionSet {
	s[pos] = struct{}{}
	return s
}

func (s symbolPositionSet) merge(t symbolPositionSet) symbolPositionSet {
	for p := range t {
		s.add(p)
	}
	return s
}

func (s symbolPositionSet) intersect(set symbolPositionSet) symbolPositionSet {
	in := newSymbolPositionSet()
	for p1 := range s {
		for p2 := range set {
			if p1 != p2 {
				continue
			}
			in.add(p1)
		}
	}
	return in
}

func (s symbolPositionSet) hash() string {
	if len(s) <= 0 {
		return ""
	}
	sorted := s.sort()
	var buf []byte
	for _, p := range sorted {
		b := make([]byte, 8)
		binary.PutUvarint(b, uint64(p))
		buf = append(buf, b...)
	}
	// Convert to a string to be able to use it as a key of a map.
	// But note this byte sequence is made from values of symbol positions,
	// so this is not a well-formed UTF-8 sequence.
	return string(buf)
}

func (s symbolPositionSet) sort() []symbolPosition {
	sorted := make([]symbolPosition, len(s))
	i := 0
	for p := range s {
		sorted[i] = p
		i++
	}
	sortSymbolPositions(sorted, 0, len(sorted)-1)
	return sorted
}

// sortSymbolPositions sorts a slice of symbol positions as it uses quick sort.
func sortSymbolPositions(ps []symbolPosition, left, right int) {
	if left >= right {
		return
	}
	var pivot symbolPosition
	{
		// Use a median as a pivot.
		p1 := ps[left]
		p2 := ps[(left+right)/2]
		p3 := ps[right]
		if p1 > p2 {
			p1, p2 = p2, p1
		}
		if p2 > p3 {
			p2, p3 = p3, p2
			if p1 > p2 {
				p1, p2 = p2, p1
			}
		}
		pivot = p2
	}
	i := left
	j := right
	for i <= j {
		for ps[i] < pivot {
			i++
		}
		for ps[j] > pivot {
			j--
		}
		if i <= j {
			ps[i], ps[j] = ps[j], ps[i]
			i++
			j--
		}
	}
	sortSymbolPositions(ps, left, j)
	sortSymbolPositions(ps, i, right)
}
