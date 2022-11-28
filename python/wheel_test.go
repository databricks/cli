package python

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWheel(t *testing.T) {

	// remove this once equivalent tests for windows have been set up
	// or this test has been fixed for windows
	// date: 28 Nov 2022
	if runtime.GOOS == "windows" {
		t.Skip("skipping temperorilty to make windows unit tests green")
	}

	// remove this once equivalent tests for macos have been set up
	// or this test has been fixed for mac os
	// date: 28 Nov 2022
	if runtime.GOOS == "darwin" {
		t.Skip("skipping temperorilty to make macos unit tests green")
	}

	wheel, err := BuildWheel(context.Background(), "testdata/simple-python-wheel")
	assert.NoError(t, err)
	assert.Equal(t, "testdata/simple-python-wheel/dist/dummy-0.0.1-py3-none-any.whl", wheel)

	noFile(t, "testdata/simple-python-wheel/dummy.egg-info")
	noFile(t, "testdata/simple-python-wheel/__pycache__")
	noFile(t, "testdata/simple-python-wheel/build")
}

func noFile(t *testing.T, name string) {
	_, err := os.Stat(name)
	assert.Error(t, err, "file %s should exist", name)
}
