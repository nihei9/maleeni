package utf8

import "fmt"

type CharBlock struct {
	From []byte
	To   []byte
}

// Refelences:
// * https://www.unicode.org/versions/Unicode13.0.0/ch03.pdf#G7404
//   * Table 3-6.  UTF-8 Bit Distribution
//   * Table 3-7.  Well-Formed UTF-8 Byte Sequences
var (
	// 1 byte character:
	// * <00..7F>
	cBlks1 = []*CharBlock{
		{
			From: []byte{0x00},
			To:   []byte{0x7f},
		},
	}

	// 2 bytes character:
	// * <C2..DF 80..BF>
	cBlks2 = []*CharBlock{
		{
			From: []byte{0xc2, 0x80},
			To:   []byte{0xdf, 0xbf},
		},
	}

	// 3 bytes character:
	// * <E0 A0..BF 80..BF>
	// * <E1..EC 80..BF 80..BF>
	// * <ED 80..9F 80..BF>
	// * <EE..EF 80..BF 80..BF>
	cBlks3 = []*CharBlock{
		{
			From: []byte{0xe0, 0xa0, 0x80},
			To:   []byte{0xe0, 0xbf, 0xbf},
		},
		{
			From: []byte{0xe1, 0x80, 0x80},
			To:   []byte{0xec, 0xbf, 0xbf},
		},
		{
			From: []byte{0xed, 0x80, 0x80},
			To:   []byte{0xed, 0x9f, 0xbf},
		},
		{
			From: []byte{0xee, 0x80, 0x80},
			To:   []byte{0xef, 0xbf, 0xbf},
		},
	}

	// 4 bytes character:
	// * <F0 90..BF 80..BF 80..BF>
	// * <F1..F3 80..BF 80..BF 80..BF>
	// * <F4 80..8F 80..BF 80..BF>
	cBlks4 = []*CharBlock{
		{
			From: []byte{0xf0, 0x90, 0x80, 0x80},
			To:   []byte{0xf0, 0xbf, 0xbf, 0xbf},
		},
		{
			From: []byte{0xf1, 0x80, 0x80, 0x80},
			To:   []byte{0xf3, 0xbf, 0xbf, 0xbf},
		},
		{
			From: []byte{0xf4, 0x80, 0x80, 0x80},
			To:   []byte{0xf4, 0x8f, 0xbf, 0xbf},
		},
	}

	cBlk1Head = cBlks1[0]
	cBlk1Last = cBlks1[len(cBlks1)-1]
	cBlk2Head = cBlks2[0]
	cBlk2Last = cBlks2[len(cBlks2)-1]
	cBlk3Head = cBlks3[0]
	cBlk3Last = cBlks3[len(cBlks3)-1]
	cBlk4Head = cBlks4[0]
	cBlk4Last = cBlks4[len(cBlks4)-1]
)

func AllCharBlocks() []*CharBlock {
	var blks []*CharBlock
	blks = append(blks, cBlks1...)
	blks = append(blks, cBlks2...)
	blks = append(blks, cBlks3...)
	blks = append(blks, cBlks4...)
	return blks
}

func GenCharBlocks(from, to []byte) ([]*CharBlock, error) {
	switch len(from) {
	case 1:
		switch len(to) {
		case 1:
			return genCharBlocks1(from, to), nil
		case 2:
			var alt []*CharBlock
			alt = append(alt, genCharBlocks1(from, cBlk1Last.To)...)
			alt = append(alt, genCharBlocks2(cBlk2Head.From, to)...)
			return alt, nil
		case 3:
			var alt []*CharBlock
			alt = append(alt, genCharBlocks1(from, cBlk1Last.To)...)
			alt = append(alt, genCharBlocks2(cBlk2Head.From, cBlk2Last.To)...)
			alt = append(alt, genCharBlocks3(cBlk3Head.From, to)...)
			return alt, nil
		case 4:
			var alt []*CharBlock
			alt = append(alt, genCharBlocks1(from, cBlk1Last.To)...)
			alt = append(alt, genCharBlocks2(cBlk2Head.From, cBlk2Last.To)...)
			alt = append(alt, genCharBlocks3(cBlk3Head.From, cBlk3Last.To)...)
			alt = append(alt, genCharBlocks4(cBlk4Head.From, to)...)
			return alt, nil
		}
	case 2:
		switch len(to) {
		case 2:
			return genCharBlocks2(from, to), nil
		case 3:
			var alt []*CharBlock
			alt = append(alt, genCharBlocks2(from, cBlk2Last.To)...)
			alt = append(alt, genCharBlocks3(cBlk3Head.From, to)...)
			return alt, nil
		case 4:
			var alt []*CharBlock
			alt = append(alt, genCharBlocks2(from, cBlk2Last.To)...)
			alt = append(alt, genCharBlocks3(cBlk3Head.From, cBlk3Last.To)...)
			alt = append(alt, genCharBlocks4(cBlk4Head.From, to)...)
			return alt, nil
		}
	case 3:
		switch len(to) {
		case 3:
			return genCharBlocks3(from, to), nil
		case 4:
			var alt []*CharBlock
			alt = append(alt, genCharBlocks3(from, cBlk3Last.To)...)
			alt = append(alt, genCharBlocks4(cBlk4Head.From, to)...)
			return alt, nil
		}
	case 4:
		return genCharBlocks4(from, to), nil
	}
	return nil, fmt.Errorf("invalid range; From: %v, To: %v", from, to)
}

