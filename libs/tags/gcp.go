package tags

import (
	"regexp"
	"unicode"
)

// Tag keys and values on GCP are limited to 63 characters and must match the
// regular expression `^([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]$`.
// For normalization, we define one table for the outer characters and
// one table for the inner characters. The outer table is used to trim
// leading and trailing characters, and the inner table is used to
// replace invalid characters with underscores.

var gcpOuter = &unicode.RangeTable{
	R16: []unicode.Range16{
		// 0-9
		{0x0030, 0x0039, 1},
		// A-Z
		{0x0041, 0x005A, 1},
		// a-z
		{0x0061, 0x007A, 1},
	},
	LatinOffset: 3,
}

var gcpInner = &unicode.RangeTable{
	R16: []unicode.Range16{
		// Hyphen-minus (dash)
		{0x002D, 0x002D, 1},
		// Full stop (period)
		{0x002E, 0x002E, 1},
		// 0-9
		{0x0030, 0x0039, 1},
		// A-Z
		{0x0041, 0x005A, 1},
		// Low line (underscore)
		{0x005F, 0x005F, 1},
		// a-z
		{0x0061, 0x007A, 1},
	},
	LatinOffset: 6,
}

var gcpTag = &tag{
	keyLength:  63,
	keyPattern: regexp.MustCompile(`^([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]$`),
	keyNormalize: chain(
		normalizeMarks(),
		replaceNotIn(latin1, '_'),
		replaceNotIn(gcpInner, '_'),
		trimIfNotIn(gcpOuter),
	),

	valueLength:  63,
	valuePattern: regexp.MustCompile(`^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$`),
	valueNormalize: chain(
		normalizeMarks(),
		replaceNotIn(latin1, '_'),
		replaceNotIn(gcpInner, '_'),
		trimIfNotIn(gcpOuter),
	),
}
