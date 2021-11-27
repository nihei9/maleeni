package ucd

// https://www.unicode.org/reports/tr44/#GC_Values_Table
var compositGeneralCategories = map[string][]string{
	// Cased_Letter
	"lc": {"lu", "ll", "lt"},
	// Letter
	"l": {"lu", "ll", "lt", "lm", "lo"},
	// Mark
	"m": {"mm", "mc", "me"},
	// Number
	"n": {"nd", "nl", "no"},
	// Punctuation
	"p": {"pc", "pd", "ps", "pi", "pe", "pf", "po"},
	// Symbol
	"s": {"sm", "sc", "sk", "so"},
	// Separator
	"z": {"zs", "zl", "zp"},
	// Other
	"c": {"cc", "cf", "cs", "co", "cn"},
}

// https://www.unicode.org/Public/13.0.0/ucd/DerivedCoreProperties.txt
var derivedCoreProperties = map[string][]string{
	// Alphabetic
	"alpha": {
		`\p{Lowercase=yes}`,
		`\p{Uppercase=yes}`,
		`\p{Lt}`,
		`\p{Lm}`,
		`\p{Lo}`,
		`\p{Nl}`,
		`\p{Other_Alphabetic=yes}`,
	},
	// Lowercase
	"lower": {
		`\p{Ll}`,
		`\p{Other_Lowercase=yes}`,
	},
	// Uppercase
	"upper": {
		`\p{Lu}`,
		`\p{Other_Uppercase=yes}`,
	},
}

// https://www.unicode.org/Public/13.0.0/ucd/PropertyAliases.txt
var propertyNameAbbs = map[string]string{
	"generalcategory": "gc",
	"gc":              "gc",
	"alphabetic":      "alpha",
	"alpha":           "alpha",
	"otheralphabetic": "oalpha",
	"oalpha":          "oalpha",
	"lowercase":       "lower",
	"lower":           "lower",
	"uppercase":       "upper",
	"upper":           "upper",
	"otherlowercase":  "olower",
	"olower":          "olower",
	"otheruppercase":  "oupper",
	"oupper":          "oupper",
	"whitespace":      "wspace",
	"wspace":          "wspace",
	"space":           "wspace",
}

// https://www.unicode.org/reports/tr44/#Type_Key_Table
// https://www.unicode.org/reports/tr44/#Binary_Values_Table
var binaryValues = map[string]bool{
	"yes":   true,
	"y":     true,
	"true":  true,
	"t":     true,
	"no":    false,
	"n":     false,
	"false": false,
	"f":     false,
}