func genCharBlocks1(from, to []byte) []*CharBlock {
	return []*CharBlock{
		{From: from, To: to},
	}
}

func genCharBlocks2(from, to []byte) []*CharBlock {
	switch {
	case from[0] == to[0]:
		return []*CharBlock{
			{From: from, To: to},
		}
	default:
		return []*CharBlock{
			{From: from, To: []byte{from[0], cBlks2[0].To[1]}},
			{From: []byte{from[0] + 1, cBlks2[0].From[1]}, To: to},
		}
	}
}

func genCharBlocks3(from, to []byte) []*CharBlock {
	switch {
	case from[0] == to[0] && from[1] == to[1]:
		return []*CharBlock{
			{From: from, To: to},
		}
	case from[0] == to[0]:
		_, fromBlk := findCharBlock3(from)
		var alt []*CharBlock
		alt = append(alt, &CharBlock{
			From: from,
			To:   []byte{from[0], from[1], fromBlk.To[2]},
		})
		if from[1]+1 < to[1] {
			alt = append(alt, &CharBlock{
				From: []byte{from[0], from[1] + 1, fromBlk.From[2]},
				To:   []byte{from[0], to[1] - 1, fromBlk.To[2]},
			})
		}
		alt = append(alt, &CharBlock{
			From: []byte{from[0], to[1], fromBlk.From[2]},
			To:   to,
		})
		return alt
	default:
		fromBlkNum, fromBlk := findCharBlock3(from)
		toBlkNum, toBlk := findCharBlock3(to)
		var alt []*CharBlock
		alt = append(alt, &CharBlock{
			From: from,
			To:   []byte{from[0], from[1], fromBlk.To[2]},
		})
		if from[1] < fromBlk.To[1] {
			alt = append(alt, &CharBlock{
				From: []byte{from[0], from[1] + 1, fromBlk.From[2]},
				To:   []byte{from[0], fromBlk.To[1], fromBlk.To[2]},
			})
		}
		if fromBlkNum == toBlkNum {
			if from[0]+1 < to[0] {
				alt = append(alt, &CharBlock{
					From: []byte{from[0] + 1, fromBlk.From[1], fromBlk.From[2]},
					To:   []byte{to[0] - 1, fromBlk.To[1], fromBlk.To[2]},
				})
			}
			if to[1] > fromBlk.From[1] {
				alt = append(alt, &CharBlock{
					From: []byte{to[0], fromBlk.From[1], fromBlk.From[2]},
					To:   []byte{to[0], to[1] - 1, fromBlk.To[2]},
				})
			}
			alt = append(alt, &CharBlock{
				From: []byte{to[0], to[1], fromBlk.From[2]},
				To:   to,
			})
			return alt
		}
		for blkNum := fromBlkNum + 1; blkNum < toBlkNum; blkNum++ {
			fromBlk := cBlks3[blkNum]
			alt = append(alt, &CharBlock{
				From: fromBlk.From,
				To:   fromBlk.To,
			})
		}
		if to[0] > toBlk.From[0] {
			alt = append(alt, &CharBlock{
				From: toBlk.From,
				To:   []byte{to[0] - 1, toBlk.To[1], toBlk.To[2]},
			})
		}
		if to[1] > toBlk.From[1] {
			alt = append(alt, &CharBlock{
				From: []byte{to[0], toBlk.From[1], toBlk.From[2]},
				To:   []byte{to[0], to[1] - 1, toBlk.To[2]},
			})
		}
		alt = append(alt, &CharBlock{
			From: []byte{to[0], to[1], toBlk.From[2]},
			To:   to,
		})
		return alt
	}
}

