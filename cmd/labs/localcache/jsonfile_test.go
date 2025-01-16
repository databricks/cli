package localcache

import (
	"context"
	"errors"
	"net/url"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreatesDirectoryIfNeeded(t *testing.T) {
	ctx := context.Background()
	c := NewLocalCache[int64](t.TempDir(), "some/nested/file", 1*time.Minute)
	thing := []int64{0}
	tick := func() (int64, error) {
		thing[0] += 1
		return thing[0], nil
	}
	first, err := c.Load(ctx, tick)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), first)
}

func TestImpossibleToCreateDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("No /dev/null on windows")
	}
	ctx := context.Background()
	c := NewLocalCache[int64]("/dev/null", "some/nested/file", 1*time.Minute)
	thing := []int64{0}
	tick := func() (int64, error) {
		thing[0] += 1
		return thing[0], nil
	}
	_, err := c.Load(ctx, tick)
	assert.Error(t, err)
}

func TestRefreshes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("No /dev/null on windows")
	}
	ctx := context.Background()
	c := NewLocalCache[int64](t.TempDir(), "time", 1*time.Minute)
	thing := []int64{0}
	tick := func() (int64, error) {
		thing[0] += 1
		return thing[0], nil
	}
	first, err := c.Load(ctx, tick)
	assert.NoError(t, err)

	second, err := c.Load(ctx, tick)
	assert.NoError(t, err)
	assert.Equal(t, first, second)

	c.validity = 0
	third, err := c.Load(ctx, tick)
	assert.NoError(t, err)
	assert.NotEqual(t, first, third)
}

func TestSupportOfflineSystem(t *testing.T) {
	c := NewLocalCache[int64](t.TempDir(), "time", 1*time.Minute)
	thing := []int64{0}
	tick := func() (int64, error) {
		thing[0] += 1
		return thing[0], nil
	}
	ctx := context.Background()
	first, err := c.Load(ctx, tick)
	assert.NoError(t, err)

	tick = func() (int64, error) {
		return 0, &url.Error{
			Op:  "X",
			URL: "Y",
			Err: errors.New("nope"),
		}
	}

	c.validity = 0

	// offline during refresh
	third, err := c.Load(ctx, tick)
	assert.NoError(t, err)
	assert.Equal(t, first, third)

	// fully offline
	c = NewLocalCache[int64](t.TempDir(), "time", 1*time.Minute)
	zero, err := c.Load(ctx, tick)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), zero)
}

func TestFolderDisappears(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("No /dev/null on windows")
	}
	c := NewLocalCache[int64]("/dev/null", "time", 1*time.Minute)
	tick := func() (int64, error) {
		now := time.Now().UnixNano()
		t.Log("TICKS")
		return now, nil
	}
	ctx := context.Background()
	_, err := c.Load(ctx, tick)
	assert.Error(t, err)
}

func TestRefreshFails(t *testing.T) {
	c := NewLocalCache[int64](t.TempDir(), "time", 1*time.Minute)
	tick := func() (int64, error) {
		return 0, errors.New("nope")
	}
	ctx := context.Background()
	_, err := c.Load(ctx, tick)
	assert.EqualError(t, err, "refresh: nope")
}

func TestWrongType(t *testing.T) {
	c := NewLocalCache[chan int](t.TempDir(), "x", 1*time.Minute)
	ctx := context.Background()
	_, err := c.Load(ctx, func() (chan int, error) {
		return make(chan int), nil
	})
	assert.EqualError(t, err, "json marshal: json: unsupported type: chan int")
}
