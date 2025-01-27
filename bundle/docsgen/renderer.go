package main

import (
	"fmt"
	"runtime"
	"strings"
)

type markdownRenderer struct {
	nodes []string
}

func newMardownRenderer() *markdownRenderer {
	return &markdownRenderer{}
}

func (m *markdownRenderer) add(s string) *markdownRenderer {
	m.nodes = append(m.nodes, s)
	return m
}

func (m *markdownRenderer) PlainText(s string) *markdownRenderer {
	return m.add(s)
}

func (m *markdownRenderer) LF() *markdownRenderer {
	return m.add("  ")
}

func (m *markdownRenderer) H2(s string) *markdownRenderer {
	return m.add("## " + s)
}

func (m *markdownRenderer) H3(s string) *markdownRenderer {
	return m.add("### " + s)
}

func (m *markdownRenderer) CodeBlocks(lang, s string) *markdownRenderer {
	return m.add(fmt.Sprintf("```%s%s%s%s```", lang, lineFeed(), s, lineFeed()))
}

func (m *markdownRenderer) String() string {
	return strings.Join(m.nodes, lineFeed())
}

func lineFeed() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}
