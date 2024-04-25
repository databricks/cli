package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func findGoFiles(t *testing.T) []string {
	var files []string
	err := filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if filepath.Base(path) == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			return nil
		}

		files = append(files, path)
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, files)
	return files
}

type stringVisitor struct {
	acc []string
}

func (v *stringVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	// Check if the node is a basic literal and a string
	if lit, ok := node.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		v.acc = append(v.acc, lit.Value)
	}

	return v
}

func findURLsInFile(t *testing.T, filename string) []string {
	src, err := os.ReadFile(filename)
	require.NoError(t, err)

	// Parse the source file
	fset := token.NewFileSet() // positions are relative to fset
	node, err := parser.ParseFile(fset, filename, src, 0)
	require.NoError(t, err, "Failed to parse file: %s", filename)

	// Traverse the AST
	v := new(stringVisitor)
	ast.Walk(v, node)

	ignorePatterns := []string{
		"https://<databricks-instance>.cloud.databricks.com",
		"must start with https://",
		"https://%s",
		"http://%s",
		`"https://"`,
		`"http://"`,
	}

	// Extract URLs from strings
	var urls []string
	for _, str := range v.acc {
		hasURL := strings.Contains(str, "http://") || strings.Contains(str, "https://")
		// re := regexp.MustCompile(`^https?://[a-zA-Z0-9\-\.]+\.[a-zA-Z]{2,}\b(?:[-a-zA-Z0-9()@:%_\-\+.~#?&\/=]*)$/`)
		re := regexp.MustCompile(`https?://[a-zA-Z0-9\-\.]+\.[a-zA-Z]{2,}(?:(/[\w\.\-#]*)+([\w\-]))?`)
		matches := re.FindAllString(str, -1)
		if len(matches) == 0 {
			if !hasURL {
				continue
			}
			// Check if any of the ignore substrings match
			ignore := false
			for _, pattern := range ignorePatterns {
				if strings.Contains(str, pattern) {
					ignore = true
					break
				}
			}
			if ignore {
				continue
			}
			require.False(t, hasURL, "Found URL in %s, but regexp didn't match it -- %q", filename, str)
			continue
		}

		urls = append(urls, matches...)
	}

	return urls
}

func TestValidateURLs(t *testing.T) {
	files := findGoFiles(t)
	all := []string{}
	for _, file := range files {
		urls := findURLsInFile(t, file)
		all = append(all, urls...)
	}
}
