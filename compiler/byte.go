package compiler

import "fmt"

type byteRange struct {
	from byte
	to   byte
}

func newByteRange(from, to byte) byteRange {
	return byteRange{
		from: from,
		to:   to,
	}
}

func (r byteRange) String() string {
	return fmt.Sprintf("%X-%X (%v-%v)", r.from, r.to, r.from, r.to)
}

func newByteRangeSequence(from, to []byte) ([]byteRange, error) {
	if len(from) != len(to) {
		return nil, fmt.Errorf("length of `from` and `to` are mismatched; from: %+v, to: %+v", from, to)
	}
	seq := []byteRange{}
	for i := 0; i < len(from); i++ {
		seq = append(seq, newByteRange(from[i], to[i]))
	}
	return seq, nil
}

func excludeByteRangeSequence(b1, b2 []byteRange) [][]byteRange {
	if len(b1) != len(b2) {
		// no overlapping
		return [][]byteRange{b2}
	}
	switch len(b1) {
	case 1:
		return exclude1Byte(b1, b2)
	case 2:
		return exclude2Byte(b1, b2)
	case 3:
		return exclude3Byte(b1, b2)
	case 4:
		return exclude4Byte(b1, b2)
	}
	panic(fmt.Errorf("excludeByteRangeSequence can only handle sequences up to 4 bytes in size; b1: %+v, b2: %+v", b1, b2))
}

func exclude1Byte(b1, b2 []byteRange) [][]byteRange {
	r01, r02, overlapping0 := excludeByteRange(&b1[0], &b2[0])
	if !overlapping0 {
		// no overlapping
		return [][]byteRange{b2}
	}
	result := [][]byteRange{}
	{
		if r01 != nil {
			result = append(result, []byteRange{
				newByteRange(r01.from, r01.to),
			})
		}
		if r02 != nil {
			result = append(result, []byteRange{
				newByteRange(r02.from, r02.to),
			})
		}
	}
	return result
}

func exclude2Byte(b1, b2 []byteRange) [][]byteRange {
	r01, r02, overlapping0 := excludeByteRange(&b1[0], &b2[0])
	r11, r12, overlapping1 := excludeByteRange(&b1[1], &b2[1])
	if !overlapping0 || !overlapping1 {
		// no overlapping
		return [][]byteRange{b2}
	}
	result := [][]byteRange{}
	{
		if r11 != nil {
			result = append(result, []byteRange{
				newByteRange(b1[0].from, b1[0].to),
				newByteRange(r11.from, r11.to),
			})
		}
		if r12 != nil {
			result = append(result, []byteRange{
				newByteRange(b1[0].from, b1[0].to),
				newByteRange(r12.from, r12.to),
			})
		}
		if r01 != nil {
			result = append(result, []byteRange{
				newByteRange(r01.from, r01.to),
				newByteRange(b2[1].from, b2[1].to),
			})
		}
		if r02 != nil {
			result = append(result, []byteRange{
				newByteRange(r02.from, r02.to),
				newByteRange(b2[1].from, b2[1].to),
			})
		}
	}
	return result
}

func exclude3Byte(b1, b2 []byteRange) [][]byteRange {
	r01, r02, overlapping0 := excludeByteRange(&b1[0], &b2[0])
	r11, r12, overlapping1 := excludeByteRange(&b1[1], &b2[1])
	r21, r22, overlapping2 := excludeByteRange(&b1[2], &b2[2])
	if !overlapping0 || !overlapping1 || !overlapping2 {
		// no overlapping
		return [][]byteRange{b2}
	}
	result := [][]byteRange{}
	{
		if r21 != nil {
			result = append(result, []byteRange{
				newByteRange(b1[0].from, b1[0].to),
				newByteRange(b1[1].from, b1[1].to),
				newByteRange(r21.from, r21.to),
			})
		}
		if r22 != nil {
			result = append(result, []byteRange{
				newByteRange(b1[0].from, b1[0].to),
				newByteRange(b1[1].from, b1[1].to),
				newByteRange(r22.from, r22.to),
			})
		}
		if r11 != nil {
			result = append(result, []byteRange{
				newByteRange(b1[0].from, b1[0].to),
				newByteRange(r11.from, r11.to),
				newByteRange(b2[2].from, b2[2].to),
			})
		}
		if r12 != nil {
			result = append(result, []byteRange{
				newByteRange(b1[0].from, b1[0].to),
				newByteRange(r12.from, r12.to),
				newByteRange(b2[2].from, b2[2].to),
			})
		}
		if r01 != nil {
			result = append(result, []byteRange{
				newByteRange(r01.from, r01.to),
				newByteRange(b2[1].from, b2[1].to),
				newByteRange(b2[2].from, b2[2].to),
			})
		}
		if r02 != nil {
			result = append(result, []byteRange{
				newByteRange(r02.from, r02.to),
				newByteRange(b2[1].from, b2[1].to),
				newByteRange(b2[2].from, b2[2].to),
			})
		}
	}
	return result
}

