package git

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/databricks/cli/libs/vfs"
)

type ReferenceType string

var (
	ErrNotAReferencePointer = errors.New("HEAD does not point to another reference")
	ErrNotABranch           = errors.New("HEAD is not a reference to a git branch")
)

const (
	// pointer to a secondary reference file path containing sha-1 object ID.
	// eg: `ref: refs/heads/my-branch-name`
	ReferenceTypePointer = ReferenceType("pointer")
	// A hexadecimal encoded SHA1 hash
	ReferenceTypeSHA1 = ReferenceType("sha-1")
)

// relevant documentation about git references:
// https://git-scm.com/book/en/v2/Git-Internals-Git-References
type Reference struct {
	Type    ReferenceType
	Content string
}

const (
	ReferencePrefix = "ref: "
	HeadPathPrefix  = "refs/heads/"
)

// asserts if a string is a 40 character hexadecimal encoded string
func isSHA1(s string) bool {
	re := regexp.MustCompile("^[0-9a-f]{40}$")
	return re.MatchString(s)
}

func LoadReferenceFile(root vfs.Path, path string) (*Reference, error) {
	// read reference file content
	b, err := fs.ReadFile(root, path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// trim new line characters
	content := strings.TrimRight(string(b), "\r\n")

	// determine HEAD type
	var refType ReferenceType
	switch {
	case strings.HasPrefix(content, ReferencePrefix):
		refType = ReferenceTypePointer
	case isSHA1(content):
		refType = ReferenceTypeSHA1
	default:
		return nil, fmt.Errorf("unknown format for git HEAD: %s", content)
	}

	return &Reference{
		Type:    refType,
		Content: content,
	}, nil
}

// resolves the path to the secondary reference file pointd to. eg: if the file
// contents are `ref: a/b/c`, then this function returns `a/b/c`
func (ref *Reference) ResolvePath() (string, error) {
	if ref.Type != ReferenceTypePointer {
		return "", ErrNotAReferencePointer
	}
	return strings.TrimPrefix(ref.Content, ReferencePrefix), nil
}

// resolves the name of the current branch from the reference file content. For example
// `ref: refs/heads/my-branch` returns `my-branch`
func (ref *Reference) CurrentBranch() (string, error) {
	branchRefPath, err := ref.ResolvePath()
	if err == ErrNotAReferencePointer {
		return "", ErrNotABranch
	}
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(branchRefPath, HeadPathPrefix) {
		return "", fmt.Errorf("reference path %s does not have expected prefix %s", branchRefPath, HeadPathPrefix)
	}
	return strings.TrimPrefix(branchRefPath, HeadPathPrefix), nil
}