func genCharBlocks4(from, to []byte) []*CharBlock {
	switch {
	case from[0] == to[0] && from[1] == to[1] && from[2] == to[2]:
		return []*CharBlock{
			{
				From: from,
				To:   to,
			},
		}
	case from[0] == to[0] && from[1] == to[1]:
		_, fromBlk := findCharBlock4(from)
		var alt []*CharBlock
		alt = append(alt, &CharBlock{
			From: from,
			To:   []byte{to[0], to[1], from[2], fromBlk.To[3]},
		})
		if from[2]+1 < to[2] {
			alt = append(alt, &CharBlock{
				From: []byte{from[0], from[1], from[2] + 2, fromBlk.From[3]},
				To:   []byte{to[0], to[1], to[2] - 1, fromBlk.To[3]},
			})
		}
		alt = append(alt, &CharBlock{
			From: []byte{from[0], from[1], to[2], fromBlk.From[3]},
			To:   []byte{to[0], to[1], to[2], to[3]},
		})
		return alt
	case from[0] == to[0]:
		_, fromBlk := findCharBlock4(from)
		var alt []*CharBlock
		alt = append(alt, &CharBlock{
			From: from,
			To:   []byte{to[0], from[1], from[2], fromBlk.To[3]},
		})
		if from[2] < fromBlk.To[2] {
			alt = append(alt, &CharBlock{
				From: []byte{from[0], from[1], from[2] + 1, fromBlk.From[3]},
				To:   []byte{to[0], from[1], fromBlk.To[2], fromBlk.To[3]},
			})
		}
		if from[1]+1 < to[1] {
			alt = append(alt, &CharBlock{
				From: []byte{from[0], from[1] + 1, fromBlk.From[2], fromBlk.From[3]},
				To:   []byte{to[0], to[1] - 1, fromBlk.To[2], fromBlk.To[3]},
			})
		}
		if to[2] > fromBlk.From[2] {
			alt = append(alt, &CharBlock{
				From: []byte{from[0], to[1], fromBlk.From[2], fromBlk.From[3]},
				To:   []byte{from[0], to[1], to[2] - 1, fromBlk.To[3]},
			})
		}
		alt = append(alt, &CharBlock{
			From: []byte{from[0], to[1], to[2], fromBlk.From[3]},
			To:   to,
		})
		return alt
	default:
		fromBlkNum, fromBlk := findCharBlock4(from)
		toBlkNum, toBlk := findCharBlock4(to)
		var alt []*CharBlock
		alt = append(alt, &CharBlock{
			From: from,
			To:   []byte{from[0], from[1], from[2], fromBlk.To[3]},
		})
		if from[2] < fromBlk.To[2] {
			alt = append(alt, &CharBlock{
				From: []byte{from[0], from[1], from[2] + 1, fromBlk.From[3]},
				To:   []byte{from[0], from[1], fromBlk.To[2], fromBlk.To[3]},
			})
		}
		if from[1] < fromBlk.To[1] {
			alt = append(alt, &CharBlock{
				From: []byte{from[0], from[1] + 1, fromBlk.From[2], fromBlk.From[3]},
				To:   []byte{from[0], fromBlk.To[1], fromBlk.To[2], fromBlk.To[3]},
			})
		}
		if fromBlkNum == toBlkNum {
			if from[0]+1 < to[0] {
				alt = append(alt, &CharBlock{
					From: []byte{from[0] + 1, fromBlk.From[1], fromBlk.From[2], fromBlk.From[3]},
					To:   []byte{from[0] - 1, fromBlk.To[1], fromBlk.To[2], fromBlk.To[3]},
				})
			}
			if to[1] > fromBlk.From[1] {
				alt = append(alt, &CharBlock{
					From: []byte{to[0], fromBlk.From[1], fromBlk.From[2], fromBlk.From[3]},
					To:   []byte{to[0], to[1] - 1, fromBlk.To[2], fromBlk.To[3]},
				})
			}
			if to[2] > fromBlk.From[2] {
				alt = append(alt, &CharBlock{
					From: []byte{to[0], to[1], fromBlk.From[2], fromBlk.From[3]},
					To:   []byte{to[0], to[1], to[2] - 1, fromBlk.To[3]},
				})
			}
			alt = append(alt, &CharBlock{
				From: []byte{to[0], to[1], to[2], fromBlk.From[3]},
				To:   to,
			})
			return alt
		}
		for blkNum := fromBlkNum + 1; blkNum < toBlkNum; blkNum++ {
			blk := cBlks4[blkNum]
			alt = append(alt, &CharBlock{
				From: blk.From,
				To:   blk.To,
			})
		}
		if to[0] > toBlk.From[0] {
			alt = append(alt, &CharBlock{
				From: toBlk.From,
				To:   []byte{to[0] - 1, toBlk.To[1], toBlk.To[2], toBlk.To[3]},
			})
		}
		if to[1] > toBlk.From[1] {
			alt = append(alt, &CharBlock{
				From: []byte{to[0], toBlk.From[1], toBlk.From[2], toBlk.From[3]},
				To:   []byte{to[0], to[1] - 1, toBlk.To[2], toBlk.To[3]},
			})
		}
		if to[2] > toBlk.From[2] {
			alt = append(alt, &CharBlock{
				From: []byte{to[0], to[1], toBlk.From[2], toBlk.From[3]},
				To:   []byte{to[0], to[1], to[2] - 1, toBlk.To[3]},
			})
		}
		alt = append(alt, &CharBlock{
			From: []byte{to[0], to[1], to[2], toBlk.From[3]},
			To:   to,
		})
		return alt
	}
}

func findCharBlock3(c []byte) (int, *CharBlock) {
	for i, blk := range cBlks3 {
		if c[0] >= blk.From[0] && c[0] <= blk.To[0] {
			return i, blk
		}
	}
	return 0, nil
}

func findCharBlock4(c []byte) (int, *CharBlock) {
	for i, blk := range cBlks4 {
		if c[0] >= blk.From[0] && c[0] <= blk.To[0] {
			return i, blk
		}
	}
	return 0, nil
}
