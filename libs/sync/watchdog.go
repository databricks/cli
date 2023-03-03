package sync

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// See https://docs.databricks.com/resources/limits.html#limits-api-rate-limits for per api
// rate limits
const MaxRequestsInFlight = 20

func syncCallback(ctx context.Context, s *Sync) func(localDiff diff) error {
	return func(d diff) error {
		// Abstraction over wait groups which allows you to get the errors
		// returned in goroutines
		var g errgroup.Group

		// Allow MaxRequestLimit maxiumum concurrent api calls
		g.SetLimit(MaxRequestsInFlight)

		for _, remoteName := range d.delete {
			// Copy of remoteName created to make this safe for concurrent use.
			// directly using remoteName can cause race conditions since the loop
			// might iterate over to the next remoteName before the go routine function
			// is evaluated
			remoteNameCopy := remoteName
			g.Go(func() error {
				s.notifyProgress(ctx, EventActionDelete, remoteNameCopy, 0.0)
				err := s.repoFiles.DeleteFile(ctx, remoteNameCopy)
				if err != nil {
					return err
				}
				s.notifyProgress(ctx, EventActionDelete, remoteNameCopy, 1.0)
				return nil
			})
		}
		for _, localRelativePath := range d.put {
			// Copy of localName created to make this safe for concurrent use.
			localRelativePathCopy := localRelativePath
			g.Go(func() error {
				s.notifyProgress(ctx, EventActionPut, localRelativePathCopy, 0.0)
				err := s.repoFiles.PutFile(ctx, localRelativePathCopy)
				if err != nil {
					return err
				}
				s.notifyProgress(ctx, EventActionPut, localRelativePathCopy, 1.0)
				return nil
			})
		}
		// wait for goroutines to finish and return first non-nil error return
		// if any
		if err := g.Wait(); err != nil {
			return err
		}
		return nil
	}
}
