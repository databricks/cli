package sync

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/filer"
	"golang.org/x/sync/errgroup"
)

// Maximum number of concurrent requests during sync.
const MaxRequestsInFlight = 20

// Perform a DELETE of the specified remote path.
func (s *Sync) applyDelete(ctx context.Context, group *errgroup.Group, remoteName string) {
	// Return early if the context has already been cancelled.
	select {
	case <-ctx.Done():
		return
	default:
		// Proceed.
	}

	group.Go(func() error {
		s.notifyProgress(ctx, EventActionDelete, remoteName, 0.0)
		err := s.filer.Delete(ctx, remoteName)
		if err != nil {
			return err
		}
		s.notifyProgress(ctx, EventActionDelete, remoteName, 1.0)
		return nil
	})
}

// Perform a PUT of the specified local path.
func (s *Sync) applyPut(ctx context.Context, group *errgroup.Group, localName string) {
	// Return early if the context has already been cancelled.
	select {
	case <-ctx.Done():
		return
	default:
		// Proceed.
	}

	group.Go(func() error {
		s.notifyProgress(ctx, EventActionPut, localName, 0.0)

		contents, err := os.ReadFile(filepath.Join(s.LocalPath, localName))
		if err != nil {
			return err
		}

		opts := []filer.WriteMode{filer.CreateParentDirectories, filer.OverwriteIfExists}
		err = s.filer.Write(ctx, localName, bytes.NewReader(contents), opts...)
		if err != nil {
			return err
		}

		s.notifyProgress(ctx, EventActionPut, localName, 1.0)
		return nil
	})
}

func (s *Sync) applyDiff(ctx context.Context, d diff) error {
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(MaxRequestsInFlight)

	for _, remoteName := range d.delete {
		s.applyDelete(ctx, group, remoteName)
	}

	for _, localName := range d.put {
		s.applyPut(ctx, group, localName)
	}

	// Wait for goroutines to finish and return first non-nil error return if any.
	return group.Wait()
}
