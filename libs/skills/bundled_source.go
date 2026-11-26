package skills

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
)

// bundledSnapshot is a skills release compiled into the binary. It is the
// fallback for when the published release cannot be reached and nothing is
// cached. The snapshot is filled when release binaries are built, so the
// repo holds only an empty manifest.
//
//go:embed snapshot
var bundledSnapshot embed.FS

// BundledLabel says where a catalog from the bundled snapshot came from.
const BundledLabel = "bundled copy"

// bundledFS is the root of the bundled snapshot. Tests swap it for a fixture release.
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
//
//deadcode:allow the wiring task calls it; nothing reads the snapshot yet
func NewBundledSource() Source {
	return &bundledSource{src: NewFSSource(bundledFS, BundledLabel)}
}

type bundledSource struct {
	src Source
}

func (s *bundledSource) Catalog(ctx context.Context) (*Catalog, error) {
	catalog, err := s.src.Catalog(ctx)
	if err != nil {
		return nil, err
	}
	if len(catalog.Skills) == 0 {
		return nil, fmt.Errorf("%w: the bundled copy has no skills", ErrSourceUnreachable)
	}
	return catalog, nil
}

func (s *bundledSource) ReadFile(ctx context.Context, skill, file string) ([]byte, error) {
	if _, err := s.Catalog(ctx); err != nil {
		return nil, err
	}
	return s.src.ReadFile(ctx, skill, file)
}
