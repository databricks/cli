package retries

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/databricks/bricks/libs/log"
)

type Err struct {
	Err  error
	Halt bool
}

func Halt(err error) *Err {
	return &Err{err, true}
}

func Continue(err error) *Err {
	return &Err{err, false}
}

func Continues(msg string) *Err {
	return Continue(fmt.Errorf(msg))
}

func Continuef(format string, err error, args ...interface{}) *Err {
	wrapped := fmt.Errorf(format, append([]interface{}{err}, args...))
	return Continue(wrapped)
}

type WaitFn func() *Err

var maxWait = 10 * time.Second
var minJitter = 50 * time.Millisecond
var maxJitter = 750 * time.Millisecond

func Wait(pctx context.Context, timeout time.Duration, fn WaitFn) error {
	ctx, cancel := context.WithTimeout(pctx, timeout)
	defer cancel()
	var attempt int
	var lastErr error
	for {
		attempt++
		res := fn()
		if res == nil {
			return nil
		}
		if res.Halt {
			return res.Err
		}
		lastErr = res.Err
		wait := time.Duration(attempt) * time.Second
		if wait > maxWait {
			wait = maxWait
		}
		// add some random jitter
		rand.Seed(time.Now().UnixNano())
		jitter := rand.Intn(int(maxJitter)-int(minJitter)+1) + int(minJitter)
		wait += time.Duration(jitter)
		timer := time.NewTimer(wait)
		log.Tracef(ctx, "%s. Sleeping %s",
			strings.TrimSuffix(res.Err.Error(), "."),
			wait.Round(time.Millisecond))
		select {
		// stop when either this or parent context times out
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("timed out: %w", lastErr)
		case <-timer.C:
		}
	}
}
