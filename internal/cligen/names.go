package main

import (
	"strings"
	"unicode"
)

// This file ports genkit's pure name-casing functions (codegen/code/named.go)
// so the CLI can derive kebab/snake/pascal/camel/constant/title casings from a
// single stored `name`, instead of the producer denormalizing every variant
// into cli.json. The port must match genkit byte-for-byte; names_test.go pins
// samples that were validated against genkit's output before the stored
// casings were dropped from cli.json.

// title mirrors strings.Title for the ASCII words splitASCII emits: it
// uppercases each letter that begins a word. Like strings.Title's isSeparator,
// digits are not separators, so a letter following a digit stays lowercase
// ("ai21labs" -> "Ai21labs", matching the SDK's serving.Ai21labsApiKey).
// Implemented directly to avoid the deprecated strings.Title and stay
// lint-clean.
func title(s string) string {
	out := []rune(s)
	prevIsSeparator := true
	for i, r := range out {
		if prevIsSeparator && unicode.IsLetter(r) {
			out[i] = unicode.ToUpper(r)
		}
		prevIsSeparator = !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}
	return string(out)
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
