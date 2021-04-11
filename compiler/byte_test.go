package compiler

import "testing"

func TestExcludeByteRangeSequence(t *testing.T) {
	tests := []struct {
		b1       []byteRange
		b2       []byteRange
		excluded [][]byteRange
	}{
		// 1 Byte
		{
			b1: newByteRangeSeq([]byte{0x00}, []byte{0x00}),
			b2: newByteRangeSeq([]byte{0x00}, []byte{0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x01}, []byte{0xff}),
			},
		},
		{
			b1: newByteRangeSeq([]byte{0xff}, []byte{0xff}),
			b2: newByteRangeSeq([]byte{0x00}, []byte{0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00}, []byte{0xfe}),
			},
		},
		{
			b1: newByteRangeSeq([]byte{0xc0}, []byte{0xcf}),
			b2: newByteRangeSeq([]byte{0x00}, []byte{0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00}, []byte{0xbf}),
				newByteRangeSeq([]byte{0xd0}, []byte{0xff}),
			},
		},
		{
			b1:       newByteRangeSeq([]byte{0x00}, []byte{0xff}),
			b2:       newByteRangeSeq([]byte{0xc0}, []byte{0xcf}),
			excluded: nil,
		},
		{
			b1:       newByteRangeSeq([]byte{0x00}, []byte{0xff}),
			b2:       newByteRangeSeq([]byte{0x00}, []byte{0xff}),
			excluded: nil,
		},

		// 2 Byte
		{
			b1: newByteRangeSeq([]byte{0x00, 0x00}, []byte{0x00, 0x00}),
			b2: newByteRangeSeq([]byte{0x00, 0x00}, []byte{0xff, 0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00, 0x01}, []byte{0x00, 0xff}),
				newByteRangeSeq([]byte{0x01, 0x00}, []byte{0xff, 0xff}),
			},
		},
		{
			b1: newByteRangeSeq([]byte{0xff, 0xff}, []byte{0xff, 0xff}),
			b2: newByteRangeSeq([]byte{0x00, 0x00}, []byte{0xff, 0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00, 0x00}, []byte{0xfe, 0xff}),
				newByteRangeSeq([]byte{0xff, 0x00}, []byte{0xff, 0xfe}),
			},
		},
		{
			b1: newByteRangeSeq([]byte{0xc0, 0xc0}, []byte{0xc0, 0xc0}),
			b2: newByteRangeSeq([]byte{0x00, 0x00}, []byte{0xff, 0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00, 0x00}, []byte{0xbf, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0x00}, []byte{0xc0, 0xbf}),
				newByteRangeSeq([]byte{0xc0, 0xc1}, []byte{0xc0, 0xff}),
				newByteRangeSeq([]byte{0xc1, 0x00}, []byte{0xff, 0xff}),
			},
		},
		{
			b1: newByteRangeSeq([]byte{0xc0, 0x00}, []byte{0xc0, 0xff}),
			b2: newByteRangeSeq([]byte{0x00, 0x00}, []byte{0xff, 0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00, 0x00}, []byte{0xbf, 0xff}),
				newByteRangeSeq([]byte{0xc1, 0x00}, []byte{0xff, 0xff}),
			},
		},
		{
			b1:       newByteRangeSeq([]byte{0x00, 0x00}, []byte{0xff, 0xff}),
			b2:       newByteRangeSeq([]byte{0xc0, 0xc0}, []byte{0xc0, 0xc0}),
			excluded: nil,
		},
		{
			b1:       newByteRangeSeq([]byte{0x00, 0x00}, []byte{0xff, 0xff}),
			b2:       newByteRangeSeq([]byte{0x00, 0x00}, []byte{0xff, 0xff}),
			excluded: nil,
		},

		// 3 Byte
		{
			b1: newByteRangeSeq([]byte{0x00, 0x00, 0x00}, []byte{0x00, 0x00, 0x00}),
			b2: newByteRangeSeq([]byte{0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00, 0x00, 0x01}, []byte{0x00, 0x00, 0xff}),
				newByteRangeSeq([]byte{0x00, 0x01, 0x00}, []byte{0x00, 0xff, 0xff}),
				newByteRangeSeq([]byte{0x01, 0x00, 0x00}, []byte{0xff, 0xff, 0xff}),
			},
		},
		{
			b1: newByteRangeSeq([]byte{0xff, 0xff, 0xff}, []byte{0xff, 0xff, 0xff}),
			b2: newByteRangeSeq([]byte{0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00, 0x00, 0x00}, []byte{0xfe, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xff, 0x00, 0x00}, []byte{0xff, 0xfe, 0xff}),
				newByteRangeSeq([]byte{0xff, 0xff, 0x00}, []byte{0xff, 0xff, 0xfe}),
			},
		},
		{
			b1: newByteRangeSeq([]byte{0xc0, 0xc0, 0xc0}, []byte{0xc0, 0xc0, 0xc0}),
			b2: newByteRangeSeq([]byte{0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00, 0x00, 0x00}, []byte{0xbf, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0x00, 0x00}, []byte{0xc0, 0xbf, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0xc0, 0x00}, []byte{0xc0, 0xc0, 0xbf}),
				newByteRangeSeq([]byte{0xc0, 0xc0, 0xc1}, []byte{0xc0, 0xc0, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0xc1, 0x00}, []byte{0xc0, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc1, 0x00, 0x00}, []byte{0xff, 0xff, 0xff}),
			},
		},
		{
			b1: newByteRangeSeq([]byte{0xc0, 0xc0, 0x00}, []byte{0xc0, 0xc0, 0xff}),
			b2: newByteRangeSeq([]byte{0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00, 0x00, 0x00}, []byte{0xbf, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0x00, 0x00}, []byte{0xc0, 0xbf, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0xc1, 0x00}, []byte{0xc0, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc1, 0x00, 0x00}, []byte{0xff, 0xff, 0xff}),
			},
		},
		{
			b1: newByteRangeSeq([]byte{0xc0, 0x00, 0x00}, []byte{0xc0, 0xff, 0xff}),
			b2: newByteRangeSeq([]byte{0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00, 0x00, 0x00}, []byte{0xbf, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc1, 0x00, 0x00}, []byte{0xff, 0xff, 0xff}),
			},
		},
		{
			b1:       newByteRangeSeq([]byte{0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff}),
			b2:       newByteRangeSeq([]byte{0xff, 0xff, 0xff}, []byte{0xff, 0xff, 0xff}),
			excluded: nil,
		},
		{
			b1:       newByteRangeSeq([]byte{0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff}),
			b2:       newByteRangeSeq([]byte{0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff}),
			excluded: nil,
		},

		// 4 Byte
		{
			b1: newByteRangeSeq([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0x00, 0x00, 0x00, 0x00}),
			b2: newByteRangeSeq([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff, 0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00, 0x00, 0x00, 0x01}, []byte{0x00, 0x00, 0x00, 0xff}),
				newByteRangeSeq([]byte{0x00, 0x00, 0x01, 0x00}, []byte{0x00, 0x00, 0xff, 0xff}),
				newByteRangeSeq([]byte{0x00, 0x01, 0x00, 0x00}, []byte{0x00, 0xff, 0xff, 0xff}),
				newByteRangeSeq([]byte{0x01, 0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff, 0xff}),
			},
		},
		{
			b1: newByteRangeSeq([]byte{0xff, 0xff, 0xff, 0xff}, []byte{0xff, 0xff, 0xff, 0xff}),
			b2: newByteRangeSeq([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff, 0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0xfe, 0xff, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xff, 0x00, 0x00, 0x00}, []byte{0xff, 0xfe, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xff, 0xff, 0x00, 0x00}, []byte{0xff, 0xff, 0xfe, 0xff}),
				newByteRangeSeq([]byte{0xff, 0xff, 0xff, 0x00}, []byte{0xff, 0xff, 0xff, 0xfe}),
			},
		},
		{
			b1: newByteRangeSeq([]byte{0xc0, 0xc0, 0xc0, 0xc0}, []byte{0xc0, 0xc0, 0xc0, 0xc0}),
			b2: newByteRangeSeq([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff, 0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0xbf, 0xff, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0x00, 0x00, 0x00}, []byte{0xc0, 0xbf, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0xc0, 0x00, 0x00}, []byte{0xc0, 0xc0, 0xbf, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0xc0, 0xc0, 0x00}, []byte{0xc0, 0xc0, 0xc0, 0xbf}),
				newByteRangeSeq([]byte{0xc0, 0xc0, 0xc0, 0xc1}, []byte{0xc0, 0xc0, 0xc0, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0xc0, 0xc1, 0x00}, []byte{0xc0, 0xc0, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0xc1, 0x00, 0x00}, []byte{0xc0, 0xff, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc1, 0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff, 0xff}),
			},
		},
		{
			b1: newByteRangeSeq([]byte{0xc0, 0xc0, 0xc0, 0x00}, []byte{0xc0, 0xc0, 0xc0, 0xff}),
			b2: newByteRangeSeq([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff, 0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0xbf, 0xff, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0x00, 0x00, 0x00}, []byte{0xc0, 0xbf, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0xc0, 0x00, 0x00}, []byte{0xc0, 0xc0, 0xbf, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0xc0, 0xc1, 0x00}, []byte{0xc0, 0xc0, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0xc1, 0x00, 0x00}, []byte{0xc0, 0xff, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc1, 0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff, 0xff}),
			},
		},
		{
			b1: newByteRangeSeq([]byte{0xc0, 0xc0, 0x00, 0x00}, []byte{0xc0, 0xc0, 0xff, 0xff}),
			b2: newByteRangeSeq([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff, 0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0xbf, 0xff, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0x00, 0x00, 0x00}, []byte{0xc0, 0xbf, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc0, 0xc1, 0x00, 0x00}, []byte{0xc0, 0xff, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc1, 0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff, 0xff}),
			},
		},
		{
			b1: newByteRangeSeq([]byte{0xc0, 0x00, 0x00, 0x00}, []byte{0xc0, 0xff, 0xff, 0xff}),
			b2: newByteRangeSeq([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff, 0xff}),
			excluded: [][]byteRange{
				newByteRangeSeq([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0xbf, 0xff, 0xff, 0xff}),
				newByteRangeSeq([]byte{0xc1, 0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff, 0xff}),
			},
		},
		{
			b1:       newByteRangeSeq([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff, 0xff}),
			b2:       newByteRangeSeq([]byte{0xc0, 0xc0, 0xc0, 0xc0}, []byte{0xc0, 0xc0, 0xc0, 0xc0}),
			excluded: nil,
		},
		{
			b1:       newByteRangeSeq([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff, 0xff}),
			b2:       newByteRangeSeq([]byte{0x00, 0x00, 0x00, 0x00}, []byte{0xff, 0xff, 0xff, 0xff}),
			excluded: nil,
		},
	}
	for _, tt := range tests {
		excluded := excludeByteRangeSequence(tt.b1, tt.b2)
		t.Logf("b1: %v, b2: %v", tt.b1, tt.b2)
		t.Logf("excluded: %+v", excluded)
		if len(excluded) != len(tt.excluded) {
			t.Errorf("unexpected results; expected: %+v, actual: %+v", tt.excluded, excluded)
		}
		for _, expectedSeq := range tt.excluded {
			found := false
			for _, actualSeq := range excluded {
				mismatched := false
				for i := 0; i < len(expectedSeq); i++ {
					if actualSeq[i] != expectedSeq[i] {
						mismatched = true
						break
					}
				}
				if mismatched {
					continue
				}
				found = true
				break
			}
			if !found {
				t.Errorf("an expected byte range sequence was not found: %+v", expectedSeq)
			}
		}
	}
}

func newByteRangeSeq(from, to []byte) []byteRange {
	seq, err := newByteRangeSequence(from, to)
	if err != nil {
		panic(err)
	}
	return seq
}