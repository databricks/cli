package localcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/databricks/cli/libs/log"
)

const (
	userRW          = 0o600
	ownerRWXworldRX = 0o755
)

func NewLocalCache[T any](dir, name string, validity time.Duration) LocalCache[T] {
	return LocalCache[T]{
		dir:      dir,
		name:     name,
		validity: validity,
	}
}

type LocalCache[T any] struct {
	name     string
	dir      string
	validity time.Duration
	zero     T
}

func (r *LocalCache[T]) Load(ctx context.Context, refresh func() (T, error)) (T, error) {
	cached, err := r.LoadCache()
	if errors.Is(err, fs.ErrNotExist) {
		return r.refreshCache(ctx, refresh, r.zero)
	} else if err != nil {
		return r.zero, err
	} else if time.Since(cached.Refreshed) > r.validity {
		return r.refreshCache(ctx, refresh, cached.Data)
	}
	return cached.Data, nil
}

type cached[T any] struct {
	// we don't use mtime of the file because it's easier to
	// for testdata used in the unit tests to be somewhere far
	// in the future and don't bother about switching the mtime bit.
	Refreshed time.Time `json:"refreshed_at"`
	Data      T         `json:"data"`
}

func (r *LocalCache[T]) refreshCache(ctx context.Context, refresh func() (T, error), offlineVal T) (T, error) {
	data, err := refresh()
	var urlError *url.Error
	if errors.As(err, &urlError) {
		log.Warnf(ctx, "System offline. Cannot refresh cache: %s", urlError)
		return offlineVal, nil
	}
	if err != nil {
		return r.zero, fmt.Errorf("refresh: %w", err)
	}
	return r.writeCache(ctx, data)
}

func (r *LocalCache[T]) writeCache(ctx context.Context, data T) (T, error) {
	cached := &cached[T]{time.Now(), data}
	raw, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return r.zero, fmt.Errorf("json marshal: %w", err)
	}
	cacheFile := r.FileName()
	err = os.WriteFile(cacheFile, raw, userRW)
	if errors.Is(err, fs.ErrNotExist) {
		cacheDir := filepath.Dir(cacheFile)
		err := os.MkdirAll(cacheDir, ownerRWXworldRX)
		if err != nil {
			return r.zero, fmt.Errorf("create %s: %w", cacheDir, err)
		}
		err = os.WriteFile(cacheFile, raw, userRW)
		if err != nil {
			return r.zero, fmt.Errorf("retry save cache: %w", err)
		}
		return data, nil
	} else if err != nil {
		return r.zero, fmt.Errorf("save cache: %w", err)
	}
	return data, nil
}

func (r *LocalCache[T]) FileName() string {
	return filepath.Join(r.dir, r.name+".json")
}

func (r *LocalCache[T]) LoadCache() (*cached[T], error) {
	jsonFile := r.FileName()
	raw, err := os.ReadFile(r.FileName())
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", jsonFile, err)
	}
	var v cached[T]
	err = json.Unmarshal(raw, &v)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", jsonFile, err)
	}
	return &v, nil
}
