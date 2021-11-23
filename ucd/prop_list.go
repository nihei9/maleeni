package ucd

import "io"

type PropList struct {
	WhiteSpace []*CodePointRange
}

// ParsePropList parses the PropList.txt.
func ParsePropList(r io.Reader) (*PropList, error) {
	var ws []*CodePointRange
	p := newParser(r)
	for p.parse() {
		if len(p.fields) == 0 {
			continue
		}
		if p.fields[1].symbol() != "White_Space" {
			continue
		}
		
		cp, err := p.fields[0].codePointRange()
		if err != nil {
			return nil, err
		}
		ws = append(ws, cp)
	}
	if p.err != nil {
		return nil, p.err
	}

	return &PropList{
		WhiteSpace: ws,
	}, nil
}
