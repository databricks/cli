package tags

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type transformer interface {
	transform(string) string
}

type chainTransformer []transformer

func (c chainTransformer) transform(s string) string {
	for _, t := range c {
		s = t.transform(s)
	}
	return s
}

func chain(t ...transformer) transformer {
	return chainTransformer(t)
}

// Implement [transformer] interface with text/transform package.
type textTransformer struct {
	transform.Transformer
}

func (t textTransformer) transform(s string) string {
	s, _, _ = transform.String(t, s)
	return s
}

func normalizeMarks() transformer {
	// Decompose unicode characters, then remove all non-spacing marks, then recompose.
	// This turns 'é' into 'e' and 'ü' into 'u'.
	return textTransformer{
		transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC),
	}
}

// Replaces characters in the given set with replacement.
type replaceTransformer struct {
	set         runes.Set
	replacement rune
}

func (t replaceTransformer) transform(s string) string {
	return strings.Map(func(r rune) rune {
		if t.set.Contains(r) {
			return t.replacement
		}
		return r
	}, s)
}

func replaceIn(table *unicode.RangeTable, replacement rune) transformer {
	return replaceTransformer{runes.In(table), replacement}
}

func replaceNotIn(table *unicode.RangeTable, replacement rune) transformer {
	return replaceTransformer{runes.NotIn(table), replacement}
}

// Trims the given string of all characters in the given set.
type trimTransformer struct {
	set runes.Set
}

func (t trimTransformer) transform(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		return t.set.Contains(r)
	})
}

func trimIfIn(table *unicode.RangeTable) transformer {
	return trimTransformer{runes.In(table)}
}

func trimIfNotIn(table *unicode.RangeTable) transformer {
	return trimTransformer{runes.NotIn(table)}
}
