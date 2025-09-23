package textutil

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Transformer represents a text transformation operation.
type Transformer interface {
	TransformString(string) string
}

type chainTransformer []Transformer

func (c chainTransformer) TransformString(s string) string {
	for _, t := range c {
		s = t.TransformString(s)
	}
	return s
}

// Chain creates a transformer that applies multiple transformers in sequence.
func Chain(t ...Transformer) Transformer {
	return chainTransformer(t)
}

// Implement [Transformer] interface with text/transform package.
type textTransformer struct {
	transform.Transformer
}

func (t textTransformer) TransformString(s string) string {
	s, _, _ = transform.String(t, s)
	return s
}

// NormalizeMarks creates a transformer that removes diacritical marks from characters.
// This turns 'é' into 'e' and 'ü' into 'u'.
func NormalizeMarks() Transformer {
	// Decompose unicode characters, then remove all non-spacing marks, then recompose
	return textTransformer{
		transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC),
	}
}

// Replaces characters in the given set with replacement.
type replaceTransformer struct {
	set         runes.Set
	replacement rune
}

func (t replaceTransformer) TransformString(s string) string {
	return strings.Map(func(r rune) rune {
		if t.set.Contains(r) {
			return t.replacement
		}
		return r
	}, s)
}

// ReplaceIn creates a transformer that replaces characters within the given Unicode range table with the replacement rune.
func ReplaceIn(table *unicode.RangeTable, replacement rune) Transformer {
	return replaceTransformer{runes.In(table), replacement}
}

// ReplaceNotIn creates a transformer that replaces characters NOT within the given Unicode range table with the replacement rune.
func ReplaceNotIn(table *unicode.RangeTable, replacement rune) Transformer {
	return replaceTransformer{runes.NotIn(table), replacement}
}

// Trims the given string of all characters in the given set.
type trimTransformer struct {
	set runes.Set
}

func (t trimTransformer) TransformString(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		return t.set.Contains(r)
	})
}

// TrimIfIn creates a transformer that trims characters from the beginning and end of strings if they are within the given Unicode range table.
func TrimIfIn(table *unicode.RangeTable) Transformer {
	return trimTransformer{runes.In(table)}
}

// TrimIfNotIn creates a transformer that trims characters from the beginning and end of strings if they are NOT within the given Unicode range table.
func TrimIfNotIn(table *unicode.RangeTable) Transformer {
	return trimTransformer{runes.NotIn(table)}
}
