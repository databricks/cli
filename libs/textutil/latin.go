package textutil

import "unicode"

// Range table for all characters in the Latin1 character set.
var Latin1 = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x0000, 0x00ff, 1},
	},
	LatinOffset: 1,
}
