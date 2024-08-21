package mutator

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncInferRootInternal_ComputeRoot(t *testing.T) {
	s := syncInferRoot{}

	tcases := []struct {
		path string
		root string
		out  string
	}{
		{
			// Test that "." doesn't change the root.
			path: ".",
			root: "/tmp/some/dir",
			out:  "/tmp/some/dir",
		},
		{
			// Test that a subdirectory doesn't change the root.
			path: "sub",
			root: "/tmp/some/dir",
			out:  "/tmp/some/dir",
		},
		{
			// Test that a parent directory changes the root.
			path: "../common",
			root: "/tmp/some/dir",
			out:  "/tmp/some",
		},
		{
			// Test that a deeply nested parent directory changes the root.
			path: "../../../../../../common",
			root: "/tmp/some/dir/that/is/very/deeply/nested",
			out:  "/tmp/some",
		},
		{
			// Test that a parent directory changes the root at the filesystem root boundary.
			path: "../common",
			root: "/tmp",
			out:  "/",
		},
		{
			// Test that an invalid parent directory doesn't change the root and returns an empty string.
			path: "../common",
			root: "/",
			out:  "",
		},
		{
			// Test that the returned path is cleaned even if the root doesn't change.
			path: "sub",
			root: "/tmp/some/../dir",
			out:  "/tmp/dir",
		},
		{
			// Test that a relative root path also works.
			path: "../common",
			root: "foo/bar",
			out:  "foo",
		},
	}

	for _, tc := range tcases {
		out := s.computeRoot(tc.path, tc.root)
		assert.Equal(t, tc.out, filepath.ToSlash(out))
	}
}
