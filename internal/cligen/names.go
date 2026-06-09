package main

import (
	"strings"
	"unicode"
)

// This file ports genkit's pure name-casing functions (codegen/code/named.go)
// so the CLI can derive kebab/snake/pascal/camel/constant/title casings from a
// single stored `name`, instead of the producer denormalizing every variant
// into cli.json. The port must match genkit byte-for-byte; names_test.go
// asserts that against a cli.json that still carries the stored casings.

// title mirrors strings.Title for the ASCII words splitASCII emits: it
// uppercases each letter that follows a non-letter (or the start). Implemented
// directly to avoid the deprecated strings.Title and stay lint-clean.
func title(s string) string {
	out := []rune(s)
	prevIsLetter := false
	for i, r := range out {
		if !prevIsLetter && unicode.IsLetter(r) {
			out[i] = unicode.ToUpper(r)
		}
		prevIsLetter = unicode.IsLetter(r)
	}
	return string(out)
}

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

func pascalName(name string) string {
	var sb strings.Builder
	for _, w := range splitASCII(name) {
		sb.WriteString(title(w))
	}
	return sb.String()
}

func titleName(name string) string {
	words := splitASCII(name)
	for i, w := range words {
		words[i] = title(w)
	}
	return strings.Join(words, " ")
}

func camelName(name string) string {
	if name == "" {
		return ""
	}
	if name == "_" {
		return "_"
	}
	cc := pascalName(name)
	return strings.ToLower(cc[0:1]) + cc[1:]
}

func snakeName(name string) string {
	if name == "_" {
		return "_"
	}
	return strings.Join(splitASCII(name), "_")
}

func constantName(name string) string {
	return strings.ToUpper(snakeName(name))
}

func kebabName(name string) string {
	return strings.Join(splitASCII(name), "-")
}

// trimPrefixName mirrors Named.TrimPrefix: trims the prefix from the camel-cased
// name and returns the resulting bare name (whose casings are derived as usual).
func trimPrefixName(name, prefix string) string {
	return strings.TrimPrefix(camelName(name), prefix)
}
