package textutil

import "unicode"

// Range table for all characters in the Latin1 character set.
var Latin1 = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x0000, 0x00ff, 1},
	},
	LatinOffset: 1,
}

// Range table for alphanumeric characters (ASCII letters and digits only).
var Alphanumeric = &unicode.RangeTable{
	R16: []unicode.Range16{
		// ASCII digits 0-9
		{0x0030, 0x0039, 1},
		// ASCII uppercase letters A-Z
		{0x0041, 0x005A, 1},
		// ASCII lowercase letters a-z
		{0x0061, 0x007A, 1},
	},
	LatinOffset: 3,
}
