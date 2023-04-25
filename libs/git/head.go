package git

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type HeadType string

const (
	// a reference of the format `ref: refs/heads/my-branch-name`
	HeadTypeReference = HeadType("reference")
	// A hexadecimal encoded SHA1 hash
	HeadTypeSHA1 = HeadType("sha-1")
)

// relevant documentation about git references:
// https://git-scm.com/book/en/v2/Git-Internals-Git-References
type Head struct {
	Type    HeadType
	Content string
}

const ReferencePrefix = "ref: "
const ReferencePathPrefix = "refs/heads/"

// asserts if a string is a 40 character hexadecimal encoded string
func isSHA1(s string) bool {
	if len(s) != 40 {
		return false
	}
	re := regexp.MustCompile("^[0-9a-f]+$")
	return re.MatchString(s)
}

func LoadHead(path string) (*Head, error) {
	// read head file content
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// trim new line characters
	content := strings.TrimSuffix(string(b), "\n")
	content = strings.TrimSuffix(content, "\r")

	// determine HEAD type
	var headType HeadType
	switch {
	case strings.HasPrefix(content, ReferencePrefix):
		headType = HeadTypeReference
	case isSHA1(content):
		headType = HeadTypeSHA1
	default:
		return nil, fmt.Errorf("unknown format for git HEAD: %s", content)
	}

	return &Head{
		Type:    headType,
		Content: content,
	}, nil
}

func (head *Head) ReferencePath() (string, error) {
	if head.Type != HeadTypeReference {
		return "", fmt.Errorf("HEAD is not a git reference")
	}
	refPath := strings.TrimPrefix(head.Content, ReferencePrefix)
	return filepath.FromSlash(refPath), nil
}

func (head *Head) CurrentBranch() (string, error) {
	refPath, err := head.ReferencePath()
	if err != nil {
		return "", err
	}
	normalizeRefPath := filepath.ToSlash(refPath)
	if !strings.HasPrefix(normalizeRefPath, ReferencePathPrefix) {
		return "", fmt.Errorf("reference path %s does not have expected prefix %s", normalizeRefPath, ReferencePathPrefix)
	}
	return strings.TrimPrefix(normalizeRefPath, ReferencePathPrefix), nil
}
