package sync

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
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
	apiClient, err := client.New(wsc.Config)
	if err != nil {
		return err
	}
	apiPath := fmt.Sprintf(
		"/api/2.0/workspace-files/import-file/%s?overwrite=true",
		strings.TrimLeft(path, "/"))
	return apiClient.Post(ctx, apiPath, content, nil)
}

func deleteFile(ctx context.Context, path string, wsc *workspaces.WorkspacesClient) error {
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

// Python notebooks when imported to a workspace filetree strip “.py” from the
// file name which causes our APIs to have unexpected behavior. If you import a
// notebook called `foo.py`, it will be referred to as `foo` in the workspace file
// tree. This causes the following inconsistent behaviors:
//
// a. Consider you have a python notebook foo.py. If you query “PUT foo.py” then
//    “DELETE foo.py” will return a RESOURCE_DOES_NOT_EXIST error.
//    You would need  “DELETE foo” to delete the notebook
//
// b. Consider you have a python notebook foo.py and a file foo. If you query “PUT foo.py”
//    and “PUT foo” then one of the PUT queries will fail with a file already exists error
//
// This functions does 4 things
//
// 1. Errors out if a file with the same name (without .py) exists as the
//    notebook (to deal with problem b)
// 2. Tracks all python files that are notebooks. This is later used to safely
//    delete notebooks without duplication
// 3. Delete the remote python file if a notebook -> python file
// 4. Delete the remote notebook if a python file -> notebook
func dedupDatabricksNotebook(ctx context.Context, localRoot, localFileName, remoteRoot string, snapshot *Snapshot, wsc *workspaces.WorkspacesClient) error {
	f, err := os.Open(filepath.Join(localRoot, localFileName))
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	// A python file is a notebook if it contains the following magic string
	isNotebook := strings.Contains(scanner.Text(), "# Databricks notebook source")
	_, isKnownNotebook := snapshot.NotebookLocalToRemoteNames[localFileName]
	_, hasBeenSyncedOnce := snapshot.LastUpdatedTimes[localFileName]

	// File names of databricks source notebooks when uploaded
	// are stripped of their .py suffix in the remote workspace filetree
	remoteFileName := strings.TrimSuffix(localFileName, `.py`)

	// Notebook has the same name as a file
	if _, ok := snapshot.LastUpdatedTimes[remoteFileName]; ok {
		return fmt.Errorf("file %s and notebook %s simulatanesly reside in project. Please remove one of them", remoteFileName, localFileName)
	}
	// track new databricks notebook
	if isNotebook && !isKnownNotebook && !hasBeenSyncedOnce {
		snapshot.NotebookLocalToRemoteNames[localFileName] = remoteFileName
	}
	// A python file that has been converted to a databricks notebook
	if isNotebook && !isKnownNotebook && hasBeenSyncedOnce {
		snapshot.NotebookLocalToRemoteNames[localFileName] = remoteFileName
		// delete remote python file
		err := deleteFile(ctx, path.Join(remoteRoot, localFileName), wsc)
		if err != nil {
			return err
		}
	}
	// A local databricks notebook has been converted to a python file
	if !isNotebook && isKnownNotebook {
		delete(snapshot.NotebookLocalToRemoteNames, localFileName)
		// delete remote notebook
		err := deleteFile(ctx, path.Join(remoteRoot, remoteFileName), wsc)
		if err != nil {
			return err
		}
	}
	return nil
}

func getRemoteSyncCallback(ctx context.Context, root, remoteDir string, wsc *workspaces.WorkspacesClient) func(localDiff diff, s *Snapshot) error {
	return func(d diff, snapshot *Snapshot) error {

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
				remoteFileName, isNotebook := snapshot.NotebookLocalToRemoteNames[localFileName]
				if isNotebook {
					delete(snapshot.NotebookLocalToRemoteNames, localFileName)
				} else {
					remoteFileName = localFileName
				}
				err := deleteFile(ctx, path.Join(remoteDir, remoteFileName), wsc)
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
				isPythonFile, err := regexp.Match(`\.py$`, []byte(localFileName))
				if err != nil {
					return err
				}
				if isPythonFile {
					err = dedupDatabricksNotebook(ctx, root, localFileName, remoteDir, snapshot, wsc)
					if err != nil {
						return err
					}
				} else {
					if _, ok := snapshot.NotebookLocalToRemoteNames[localFileName+".py"]; ok {
						return fmt.Errorf("file %s and notebook %s simulatanesly reside in project. Please remove one of them", localFileName, localFileName+".py")
					}
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
	applyDiff func(diff, *Snapshot) error,
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
func (w *watchdog) main(ctx context.Context, applyDiff func(diff, *Snapshot) error, remotePath string) {
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
			change := snapshot.diff(all)
			if change.IsEmpty() {
				onlyOnceInitLog.Do(func() {
					log.Printf("[INFO] Initial Sync Complete")
				})
				continue
			}
			log.Printf("[INFO] Action: %v", change)
			err = applyDiff(change, snapshot)
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
