package patchwheel

import (
	"strconv"
	"unicode"
)

// compareVersion compares version strings a and b.
// Returns:
//
//	 1 if a > b
//	-1 if a < b
//	 0 if equal.
//
// The algorithm splits each string into consecutive numeric and non-numeric tokens and compares them
// pair-wise:
//   - Numeric tokens are compared as integers.
//   - Non-numeric tokens are compared lexicographically.
//   - Numeric tokens are considered greater than non-numeric tokens when types differ.
//
// Missing tokens are treated as zero-length strings / 0.
func compareVersion(a, b string) int {
	ta := tokenizeVersion(a)
	tb := tokenizeVersion(b)

	minLen := min(len(ta), len(tb))

	for i, tokA := range ta[:minLen] {
		tokB := tb[i]

		if tokA.numeric && tokB.numeric {
			intA, _ := strconv.Atoi(tokA.value)
			intB, _ := strconv.Atoi(tokB.value)
			if intA != intB {
				if intA > intB {
					return 1
				}
				return -1
			}
			continue
		}

		if tokA.value != tokB.value {
			if tokA.value > tokB.value {
				return 1
			}
			return -1
		}
	}

	// All shared tokens are equal; the longer version wins if it has extra tokens.
	if len(ta) > len(tb) {
		return 1
	}
	if len(tb) > len(ta) {
		return -1
	}
	return 0
}

type versionToken struct {
	numeric bool
	value   string
}

func tokenizeVersion(v string) []versionToken {
	var tokens []versionToken
	start := 0
	for start < len(v) {
		isDigit := unicode.IsDigit(rune(v[start]))
		end := start
		for end < len(v) && unicode.IsDigit(rune(v[end])) == isDigit {
			end++
		}
		tokens = append(tokens, versionToken{numeric: isDigit, value: v[start:end]})
		start = end
	}
	return tokens
}
