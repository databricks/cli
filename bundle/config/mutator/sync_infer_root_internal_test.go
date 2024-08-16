package mutator

import (
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
			path: ".",
			root: "/tmp/some/dir",
			out:  "/tmp/some/dir",
		},
		{
			path: "sub",
			root: "/tmp/some/dir",
			out:  "/tmp/some/dir",
		},
		{
			path: "../common",
			root: "/tmp/some/dir",
			out:  "/tmp/some",
		},
		{
			path: "../../../../../../common",
			root: "/tmp/some/dir/that/is/very/deeply/nested",
			out:  "/tmp/some",
		},
		{
			path: "../common",
			root: "/tmp",
			out:  "/",
		},
		{
			path: "../common",
			root: "/",
			out:  "",
		},
	}

	for _, tc := range tcases {
		out := s.computeRoot(tc.path, tc.root)
		assert.Equal(t, tc.out, out)
	}
}
