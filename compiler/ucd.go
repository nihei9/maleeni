//go:generate go run ../cmd/generator/main.go
//go:generate go fmt ucd_table.go

package compiler

import (
	"fmt"

	"github.com/nihei9/maleeni/ucd"
)

func findCodePointRanges(propName, propVal string) ([]*ucd.CodePointRange, error) {
	name := ucd.NormalizeSymbolicValue(propName)
	val := ucd.NormalizeSymbolicValue(propVal)
	name, ok := propertyNameAbbs[name]
	if !ok {
		return nil, fmt.Errorf("unsupported character property: %v", propName)
	}
	val, ok = generalCategoryValueAbbs[val]
	if !ok {
		return nil, fmt.Errorf("unsupported character property value: %v", val)
	}
	vals, ok := compositGeneralCategories[val]
	if !ok {
		vals = []string{val}
	}
	var ranges []*ucd.CodePointRange
	for _, v := range vals {
		rs, ok := generalCategoryCodePoints[v]
		if !ok {
			return nil, fmt.Errorf("invalie value of the General_Category property: %v", v)
		}
		ranges = append(ranges, rs...)
	}
	return ranges, nil
}
