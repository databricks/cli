package skills

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"time"
)

// bundledSnapshot is a skills release compiled into the binary. It is the
// fallback for when the published release cannot be reached and nothing is
// cached. The repo holds only an empty manifest; a later change fills the
// snapshot when release binaries are built.
//
// The all: prefix embeds files whose names start with "." or "_" too. Without
// it the compiler drops them silently, and a skill file such as
// scripts/__init__.py would be listed in the catalog but missing from the copy.
//
//go:embed all:snapshot
var bundledSnapshot embed.FS

// BundledLabel says where a catalog from the bundled snapshot came from.
const BundledLabel = "bundled copy"

// bundledFS is the root of the compiled-in snapshot.
var bundledFS = subSnapshot()

func subSnapshot() fs.FS {
	sub, err := fs.Sub(bundledSnapshot, "snapshot")
	if err != nil {
		panic(err)
	}
	return sub
}

// NewBundledSource returns the source over the snapshot compiled into the
// binary. A snapshot with no skills reports ErrSourceUnreachable, so a
// development build behaves as if it had no bundled copy.
func NewBundledSource() Source {
	return newBundledSource(bundledFS)
}

// newBundledSource serves a snapshot rooted at fsys. Tests pass a fixture
// release here, so they do not swap the package variable.
func newBundledSource(fsys fs.FS) Source {
	return &bundledSource{src: NewFSSource(fsys, BundledLabel)}
}

type bundledSource struct {
	src Source
}

// Catalog reports the snapshot as a stale copy with no fetch time. The binary
// does not record when the snapshot was built, and the snapshot is only used
// when the release cannot be reached, so it is never current.
func (s *bundledSource) Catalog(ctx context.Context) (*Catalog, error) {
	catalog, err := s.src.Catalog(ctx)
	if err != nil {
		return nil, err
	}
	if len(catalog.Skills) == 0 {
		return nil, fmt.Errorf("%w: the bundled copy has no skills", ErrSourceUnreachable)
	}
	catalog.FetchedAt = time.Time{}
	catalog.Stale = true
	return catalog, nil
}

func (s *bundledSource) ReadFile(ctx context.Context, skill, file string) ([]byte, error) {
	if _, err := s.Catalog(ctx); err != nil {
		return nil, err
	}
	return s.src.ReadFile(ctx, skill, file)
}
