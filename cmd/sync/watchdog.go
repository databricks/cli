package sync

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/databricks/bricks/cmd/sync/repofiles"
	"github.com/databricks/bricks/project"
	"golang.org/x/sync/errgroup"
)

// TODO: add .databricks to .gitignore on bricks init
type watchdog struct {
	ticker  *time.Ticker
	wg      sync.WaitGroup
	failure error // data race? make channel?
}

// See https://docs.databricks.com/resources/limits.html#limits-api-rate-limits for per api
// rate limits
const MaxRequestsInFlight = 20

func syncCallback(ctx context.Context, repoFiles *repofiles.RepoFiles) func(localDiff diff) error {
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
				err := repoFiles.DeleteFile(ctx, remoteNameCopy)
				if err != nil {
					return err
				}
				log.Printf("[INFO] Deleted %s", remoteNameCopy)
				return nil
			})
		}
		for _, localRelativePath := range d.put {
			// Copy of localName created to make this safe for concurrent use.
			localRelativePathCopy := localRelativePath
			g.Go(func() error {
				err := repoFiles.PutFile(ctx, localRelativePathCopy)
				// if err == repofiles.ErrorFileIsIgnored {
				// 	log.Printf("[INFO] Ignored for upload %s", localRelativePathCopy)
				// 	return nil
				// }
				if err != nil {
					return err
				}
				log.Printf("[INFO] Uploaded %s", localRelativePathCopy)
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

func spawnWatchdog(ctx context.Context,
	interval time.Duration,
	applyDiff func(diff) error,
	remotePath string) error {
	w := &watchdog{
		ticker: time.NewTicker(interval),
	}
	w.wg.Add(1)
	go w.main(ctx, applyDiff, remotePath)
	w.wg.Wait()
	return w.failure
}

// tradeoff: doing portable monitoring only due to macOS max descriptor manual ulimit setting requirement
// https://github.com/gorakhargosh/watchdog/blob/master/src/watchdog/observers/kqueue.py#L394-L418
func (w *watchdog) main(ctx context.Context, applyDiff func(diff) error, remotePath string) {
	defer w.wg.Done()
	snapshot, err := newSnapshot(ctx, remotePath)
	if err != nil {
		log.Printf("[ERROR] cannot create snapshot: %s", err)
		w.failure = err
		return
	}
	if *persistSnapshot {
		err := snapshot.loadSnapshot(ctx)
		if err != nil {
			log.Printf("[ERROR] cannot load snapshot: %s", err)
			w.failure = err
			return
		}
	}
	prj := project.Get(ctx)
	var onlyOnceInitLog sync.Once
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.ticker.C:
			all, err := prj.GetFileSet().All()
			if err != nil {
				log.Printf("[ERROR] cannot list files: %s", err)
				w.failure = err
				return
			}
			change, err := snapshot.diff(all)
			if err != nil {
				w.failure = err
				return
			}
			if change.IsEmpty() {
				onlyOnceInitLog.Do(func() {
					log.Printf("[INFO] Initial Sync Complete")
				})
				continue
			}
			log.Printf("[INFO] Action: %v", change)
			err = applyDiff(change)
			if err != nil {
				w.failure = err
				return
			}
			if *persistSnapshot {
				err = snapshot.storeSnapshot(ctx)
				if err != nil {
					log.Printf("[ERROR] cannot store snapshot: %s", err)
					w.failure = err
					return
				}
			}
			onlyOnceInitLog.Do(func() {
				log.Printf("[INFO] Initial Sync Complete")
			})
		}
	}
}
