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

	"github.com/databricks/bricks/git"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/databricks/client"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/databricks/databricks-sdk-go/workspaces"
	"golang.org/x/sync/errgroup"
)

type watchdog struct {
	files   git.FileSet
	ticker  *time.Ticker
	wg      sync.WaitGroup
	failure error // data race? make channel?
}

const MaxRequestLimit = 30

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

func getRemoteSyncCallback(ctx context.Context, root, remoteDir string, wsc *workspaces.WorkspacesClient) func(localDiff diff) error {
	return func(d diff) error {

		// Abstraction over wait groups which allows you to get the errors
		// returned in goroutines
		var g errgroup.Group

		// Allow MaxRequestLimit maxumim concurrent api calls
		g.SetLimit(MaxRequestLimit)

		for _, fileName := range d.delete {
			// Copy of fileName created to make this safe for concurrent use.
			// directly using fileName can cause race conditions since the loop
			// might iterate over to the next fileName before the go routine function
			// is evaluated
			localFileName := fileName
			g.Go(func() error {
				wsc := project.Get(ctx).WorkspacesClient()
				err := wsc.Workspace.Delete(ctx,
					workspace.Delete{
						Path:      path.Join(remoteDir, localFileName),
						Recursive: true,
					},
				)
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
	files git.FileSet,
	interval time.Duration,
	applyDiff func(diff) error) error {
	w := &watchdog{
		files:  files,
		ticker: time.NewTicker(interval),
	}
	w.wg.Add(1)
	go w.main(ctx, applyDiff)
	w.wg.Wait()
	return w.failure
}

// tradeoff: doing portable monitoring only due to macOS max descriptor manual ulimit setting requirement
// https://github.com/gorakhargosh/watchdog/blob/master/src/watchdog/observers/kqueue.py#L394-L418
func (w *watchdog) main(ctx context.Context, applyDiff func(diff) error) {
	defer w.wg.Done()
	// load from json or sync it every time there's an action
	state := snapshot{}
	root := w.files.Root()
	if *persistSnapshot {
		err := state.loadSnapshot(root)
		if err != nil {
			log.Printf("[ERROR] cannot load snapshot: %s", err)
			w.failure = err
			return
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.ticker.C:
			all, err := w.files.All()
			if err != nil {
				log.Printf("[ERROR] cannot list files: %s", err)
				w.failure = err
				return
			}
			change := state.diff(all)
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
				err = state.storeSnapshot(root)
				if err != nil {
					log.Printf("[ERROR] cannot store snapshot: %s", err)
					w.failure = err
					return
				}
			}
		}
	}
}
