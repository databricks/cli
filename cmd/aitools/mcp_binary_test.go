package aitools

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBinaryPathFor(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	absolute := filepath.Join(t.TempDir(), "bin", "databricks")
	foundPath := filepath.Join(t.TempDir(), "databricks")
	relativeOnPath := filepath.Join("bin", "databricks")

	tests := []struct {
		name string
		arg0 string
		// onPath is what the PATH lookup finds, or "" when the name is not on PATH.
		onPath string
		// wantLookup is the name the PATH lookup receives, or "" when the lookup must not run.
		wantLookup string
		wantPath   string
		wantAbs    bool
	}{
		{
			// The invoked path is kept as is, so a Homebrew symlink is not replaced by its target.
			name:     "absolute arg0 is kept",
			arg0:     absolute,
			wantPath: absolute,
			wantAbs:  true,
		},
		{
			name:     "relative arg0 with a directory is made absolute, not looked up",
			arg0:     filepath.Join("bin", "databricks"),
			onPath:   foundPath,
			wantPath: mustAbs(t, filepath.Join("bin", "databricks")),
			wantAbs:  true,
		},
		{
			name:     "dot-relative arg0 with a directory is made absolute",
			arg0:     "." + string(filepath.Separator) + "databricks",
			onPath:   foundPath,
			wantPath: wd + string(filepath.Separator) + "databricks",
			wantAbs:  true,
		},
		{
			// ".//bin" is still relative: the separators after "." do not make it a root.
			name:     "dot-relative arg0 with repeated separators stays relative",
			arg0:     "." + string(filepath.Separator) + string(filepath.Separator) + "bin" + string(filepath.Separator) + "databricks",
			onPath:   foundPath,
			wantPath: wd + string(filepath.Separator) + "bin" + string(filepath.Separator) + "databricks",
			wantAbs:  true,
		},
		{
			// A path with no drive is on the working directory's drive. On Windows it is not absolute, and the drive is the
			// volume of the working directory. On other systems it is absolute and kept as is.
			name:     "rooted arg0 with no drive is on the working directory's drive",
			arg0:     string(filepath.Separator) + filepath.Join("tools", "databricks.exe"),
			onPath:   foundPath,
			wantPath: filepath.VolumeName(wd) + string(filepath.Separator) + filepath.Join("tools", "databricks.exe"),
			wantAbs:  true,
		},
		{
			// The OS reads "link/.." through the link's target, so the path must not be cleaned before the symlink is followed.
			name:     "relative arg0 through a symlink keeps its dot-dot",
			arg0:     "link" + string(filepath.Separator) + ".." + string(filepath.Separator) + "databricks",
			onPath:   foundPath,
			wantPath: wd + string(filepath.Separator) + "link" + string(filepath.Separator) + ".." + string(filepath.Separator) + "databricks",
			wantAbs:  true,
		},
		{
			name:       "bare name is looked up on PATH",
			arg0:       "databricks",
			onPath:     foundPath,
			wantLookup: "databricks",
			wantPath:   foundPath,
			wantAbs:    true,
		},
		{
			name:       "bare name not on PATH falls back to the bare name",
			arg0:       "databricks",
			onPath:     "",
			wantLookup: "databricks",
			wantPath:   "databricks",
		},
		{
			// Only a databricks argv[0] may fall back to the bare name; another name is kept, so the caller refuses it.
			name:       "other bare name not on PATH is kept",
			arg0:       "dbx",
			onPath:     "",
			wantLookup: "dbx",
			wantPath:   "dbx",
		},
		{
			name:       "relative PATH entry falls back to the bare name",
			arg0:       "databricks",
			onPath:     relativeOnPath,
			wantLookup: "databricks",
			wantPath:   "databricks",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := &pathLookup{path: tt.onPath}
			path, abs := binaryPathFor(tt.arg0, lookup.lookPath, noExecutable)
			assert.Equal(t, tt.wantPath, path)
			assert.Equal(t, tt.wantAbs, abs)
			assert.Equal(t, tt.wantLookup, lookup.asked)
		})
	}
}

// TestBinaryPathForUsesRunningExecutableOnDot checks the case where the PATH lookup finds the binary only in the working
// directory. exec.LookPath then returns a relative path with exec.ErrDot, as it does for a databricks.exe run from the
// folder it was unzipped to on Windows. The running executable is written instead, since it is absolute. The lookup is
// stubbed with the error that LookPath returns, so the case runs on every OS.
func TestBinaryPathForUsesRunningExecutableOnDot(t *testing.T) {
	running := filepath.Join(t.TempDir(), "databricks.exe")
	dot := "." + string(filepath.Separator) + "databricks"
	dotErr := &exec.Error{Name: "databricks", Err: exec.ErrDot}
	notFoundErr := &exec.Error{Name: "databricks", Err: exec.ErrNotFound}

	tests := []struct {
		name     string
		lookPath string
		lookErr  error
		exe      string
		exeErr   error
		wantPath string
		wantAbs  bool
		wantExe  bool
	}{
		{
			name:     "dot lookup writes the running executable",
			lookPath: dot,
			lookErr:  dotErr,
			exe:      running,
			wantPath: running,
			wantAbs:  true,
			wantExe:  true,
		},
		{
			name:     "dot lookup keeps the bare name when the running executable is unknown",
			lookPath: dot,
			lookErr:  dotErr,
			exeErr:   errors.New("no executable"),
			wantPath: "databricks",
			wantExe:  true,
		},
		{
			name:     "not found keeps the bare name without asking for the running executable",
			lookErr:  notFoundErr,
			exe:      running,
			wantPath: "databricks",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookPath := func(string) (string, error) { return tt.lookPath, tt.lookErr }
			asked := false
			executable := func() (string, error) {
				asked = true
				return tt.exe, tt.exeErr
			}
			path, abs := binaryPathFor("databricks", lookPath, executable)
			assert.Equal(t, tt.wantPath, path)
			assert.Equal(t, tt.wantAbs, abs)
			assert.Equal(t, tt.wantExe, asked)
		})
	}
}

// noExecutable stands in for os.Executable in cases that must not reach it.
func noExecutable() (string, error) {
	return "", errors.New("os.Executable must not be called")
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	require.NoError(t, err)
	return abs
}

// pathLookup stands in for the PATH lookup. It finds the name at path, or reports it not found when path is "". It
// records the name it was asked for, so a case can check whether the lookup ran at all.
type pathLookup struct {
	path  string
	asked string
}

func (l *pathLookup) lookPath(name string) (string, error) {
	l.asked = name
	if l.path == "" {
		return "", errors.New("not found")
	}
	return l.path, nil
}

// TestMcpBinaryPathReadsArgsWhenRun checks that the invoked path is read when the command runs, not when the package
// loads. An empty argv has no invoked path and is refused, not a panic.
func TestMcpBinaryPathReadsArgsWhenRun(t *testing.T) {
	saved := os.Args
	t.Cleanup(func() { os.Args = saved })

	absolute := filepath.Join(t.TempDir(), "databricks")
	os.Args = []string{absolute}
	path, abs := mcpBinaryPath()
	assert.Equal(t, absolute, path)
	assert.True(t, abs)

	os.Args = nil
	path, abs = mcpBinaryPath()
	assert.Empty(t, path)
	assert.False(t, abs)
}
