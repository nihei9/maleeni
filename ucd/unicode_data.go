package ucd

import "io"

type UnicodeData struct {
	GeneralCategory map[string][]*CodePointRange
}

// ParseUnicodeData parses the UnicodeData.txt.
func ParseUnicodeData(r io.Reader, propValAliases *PropertyValueAliases) (*UnicodeData, error) {
	gc2CPRange := map[string][]*CodePointRange{}
	lastCPTo := rune(-1)
	p := newParser(r)
	for p.parse() {
		if len(p.fields) == 0 {
			continue
		}
		cpRange, err := p.fields[0].codePointRange()
		if err != nil {
			return nil, err
		}
		if cpRange.From-lastCPTo > 1 {
			defaultGCVal := propValAliases.GeneralCategoryDefaultValue
			gc2CPRange[defaultGCVal] = append(gc2CPRange[defaultGCVal], &CodePointRange{
				From: lastCPTo + 1,
				To:   cpRange.From - 1,
			})
		}
		lastCPTo = cpRange.To
		gc := p.fields[2].normalizedSymbol()
		if gc == "" {
			// https://www.unicode.org/reports/tr44/#Empty_Fields
			// > The data file UnicodeData.txt defines many property values in each record. When a field in a data
			// > line for a code point is empty, that indicates that the property takes the default value for that
			// > code point.
			if cpRange.From < propValAliases.GeneralCategoryDefaultRange.From || cpRange.To > propValAliases.GeneralCategoryDefaultRange.To {
				continue
			}
			gc = propValAliases.GeneralCategoryDefaultValue
		}
		rs, ok := gc2CPRange[gc]
		if ok {
			r := rs[len(rs)-1]
			if cpRange.From-r.To == 1 {
				r.To = cpRange.To
			} else {
				gc2CPRange[gc] = append(rs, cpRange)
			}
		} else {
			gc2CPRange[gc] = []*CodePointRange{
				cpRange,
			}
		}
	}
	if p.err != nil {
		return nil, p.err
	}
	if lastCPTo < propValAliases.GeneralCategoryDefaultRange.To {
		defaultGCVal := propValAliases.GeneralCategoryDefaultValue
		gc2CPRange[defaultGCVal] = append(gc2CPRange[defaultGCVal], &CodePointRange{
			From: lastCPTo + 1,
			To:   propValAliases.GeneralCategoryDefaultRange.To,
		})
	}
	return &UnicodeData{
		GeneralCategory: gc2CPRange,
	}, nil
}
