package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLongestCommonDirectoryPrefix(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  string
	}{
		{
			name:  "empty_list",
			files: []string{},
			want:  "",
		},
		{
			name:  "single_file",
			files: []string{"foo/bar/baz.txt"},
			want:  "foo/bar/",
		},
		{
			name:  "all_same_dir",
			files: []string{"foo/bar/a.go", "foo/bar/b.go"},
			want:  "foo/bar/",
		},
		{
			name:  "different_dirs",
			files: []string{"foo/bar/a.go", "foo/baz/a.go"},
			want:  "foo/",
		},
		{
			name:  "top_level",
			files: []string{"top/a.go", "top/b.go"},
			want:  "top/",
		},
		{
			name:  "absolute_paths_same_dir",
			files: []string{"/a/b/c.go", "/a/b/d.go"},
			want:  "/a/b/",
		},
		{
			name:  "absolute_paths_diff_dirs",
			files: []string{"/a/b/c.go", "/a/c/b.go"},
			want:  "/a/",
		},
		{
			name:  "no_common_prefix",
			files: []string{"foo/a.go", "bar/b.go"},
			want:  "",
		},
		{
			name:  "partial_overlap",
			files: []string{"foo/bar/baz/a.go", "foo/bar/qux/b.go"},
			want:  "foo/bar/",
		},
		{
			name:  "single_character_common",
			files: []string{"a/b.go", "a/c.go"},
			want:  "a/",
		},
		{
			name:  "root_level_file",
			files: []string{"file.go"},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := longestCommonDirectoryPrefix(tt.files)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHandleExperimental(t *testing.T) {
	// Changes only in experimental/ssh/ should match.
	packages, acceptance, err := handleExperimental([]string{
		"experimental/ssh/main.go",
		"experimental/ssh/lib/server.go",
	})
	assert.NoError(t, err)
	assert.Equal(t, []string{"experimental/ssh/..."}, packages)
	assert.Empty(t, acceptance)

	// Changes in experimental/ plus go.mod should not match.
	_, _, err = handleExperimental([]string{
		"experimental/ssh/main.go",
		"go.mod",
	})
	assert.ErrorIs(t, err, errSkip)

	// Changes spanning multiple experimental subdirs should test all of experimental.
	packages, acceptance, err = handleExperimental([]string{
		"experimental/ssh/main.go",
		"experimental/aitools/server.go",
	})
	assert.NoError(t, err)
	assert.Equal(t, []string{"experimental/..."}, packages)
	assert.Empty(t, acceptance)
}
