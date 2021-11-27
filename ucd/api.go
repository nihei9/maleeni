//go:generate go run ../cmd/generator/main.go
//go:generate go fmt codepoint.go

package ucd

import (
	"fmt"
	"strings"
)

func NormalizeCharacterProperty(propName, propVal string) (string, error) {
	if propName == "" {
		propName = "gc"
	}
	
	name, ok := propertyNameAbbs[normalizeSymbolicValue(propName)]
	if !ok {
		return "", fmt.Errorf("unsupported character property name: %v", propName)
	}
	props, ok := derivedCoreProperties[name]
	if !ok {
		return "", nil
	}
	var b strings.Builder
	yes, ok := binaryValues[normalizeSymbolicValue(propVal)]
	if !ok {
		return "", fmt.Errorf("unsupported character property value: %v", propVal)
	}
	if yes {
		fmt.Fprint(&b, "[")
	} else {
		fmt.Fprint(&b, "[^")
	}
	for _, prop := range props {
		fmt.Fprint(&b, prop)
	}
	fmt.Fprint(&b, "]")

	return b.String(), nil
}

func FindCodePointRanges(propName, propVal string) ([]*CodePointRange, bool, error) {
	if propName == "" {
		propName = "gc"
	}

	name, ok := propertyNameAbbs[normalizeSymbolicValue(propName)]
	if !ok {
		return nil, false, fmt.Errorf("unsupported character property name: %v", propName)
	}
	switch name {
	case "gc":
		val, ok := generalCategoryValueAbbs[normalizeSymbolicValue(propVal)]
		if !ok {
			return nil, false, fmt.Errorf("unsupported character property value: %v", propVal)
		}
		vals, ok := compositGeneralCategories[val]
		if !ok {
			vals = []string{val}
		}
		var ranges []*CodePointRange
		for _, v := range vals {
			rs, ok := generalCategoryCodePoints[v]
			if !ok {
				return nil, false, fmt.Errorf("invalid value of the General_Category property: %v", v)
			}
			ranges = append(ranges, rs...)
		}
		return ranges, false, nil
	case "oalpha":
		yes, ok := binaryValues[normalizeSymbolicValue(propVal)]
		if !ok {
			return nil, false, fmt.Errorf("unsupported character property value: %v", propVal)
		}
		if yes {
			return otherAlphabeticCodePoints, false, nil
		} else {
			return otherAlphabeticCodePoints, true, nil
		}
	case "olower":
		yes, ok := binaryValues[normalizeSymbolicValue(propVal)]
		if !ok {
			return nil, false, fmt.Errorf("unsupported character property value: %v", propVal)
		}
		if yes {
			return otherLowercaseCodePoints, false, nil
		} else {
			return otherLowercaseCodePoints, true, nil
		}
	case "oupper":
		yes, ok := binaryValues[normalizeSymbolicValue(propVal)]
		if !ok {
			return nil, false, fmt.Errorf("unsupported character property value: %v", propVal)
		}
		if yes {
			return otherUppercaseCodePoints, false, nil
		} else {
			return otherUppercaseCodePoints, true, nil
		}
	case "wspace":
		yes, ok := binaryValues[normalizeSymbolicValue(propVal)]
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
