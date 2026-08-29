package main

import "unicode"

// This file holds the name-splitting engine ported from genkit's
// codegen/code/named.go. The casing helpers in names.go all build on splitASCII.

func searchNearest(name string, cond func(rune) bool, forward bool, i int) bool {
	incr := 1
	if !forward {
		incr = -1
	}
	for j := i; j >= 0 && j < len(name); j += incr {
		if unicode.IsLetter(rune(name[j])) {
			return cond(rune(name[j]))
		}
	}
	return false
}

func condAtNearestLetters(name string, cond func(rune) bool, i int) bool {
	r := rune(name[i])
	if unicode.IsLetter(r) {
		return cond(r)
	}
	return searchNearest(name, cond, true, i) && searchNearest(name, cond, false, i)
}

// splitASCII is a faithful port of genkit Named.splitASCII: it splits a name
// into lowercased words at case boundaries and separators, emulating the JVM
// lookahead regex genkit documents.
func splitASCII(name string) (w []string) {
	var current []rune
	nameLen := len(name)
	var isPrevUpper bool
	for i := range nameLen {
		r := rune(name[i])
		if r == '$' {
			continue
		}
		isCurrentUpper := condAtNearestLetters(name, unicode.IsUpper, i)
		r = unicode.ToLower(r)
		isNextLower := false
		isNextUpper := false
		isNotLastChar := i+1 < nameLen
		if isNotLastChar {
			isNextLower = condAtNearestLetters(name, unicode.IsLower, i+1)
			isNextUpper = condAtNearestLetters(name, unicode.IsUpper, i+1)
		}
		split, before, after := false, false, true
		if isPrevUpper && isCurrentUpper && isNextLower && isNotLastChar {
			split = true
			before = false
			after = true
		}
		if !isCurrentUpper && isNextUpper {
			split = true
			before = true
			after = false
		}
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			split = true
			before = false
			after = false
		}
		if before {
			current = append(current, r)
		}
		if split && len(current) > 0 {
			w = append(w, string(current))
			current = []rune{}
		}
		if after {
			current = append(current, r)
		}
		isPrevUpper = isCurrentUpper
	}
	if len(current) > 0 {
		w = append(w, string(current))
	}
	return w
}
