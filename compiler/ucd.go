//go:generate go run ../cmd/generator/main.go
//go:generate go fmt ucd_table.go

package compiler

import (
	"fmt"

	"github.com/nihei9/maleeni/ucd"
)

func findCodePointRanges(propName, propVal string) ([]*ucd.CodePointRange, bool, error) {
	name, ok := propertyNameAbbs[ucd.NormalizeSymbolicValue(propName)]
	if !ok {
		return nil, false, fmt.Errorf("unsupported character property name: %v", propName)
	}
	switch name {
	case "gc":
		val, ok := generalCategoryValueAbbs[ucd.NormalizeSymbolicValue(propVal)]
		if !ok {
			return nil, false, fmt.Errorf("unsupported character property value: %v", propVal)
		}
		vals, ok := compositGeneralCategories[val]
		if !ok {
			vals = []string{val}
		}
		var ranges []*ucd.CodePointRange
		for _, v := range vals {
			rs, ok := generalCategoryCodePoints[v]
			if !ok {
				return nil, false, fmt.Errorf("invalid value of the General_Category property: %v", v)
			}
			ranges = append(ranges, rs...)
		}
		return ranges, false, nil
	case "wspace":
		yes, ok := binaryValues[ucd.NormalizeSymbolicValue(propVal)]
		if !ok {
			return nil, false, fmt.Errorf("unsupported character property value: %v", propVal)
		}
		if yes {
			return whiteSpaceCodePoints, false, nil
		} else {
			return whiteSpaceCodePoints, true, nil
		}
	}

	// If the process reaches this code, it's a bug. We must handle all of the properties registered with
	// the `propertyNameAbbs`.
	return nil, false, fmt.Errorf("character property '%v' is unavailable", propName)
}
