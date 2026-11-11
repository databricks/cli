package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

// commitTestLineRe matches the prefix of a commit message line that names acceptance tests to
// run on cloud, e.g. "aws: TestAccept/bundle/resources/jobs". It mirrors the parser in
// acceptance/internal/selection/commit.go, which decides what such a line selects (the tools
// module cannot import it); testmask only needs to know that the line is there.
var commitTestLineRe = regexp.MustCompile(`^(aws|azure|gcp|cloud):(.*)$`)

// GetCommitMessage returns the message of the commit at ref.
func GetCommitMessage(ref string) (string, error) {
	out, err := exec.Command("git", "log", "-1", "--format=%B", ref).Output()
	if err != nil {
		return "", fmt.Errorf("failed to read commit message of %s: %w", ref, err)
	}
	return string(out), nil
}

// NamesTests reports whether message has a line naming acceptance tests to run on cloud: a
// cloud prefix followed only by go test names, at least one of them an acceptance test.
func NamesTests(message string) bool {
	for line := range strings.SplitSeq(message, "\n") {
		m := commitTestLineRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		words := strings.Fields(m[2])
		if slices.ContainsFunc(words, func(word string) bool { return !strings.HasPrefix(word, "Test") }) {
			continue
		}
		if slices.ContainsFunc(words, func(word string) bool { return word == "TestAccept" || strings.HasPrefix(word, "TestAccept/") }) {
			return true
		}
	}
	return false
}
