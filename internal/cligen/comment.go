package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// This file is a faithful copy of the comment/summary helpers from genkit's
// codegen/code/named.go. The CLI owns this rendering logic so that command
// stubs can be regenerated from cli.json without depending on genkit. The
// cli.json spec carries raw Name/Description/Summary; the wrapping below must
// match genkit byte-for-byte.

var whitespace = regexp.MustCompile(`\s+`)

// markdownLink matches markdown links, ignoring new lines.
var markdownLink = regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`)

// sentences splits a description into sentences on ". ", normalizing whitespace.
func sentences(description string) []string {
	if description == "" {
		return []string{}
	}
	norm := whitespace.ReplaceAllString(description, " ")
	trimmed := strings.TrimSpace(norm)
	return strings.Split(trimmed, ". ")
}

// summarize returns the first sentence from the description, always ending in a
// dot. Mirrors Named.Summary in genkit.
func summarize(description string) string {
	s := sentences(description)
	if len(s) > 0 {
		return strings.TrimSuffix(s[0], ".") + "."
	}
	return ""
}

// commentWrap formats a description into a language-specific multi-line comment.
// Mirrors Named.Comment(prefix, maxLen) in genkit.
func commentWrap(description, prefix string, maxLen int) string {
	if description == "" {
		return ""
	}
	trimmed := strings.TrimSpace(description)
	// Collect links, which are later sorted and appended at the bottom.
	links := map[string]string{}
	for _, m := range markdownLink.FindAllStringSubmatch(trimmed, -1) {
		label := strings.TrimSpace(m[1])
		link := strings.TrimSpace(m[2])
		if !strings.HasPrefix(link, "http") {
			// This condition is here until the spec normalizes all links.
			continue
		}
		// Overriding links handles duplicates and yields alphabetical ordering below.
		links[label] = link
		// Replace [text](url) with [text].
		trimmed = strings.ReplaceAll(trimmed, m[0], fmt.Sprintf("[%s]", label))
	}
	var linksInBottom []string
	for k, v := range links {
		linksInBottom = append(linksInBottom, fmt.Sprintf("[%s]: %s", k, v))
	}
	slices.Sort(linksInBottom)
	// Fix new-line characters.
	trimmed = strings.ReplaceAll(trimmed, "\\n", "\n")
	description = strings.ReplaceAll(trimmed, "\n\n", " __BLANK__ ")
	var lines []string
	currentLine := strings.Builder{}
	for _, v := range whitespace.Split(description, -1) {
		if v == "__BLANK__" {
			lines = append(lines, currentLine.String())
			lines = append(lines, "")
			currentLine.Reset()
			continue
		}
		if len(prefix)+currentLine.Len()+len(v)+1 > maxLen {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
		}
		if currentLine.Len() > 0 {
			currentLine.WriteRune(' ')
		}
		currentLine.WriteString(v)
	}
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
		currentLine.Reset()
	}
	if len(linksInBottom) > 0 {
		lines = append(append(lines, ""), linksInBottom...)
	}
	return strings.TrimLeft(prefix, "\t ") + strings.Join(lines, "\n"+prefix)
}
