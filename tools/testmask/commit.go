package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// commitTestLineRe matches a commit message line that names acceptance tests to run on
// cloud, e.g. "aws: TestAccept/bundle/resources/jobs". It mirrors the parser in
// acceptance/internal/selection/commit.go, which decides what such a line selects; testmask
// only needs to know that the line is there.
var commitTestLineRe = regexp.MustCompile(`^(aws|azure|gcp|cloud):(.*\s)?TestAccept(/|\s|$)`)

// GetCommitMessage returns the message of the commit at ref.
func GetCommitMessage(ref string) (string, error) {
	out, err := exec.Command("git", "log", "-1", "--format=%B", ref).Output()
	if err != nil {
		return "", fmt.Errorf("failed to read commit message of %s: %w", ref, err)
	}
	return string(out), nil
}

// NamesTests reports whether message has a line naming acceptance tests to run on cloud.
func NamesTests(message string) bool {
	for line := range strings.SplitSeq(message, "\n") {
		if commitTestLineRe.MatchString(strings.TrimSpace(line)) {
			return true
		}
	}
	return false
}
