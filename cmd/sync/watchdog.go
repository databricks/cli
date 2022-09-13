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
)

type watchdog struct {
	files   git.FileSet
	ticker  *time.Ticker
	wg      sync.WaitGroup
	failure error // data race? make channel?
}

func putFile(ctx context.Context, path string, content io.Reader) error {
	wsc := project.Current.WorkspacesClient()
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

func getRemoteSyncCallback(ctx context.Context, remoteDir string, wsc *workspaces.WorkspacesClient) func(localDiff diff) error {
	return func(d diff) error {
		for _, filePath := range d.delete {
			err := wsc.Workspace.Delete(ctx,
				workspace.DeleteRequest{
					Path:      path.Join(remoteDir, filePath),
					Recursive: true,
				},
			)
			if err != nil {
				return err
			}
			log.Printf("[INFO] Deleted %s", filePath)
		}
		for _, filePath := range d.put {
			f, err := os.Open(filePath)
			if err != nil {
				return err
			}
			err = putFile(ctx, path.Join(remoteDir, filePath), f)
			if err != nil {
				return fmt.Errorf("failed to upload file: %s", err) // TODO: fmt.Errorf
			}
			err = f.Close()
			if err != nil {
				return err // TODO: fmt.Errorf
			}
			log.Printf("[INFO] Uploaded %s", filePath)
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
	root, _ := git.Root()
	err := state.loadSnapshot(root)
	if err != nil {
		log.Printf("[ERROR] cannot load snapshot: %s", err)
		w.failure = err
		return
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
			err = state.storeSnapshot(root)
			if err != nil {
				log.Printf("[ERROR] cannot store snapshot: %s", err)
				w.failure = err
				return
			}
		}
	}
}
