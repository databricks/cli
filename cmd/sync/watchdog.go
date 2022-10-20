package sync

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/databricks/apierr"
	"github.com/databricks/databricks-sdk-go/databricks/client"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/databricks/databricks-sdk-go/workspaces"
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

func putFile(ctx context.Context, path string, content io.Reader) error {
	wsc := project.Get(ctx).WorkspacesClient()
	// workspace mkdirs is idempotent
	err := wsc.Workspace.MkdirsByPath(ctx, filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("could not mkdir to put file: %s", err)
	}
	apiClient := client.New(wsc.Config)
	apiPath := fmt.Sprintf(
		"/api/2.0/workspace-files/import-file/%s?overwrite=true",
		strings.TrimLeft(path, "/"))
	return apiClient.Post(ctx, apiPath, content, nil)
}

func deleteFile(ctx context.Context, path string) error {
	wsc := project.Get(ctx).WorkspacesClient()
	err := wsc.Workspace.Delete(ctx,
		workspace.Delete{
			Path:      path,
			Recursive: true,
		},
	)
	// We explictly ignore RESOURCE_DOES_NOT_EXIST errors for deletion of files
	// This makes deletion operation idempotent and allows us to not crash syncing on
	// edge cases for eg: this api fails to delete notebooks, and returns a
	// RESOURCE_DOES_NOT_EXIST error instead
	if val, ok := err.(apierr.APIError); ok && val.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
		return nil
	}
	return err
}

func getRemoteSyncCallback(ctx context.Context, root, remoteDir string, wsc *workspaces.WorkspacesClient) func(localDiff diff) error {
	return func(d diff) error {

		// Abstraction over wait groups which allows you to get the errors
		// returned in goroutines
		var g errgroup.Group

		// Allow MaxRequestLimit maxiumum concurrent api calls
		g.SetLimit(MaxRequestsInFlight)

		for _, fileName := range d.delete {
			// Copy of fileName created to make this safe for concurrent use.
			// directly using fileName can cause race conditions since the loop
			// might iterate over to the next fileName before the go routine function
			// is evaluated
			localFileName := fileName
			g.Go(func() error {
				err := deleteFile(ctx, path.Join(remoteDir, localFileName))
				if err != nil {
					return err
				}
				log.Printf("[INFO] Deleted %s", localFileName)
				return nil
			})
		}
		for _, fileName := range d.put {
			localFileName := fileName
			g.Go(func() error {
				f, err := os.Open(filepath.Join(root, localFileName))
				if err != nil {
					return err
				}
				err = putFile(ctx, path.Join(remoteDir, localFileName), f)
				if err != nil {
					return fmt.Errorf("failed to upload file: %s", err)
				}
				err = f.Close()
				if err != nil {
					return err
				}
				log.Printf("[INFO] Uploaded %s", localFileName)
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

func spawnSyncRoutine(ctx context.Context,
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
			change := snapshot.diff(all)
			if change.IsEmpty() {
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
		}
	}
}
