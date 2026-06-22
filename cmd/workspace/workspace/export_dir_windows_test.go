//go:build windows

package workspace

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The narrow contract: only the Windows "invalid file name" error is treated as
// skippable. Genuine failures (permission, missing path, anything else) must not
// be swallowed, otherwise export-dir would silently drop files on real errors.
func TestIsInvalidLocalNameError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "invalid name wrapped in PathError",
			err:  &os.PathError{Op: "open", Path: `C:\tmp\New Notebook 13:54:24.py`, Err: syscall.Errno(0x7b)},
			want: true,
		},
		{
			name: "permission denied is not skipped",
			err:  fs.ErrPermission,
			want: false,
		},
		{
			name: "not exist is not skipped",
			err:  fs.ErrNotExist,
			want: false,
		},
		{
			name: "generic error is not skipped",
			err:  errors.New("boom"),
			want: false,
		},
		{
			name: "nil is not skipped",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isInvalidLocalNameError(tt.err))
		})
	}
}

func TestSanitizeLocalName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "colons replaced",
			in:   "New Notebook 2026-05-04 13:54:24.py",
			want: "New Notebook 2026-05-04 13_54_24.py",
		},
		{
			name: "all reserved characters replaced",
			in:   `a<b>c:d"e|f?g*h`,
			want: "a_b_c_d_e_f_g_h",
		},
		{
			name: "control character replaced",
			in:   "a\tb",
			want: "a_b",
		},
		{
			name: "legal name unchanged",
			in:   "hello world.py",
			want: "hello world.py",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeLocalName(tt.in))
		})
	}
}
