package ucd

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"io"
	"regexp"
	"strings"
)

type CodePointRange struct {
	From rune
	To   rune
}

type UnicodeData struct {
	GeneralCategory map[string][]*CodePointRange
}

func ParseUnicodeData(r io.Reader, propValAliases *PropertyValueAliases) (*UnicodeData, error) {
	gc2CPRange := map[string][]*CodePointRange{}
	lastCPTo := rune(-1)
	p := newParser(r)
	for p.parse() {
		if len(p.fields) == 0 {
			continue
		}
		cpFrom, cpTo, err := parseCodePointRange(p.fields[0])
		if err != nil {
			return nil, err
		}
		if cpFrom-lastCPTo > 1 {
			defaultGCVal := propValAliases.GeneralCategoryDefaultValue
			gc2CPRange[defaultGCVal] = append(gc2CPRange[defaultGCVal], &CodePointRange{
				From: lastCPTo + 1,
				To:   cpFrom - 1,
			})
		}
		lastCPTo = cpTo
		gc := NormalizeSymbolicValue(p.fields[2])
		rs, ok := gc2CPRange[gc]
		if ok {
			r := rs[len(rs)-1]
			if cpFrom-r.To == 1 {
				r.To = cpTo
			} else {
				gc2CPRange[gc] = append(rs, &CodePointRange{
					From: cpFrom,
					To:   cpTo,
				})
			}
		} else {
			gc2CPRange[gc] = []*CodePointRange{
				{
					From: cpFrom,
					To:   cpTo,
				},
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

type PropertyValueAliases struct {
	GeneralCategory             map[string]string
	GeneralCategoryDefaultRange *CodePointRange
	GeneralCategoryDefaultValue string
}

func ParsePropertyValueAliases(r io.Reader) (*PropertyValueAliases, error) {
	catName2Abbs := map[string]string{}
	var defaultGCCPRange *CodePointRange
	var defaultGCVal string
	p := newParser(r)
	for p.parse() {
		if len(p.fields) > 0 && p.fields[0] == "gc" {
			catNameShort := NormalizeSymbolicValue(p.fields[1])
			catNameLong := NormalizeSymbolicValue(p.fields[2])
			catName2Abbs[catNameShort] = catNameShort
			catName2Abbs[catNameLong] = catNameShort
			for _, f := range p.fields[3:] {
				catNameOther := NormalizeSymbolicValue(f)
				catName2Abbs[catNameOther] = catNameShort
			}
		}
		if len(p.defaultFields) > 0 && p.defaultFields[1] == "General_Category" {
			cpFrom, cpTo, err := parseCodePointRange(p.defaultFields[0])
			if err != nil {
				return nil, err
			}
			defaultGCCPRange = &CodePointRange{
				From: cpFrom,
				To:   cpTo,
			}
			defaultGCVal = NormalizeSymbolicValue(p.defaultFields[2])
		}
	}
	if p.err != nil {
		return nil, p.err
	}
	return &PropertyValueAliases{
		GeneralCategory:             catName2Abbs,
		GeneralCategoryDefaultRange: defaultGCCPRange,
		GeneralCategoryDefaultValue: defaultGCVal,
	}, nil
}

var symValReplacer = strings.NewReplacer("_", "", "-", "", "\x20", "")

func NormalizeSymbolicValue(original string) string {
	strings.Trim("", "")
	v := strings.ToLower(symValReplacer.Replace(original))
	if strings.HasPrefix(v, "is") && v != "is" {
		return v[3:]
	}
	return v
}

type Fields []string

var (
	reLine           = regexp.MustCompile(`^\s*(.*?)\s*(#.*)?$`)
	reCodePointRange = regexp.MustCompile(`^([[:xdigit:]]+)(?:..([[:xdigit:]]+))?$`)

	specialCommentPrefix = "# @missing:"
)

// This parser can parse data files of Unicode Character Database (UCD).
// Specifically, it has the following two functions:
// - Converts each line of the data files into a slice of fields.
// - Recognizes specially-formatted comments starting `@missing` and generates a slice of fields.
//
// However, for practical purposes, each field needs to be analyzed more specifically.
// For instance, in UnicodeData.txt, the first field represents a range of code points,
// so it needs to be recognized as a hexadecimal string.
// You can perform more specific parsing for each file by implementing a dedicated parser that wraps this parser.
//
// https://www.unicode.org/reports/tr44/#Format_Conventions
type parser struct {
	scanner       *bufio.Scanner
	fields        Fields
	defaultFields Fields
	err           error
}

func newParser(r io.Reader) *parser {
	return &parser{
		scanner: bufio.NewScanner(r),
	}
}

func (p *parser) parse() bool {
	for p.scanner.Scan() {
		p.fields, p.defaultFields, p.err = parseRecord(p.scanner.Text())
		if p.err != nil {
			return false
		}
		if p.fields != nil || p.defaultFields != nil {
			return true
		}
	}
	p.err = p.scanner.Err()
	return false
}

func parseRecord(src string) (Fields, Fields, error) {
	ms := reLine.FindStringSubmatch(src)
	fields := ms[1]
	comment := ms[2]
	var fs Fields
	if fields != "" {
		fs = parseFields(fields)
	}
	var defaultFs Fields
	if strings.HasPrefix(comment, specialCommentPrefix) {
		fields := strings.Replace(comment, specialCommentPrefix, "", -1)
		fs := parseFields(fields)
		defaultFs = fs
	}
	return fs, defaultFs, nil
}

func parseFields(src string) Fields {
	var fields Fields
	for _, f := range strings.Split(src, ";") {
		fields = append(fields, strings.TrimSpace(f))
	}
	return fields
}

func parseCodePointRange(src string) (rune, rune, error) {
	var from, to rune
	var err error
	cp := reCodePointRange.FindStringSubmatch(src)
	from, err = decodeHexToRune(cp[1])
	if err != nil {
		return 0, 0, err
	}
	if cp[2] != "" {
		to, err = decodeHexToRune(cp[2])
		if err != nil {
			return 0, 0, err
		}
	} else {
		to = from
	}
	return from, to, nil
}

func decodeHexToRune(hexCodePoint string) (rune, error) {
	h := hexCodePoint
	if len(h)%2 != 0 {
		h = "0" + h
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return 0, err
	}
	l := len(b)
	for i := 0; i < 4-l; i++ {
		b = append([]byte{0}, b...)
	}
	n := binary.BigEndian.Uint32(b)
	return rune(n), nil
}
