//go:build unix

package project

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAtLeastOnePythonInstalled(t *testing.T) {
	ctx := context.Background()
	all, err := DetectInterpreters(ctx)
	assert.NoError(t, err)
	a := all.Latest()
	t.Logf("latest is: %s", a)
	assert.NotEmpty(t, all)
}

func TestNoInterpretersFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	ctx := context.Background()
	all, err := DetectInterpreters(ctx)
	assert.Nil(t, all)
	assert.Equal(t, ErrNoPythonInterpreters, err)
}

func TestFilteringInterpreters(t *testing.T) {
	rogueBin := filepath.Join(t.TempDir(), "rogue-bin")
	err := os.Mkdir(rogueBin, 0o777)
	assert.NoError(t, err)
	err = os.Chmod(rogueBin, 0o777)
	assert.NoError(t, err)

	raw, err := os.ReadFile("testdata/world-writeable/python8.4")
	assert.NoError(t, err)

	injectedBinary := filepath.Join(rogueBin, "python8.4")
	err = os.WriteFile(injectedBinary, raw, 0o0777)
	assert.NoError(t, err)

	t.Setenv("PATH", "testdata/other-binaries-filtered:"+rogueBin)

	roguePath, err := exec.LookPath("python8.4")
	assert.NoError(t, err)
	assert.Equal(t, injectedBinary, roguePath)

	ctx := context.Background()
	all, err := DetectInterpreters(ctx)
	assert.NoError(t, err)
	assert.Len(t, all, 3)
	assert.Equal(t, "v2.7.18", all[0].Version)
	assert.Equal(t, "v3.10.5", all[1].Version)
	assert.Equal(t, "testdata/other-binaries-filtered/python3.10", all[1].Path)
	assert.Equal(t, "v3.11.4", all[2].Version)
	assert.Equal(t, "testdata/other-binaries-filtered/real-python3.11.4", all[2].Path)
}

func TestInterpretersAtLeastInvalidSemver(t *testing.T) {
	t.Setenv("PATH", "testdata/other-binaries-filtered")

	ctx := context.Background()
	all, err := DetectInterpreters(ctx)
	assert.NoError(t, err)

	_, err = all.AtLeast("v1.2.3.4")
	assert.EqualError(t, err, "invalid SemVer: v1.2.3.4")
}

func TestInterpretersAtLeast(t *testing.T) {
	t.Setenv("PATH", "testdata/other-binaries-filtered")

	ctx := context.Background()
	all, err := DetectInterpreters(ctx)
	assert.NoError(t, err)

	interpreter, err := all.AtLeast("3.10")
	assert.NoError(t, err)
	assert.Equal(t, "testdata/other-binaries-filtered/python3.10", interpreter.Path)
}

func TestInterpretersAtLeastNotSatisfied(t *testing.T) {
	t.Setenv("PATH", "testdata/other-binaries-filtered")

	ctx := context.Background()
	all, err := DetectInterpreters(ctx)
	assert.NoError(t, err)

	_, err = all.AtLeast("4.0.1")
	assert.EqualError(t, err, "cannot find Python greater or equal to v4.0.1")
}