func exclude4Byte(b1, b2 []byteRange) [][]byteRange {
	r01, r02, overlapping0 := excludeByteRange(&b1[0], &b2[0])
	r11, r12, overlapping1 := excludeByteRange(&b1[1], &b2[1])
	r21, r22, overlapping2 := excludeByteRange(&b1[2], &b2[2])
	r31, r32, overlapping3 := excludeByteRange(&b1[3], &b2[3])
	if !overlapping0 || !overlapping1 || !overlapping2 || !overlapping3 {
		// no overlapping
		return [][]byteRange{b2}
	}
	result := [][]byteRange{}
	{
		if r31 != nil {
			result = append(result, []byteRange{
				newByteRange(b1[0].from, b1[0].to),
				newByteRange(b1[1].from, b1[1].to),
				newByteRange(b1[2].from, b1[2].to),
				newByteRange(r31.from, r31.to),
			})
		}
		if r32 != nil {
			result = append(result, []byteRange{
				newByteRange(b1[0].from, b1[0].to),
				newByteRange(b1[1].from, b1[1].to),
				newByteRange(b1[2].from, b1[2].to),
				newByteRange(r32.from, r32.to),
			})
		}
		if r21 != nil {
			result = append(result, []byteRange{
				newByteRange(b1[0].from, b1[0].to),
				newByteRange(b1[1].from, b1[1].to),
				newByteRange(r21.from, r21.to),
				newByteRange(b2[3].from, b2[3].to),
			})
		}
		if r22 != nil {
			result = append(result, []byteRange{
				newByteRange(b1[0].from, b1[0].to),
				newByteRange(b1[1].from, b1[1].to),
				newByteRange(r22.from, r22.to),
				newByteRange(b2[3].from, b2[3].to),
			})
		}
		if r11 != nil {
			result = append(result, []byteRange{
				newByteRange(b1[0].from, b1[0].to),
				newByteRange(r11.from, r11.to),
				newByteRange(b2[2].from, b2[2].to),
				newByteRange(b2[3].from, b2[3].to),
			})
		}
		if r12 != nil {
			result = append(result, []byteRange{
				newByteRange(b1[0].from, b1[0].to),
				newByteRange(r12.from, r12.to),
				newByteRange(b2[2].from, b2[2].to),
				newByteRange(b2[3].from, b2[3].to),
			})
		}
		if r01 != nil {
			result = append(result, []byteRange{
				newByteRange(r01.from, r01.to),
				newByteRange(b2[1].from, b2[1].to),
				newByteRange(b2[2].from, b2[2].to),
				newByteRange(b2[3].from, b2[3].to),
			})
		}
		if r02 != nil {
			result = append(result, []byteRange{
				newByteRange(r02.from, r02.to),
				newByteRange(b2[1].from, b2[1].to),
				newByteRange(b2[2].from, b2[2].to),
				newByteRange(b2[3].from, b2[3].to),
			})
		}
	}
	return result
}

// excludeByteRange excludes `r1` from `r2`.
func excludeByteRange(r1, r2 *byteRange) (*byteRange, *byteRange, bool) {
	if r1.from <= r2.from {
		if r1.to < r2.from {
			// no overlapping
			return nil, nil, false
		}
		if r1.to < r2.to {
			// The beginning of `r2` overlaps with `r1`.
			return &byteRange{
				from: r1.to + 1,
				to:   r2.to,
			}, nil, true
		}
		// `symbol` overlaps with `base` entirely.
		return nil, nil, true
	}
	if r1.from > r2.to {
		// no overlapping
		return nil, nil, false
	}
	if r1.to >= r2.to {
		// The end of `r2` overlaps with `r1`.
		return &byteRange{
			from: r2.from,
			to:   r1.from - 1,
		}, nil, true
	}
	// `r2` overlaps  with `r1` entirely.
	return &byteRange{
			from: r2.from,
			to:   r1.from - 1,
		},
		&byteRange{
			from: r1.to + 1,
			to:   r2.to,
		}, true
}
