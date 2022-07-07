package sync

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/databricks/bricks/git"
)

type watchdog struct {
	files   git.FileSet
	ticker  *time.Ticker
	wg      sync.WaitGroup
	failure error // data race? make channel?
}

func watchForChanges(ctx context.Context, files git.FileSet,
	interval time.Duration, cb func(diff) error) error {
	w := &watchdog{
		files:  files,
		ticker: time.NewTicker(interval),
	}
	w.wg.Add(1)
	go w.main(ctx, cb)
	w.wg.Wait()
	return w.failure
}

// tradeoff: doing portable monitoring only due to macOS max descriptor manual ulimit setting requirement
// https://github.com/gorakhargosh/watchdog/blob/master/src/watchdog/observers/kqueue.py#L394-L418
func (w *watchdog) main(ctx context.Context, cb func(diff) error) {
	defer w.wg.Done()
	// load from json or sync it every time there's an action
	state := snapshot{}
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
			err = cb(change)
			if err != nil {
				w.failure = err
				return
			}
		}
	}
}
