package fs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	sdkapierr "github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/sdk-go/core/apierr"
	"github.com/databricks/sdk-go/core/apiretry"
)

// maxListedFailures is how many failed files the end-of-run summary lists.
const maxListedFailures = 10

// stallTimeout is how long a recursive copy keeps going while files fail and
// none succeeds. After that it starts no new files and lets in-flight ones
// finish. It is internal, not a flag.
const stallTimeout = 5 * time.Minute

// probeBackoffMin and probeBackoffMax bound the pause before each attempt made
// while the circuit breaker is open, and before each retry of a directory
// listing or Mkdir. A Retry-After hint from the server is honored, capped at
// probeBackoffMax.
const (
	probeBackoffMin = 2 * time.Second
	probeBackoffMax = 30 * time.Second
)

// Undocumented, test-only overrides of stallTimeout and probeBackoffMax in
// milliseconds. A probeBackoffMax below probeBackoffMin lowers the minimum too.
const (
	stallTimeoutEnvVar    = "DATABRICKS_FS_CP_STALL_TIMEOUT_MS"
	probeBackoffMaxEnvVar = "DATABRICKS_FS_CP_PROBE_BACKOFF_MAX_MS"
)

// errorClass says what a recursive copy does after an error.
type errorClass int

const (
	// classFile fails only the file it happened on.
	classFile errorClass = iota
	// classRetryable defers the file, to be tried once more after the walk.
	classRetryable
	// classFatal stops the copy.
	classFatal
)

// classifyError decides what a recursive copy does after err. It runs after the
// filer's own retries, so a retryable error here is one they did not absorb.
//
//   - 401 and 403 stop the copy: every other file would fail the same way.
//   - 408, 429 and 5xx (including a 5xx with an empty or non-JSON body, which the
//     SDKs still turn into an API error carrying the status) are retryable.
//   - Other API errors (400, 404, ...) fail only the file.
//   - Transport errors (connection reset or refused, timeouts, an unexpected
//     EOF, any other *url.Error from the HTTP client) are retryable.
//   - Cancellation of ctx stops the copy.
//   - Anything else, e.g. an error reading a local file, fails only the file.
func classifyError(ctx context.Context, err error) errorClass {
	if ctx.Err() != nil {
		return classFatal
	}
	if status, _, ok := httpError(err); ok {
		switch {
		case status == http.StatusUnauthorized || status == http.StatusForbidden:
			return classFatal
		case status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= 500:
			return classRetryable
		default:
			return classFile
		}
	}
	// ctx is alive, so a context error comes from a timeout on one request.
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
		apiretry.IsTransientNetworkError(err) || errors.Is(err, io.ErrUnexpectedEOF) {
		return classRetryable
	}
	if _, ok := errors.AsType[*url.Error](err); ok {
		return classRetryable
	}
	if _, ok := errors.AsType[net.Error](err); ok {
		return classRetryable
	}
	return classFile
}

// httpError returns the HTTP status and server message of an API error from
// either SDK: the Files API (Volumes) uses sdk-go, DBFS uses databricks-sdk-go.
func httpError(err error) (int, string, bool) {
	if aerr, ok := errors.AsType[*apierr.APIError](err); ok {
		return aerr.HTTPStatusCode(), aerr.Message(), true
	}
	if aerr, ok := errors.AsType[*sdkapierr.APIError](err); ok {
		return aerr.StatusCode, aerr.Message, true
	}
	return 0, "", false
}

// describeError adds the HTTP status to an API error. A 5xx from a proxy often
// has no JSON body, and then the error has no message of its own.
func describeError(err error) string {
	status, message, ok := httpError(err)
	if !ok {
		return err.Error()
	}
	statusText := fmt.Sprintf("HTTP %d %s", status, http.StatusText(status))
	if message == "" {
		return statusText
	}
	return fmt.Sprintf("%s (%s)", err, statusText)
}

// copyJob is one file to copy.
type copyJob struct {
	sourcePath string
	targetPath string

	// final is set on the deferred pass: a retryable failure then fails the
	// file instead of deferring it again.
	final bool

	// err is the error of the job's last attempt: the one that deferred it,
	// or the one it failed with.
	err error
}

// recursiveCopy runs one `fs cp -r`. It copies up to --concurrency files at a
// time and keeps going past failures:
//
//   - A retryable failure defers the file. After the walk, every deferred file
//     gets one more full attempt, with its source reopened.
//   - A failure that is not retryable fails only that file, except that 401,
//     403, cancellation and a failure to print stop the copy.
//   - After --concurrency consecutive retryable failures, a circuit breaker
//     opens. While it is open no new files are started. Instead, one deferred
//     file at a time is tried, with a backoff between attempts. The first
//     success closes it. A failed probe defers its file again, so during an
//     outage a file can be tried more than twice, until the stall timeout.
//   - If files keep failing and none has been copied for the stall timeout,
//     no new files are started and in-flight ones finish. A slow copy with no
//     failures never stalls.
//   - When a fatal error or a cancellation stops the copy, in-flight files are
//     cancelled and reported as failed.
//   - A retryable error listing a source directory or creating a target one is
//     retried in place, for up to the stall timeout.
//   - Once the copy stops, the walk goes on only to count the files left, which
//     are reported as not attempted.
type recursiveCopy struct {
	c *copy

	// now and sleep are the clock, injectable for tests.
	now             func() time.Time
	sleep           func(ctx context.Context, d time.Duration) error
	stallTimeout    time.Duration
	probeBackoffMin time.Duration
	probeBackoffMax time.Duration

	// cancel stops in-flight copies after a fatal error.
	cancel context.CancelFunc
	// slots holds one token per running copy, so at most --concurrency files
	// are copied at once.
	slots    chan struct{}
	inflight sync.WaitGroup

	mu           sync.Mutex
	copied       int
	skipped      int
	notAttempted int
	failures     []copyJob
	deferred     []copyJob

	lastSuccess time.Time
	// failingSince is when a file first failed with a retryable error since
	// the last file success, or zero if none has. Only a copy failing like
	// that can stall, and the stall timeout runs from the later of this and
	// lastSuccess. A permanent error such as a 404 is not a sign of an outage.
	failingSince time.Time

	// consecutiveRetryable counts retryable failures since the last success.
	consecutiveRetryable int
	breakerOpen          bool
	probeBackoff         time.Duration
	// lastRetryAfter is the largest Retry-After hint since the last success.
	lastRetryAfter time.Duration

	// fatalErr is the error that stopped the copy, if any.
	fatalErr error
	// stalled is set once no file succeeded for the stall timeout.
	stalled bool
	// walkCutShort is set when the walk ended on an error, so notAttempted
	// leaves out files it never reached.
	walkCutShort bool
}

func newRecursiveCopy(ctx context.Context, c *copy) *recursiveCopy {
	r := &recursiveCopy{
		c:               c,
		now:             time.Now,
		sleep:           sleepContext,
		stallTimeout:    durationFromEnv(ctx, stallTimeoutEnvVar, stallTimeout),
		probeBackoffMax: durationFromEnv(ctx, probeBackoffMaxEnvVar, probeBackoffMax),
	}
	r.probeBackoffMin = min(probeBackoffMin, r.probeBackoffMax)
	return r
}

// durationFromEnv reads a duration in milliseconds from the environment
// variable key, falling back to def if it is unset or invalid.
func durationFromEnv(ctx context.Context, key string, def time.Duration) time.Duration {
	ms, err := strconv.Atoi(env.Get(ctx, key))
	if err != nil || ms <= 0 {
		return def
	}
	return time.Duration(ms) * time.Millisecond
}

func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (r *recursiveCopy) run(ctx context.Context, sourceDir, targetDir string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	r.cancel = cancel
	r.lastSuccess = r.now()
	r.slots = make(chan struct{}, r.c.concurrency)

	sourceFs := filer.NewFS(ctx, &walkRetryFiler{Filer: r.c.sourceFiler, r: r})
	walkErr := fs.WalkDir(sourceFs, sourceDir, func(sourcePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(sourceDir, sourcePath)
		if err != nil {
			return err
		}
		targetPath := path.Join(targetDir, filepath.ToSlash(relPath))
		if !d.IsDir() {
			r.start(ctx, copyJob{sourcePath: sourcePath, targetPath: targetPath})
			return nil
		}
		// Once the copy is stopping, the walk only counts the files left.
		if r.stopping(ctx) {
			return nil
		}
		// Create the directory before the files in it. WalkDir visits a
		// directory before its contents.
		if err := r.retryWalk(ctx, "create directory "+targetPath, func() error {
			return r.c.targetFiler.Mkdir(ctx, targetPath)
		}); err != nil {
			return err
		}
		// A created directory is progress too, so a tree of slow directories
		// does not stall.
		r.progress()
		return nil
	})
	if walkErr != nil {
		r.walkFailed(ctx, walkErr)
	}
	r.inflight.Wait()

	// Give every deferred file one more attempt.
	for job, ok := r.popDeferred(); ok; job, ok = r.popDeferred() {
		job.final = true
		r.start(ctx, job)
	}
	r.inflight.Wait()
	return r.finish(ctx)
}

// start copies job in the background once a slot is free and the breaker
// admits it, or records it as not started if the copy is stopping.
func (r *recursiveCopy) start(ctx context.Context, job copyJob) {
	if r.stopping(ctx) {
		r.notStarted(job)
		return
	}
	// Wait for a slot before checking the breaker, so the outcome of the
	// file before is known.
	r.slots <- struct{}{}
	if !r.admit(ctx, job) {
		<-r.slots
		return
	}
	r.inflight.Go(func() {
		defer func() { <-r.slots }()
		_ = r.copyOne(ctx, job)
	})
}

// admit waits until job may start, and reports whether it should. It runs on
// the walker goroutine, so probing while the breaker is open is serial by
// construction. While the
// breaker is open, it probes one deferred file at a time, after a backoff,
// until one succeeds or the copy stops. On the deferred pass, or when nothing
// is deferred, job itself is the probe, and then admit has copied it. If the
// copy stops, admit records job as not started.
func (r *recursiveCopy) admit(ctx context.Context, job copyJob) bool {
	for {
		if r.stopping(ctx) {
			r.notStarted(job)
			return false
		}
		delay, open := r.breakerDelay()
		if !open {
			return true
		}
		if r.sleep(ctx, delay) != nil || r.stopping(ctx) {
			r.notStarted(job)
			return false
		}
		probe, deferred := job, false
		if !job.final {
			if p, ok := r.popDeferred(); ok {
				probe, deferred = p, true
			}
		}
		log.Debugf(ctx, "fs cp: breaker open, probing with %s after waiting %s", probe.sourcePath, delay)
		if err := r.copyOne(ctx, probe); err != nil {
			log.Debugf(ctx, "fs cp: probe with %s failed: %s", probe.sourcePath, describeError(err))
		} else {
			log.Debugf(ctx, "fs cp: probe with %s succeeded", probe.sourcePath)
		}
		if !deferred {
			return false
		}
	}
}

// breakerDelay reports whether the breaker is open and, if it is, how long to
// wait before the next probe. Each call doubles the backoff for the next one.
func (r *recursiveCopy) breakerDelay() (time.Duration, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.breakerOpen {
		return 0, false
	}
	delay := r.probeBackoff
	r.probeBackoff = min(r.probeBackoff*2, r.probeBackoffMax)
	return min(max(delay, r.lastRetryAfter), r.probeBackoffMax), true
}

// retryAfter returns the server's Retry-After hint for err, or 0.
func retryAfter(err error) time.Duration {
	if d := apiretry.RetryDurationHint(err); d > 0 {
		return d
	}
	aerr, ok := errors.AsType[*sdkapierr.APIError](err)
	if !ok || aerr.ResponseWrapper == nil || aerr.ResponseWrapper.Response == nil {
		return 0
	}
	seconds, err := strconv.Atoi(aerr.ResponseWrapper.Response.Header.Get("Retry-After"))
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

// stopping reports whether the copy should start no new files: after a fatal
// error, which cancels ctx, or a stall.
func (r *recursiveCopy) stopping(ctx context.Context) bool {
	if ctx.Err() != nil {
		return true
	}
	r.mu.Lock()
	since := r.failingSince
	if r.lastSuccess.After(since) {
		since = r.lastSuccess
	}
	stalledNow := !r.stalled && !r.failingSince.IsZero() && r.now().Sub(since) >= r.stallTimeout
	if stalledNow {
		r.stalled = true
	}
	stalled := r.stalled
	r.mu.Unlock()
	if stalledNow {
		log.Infof(ctx, "fs cp: files kept failing and none was copied for %s; starting no new files", formatDuration(r.stallTimeout))
	}
	return stalled
}

// walkFailed records an error that ended the walk. It stops the copy, unless
// the copy was already stopping and the walk was only counting files.
func (r *recursiveCopy) walkFailed(ctx context.Context, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.walkCutShort = true
	if ctx.Err() == nil && r.fatalErr == nil && !r.stalled {
		r.fatalErr = err
		r.cancel()
		return
	}
	log.Debugf(ctx, "fs cp: the walk stopped counting files: %s", describeError(err))
}

// progress records that the copy created a directory, which resets the stall
// timer. Only a file success clears failingSince.
func (r *recursiveCopy) progress() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastSuccess = r.now()
}

func (r *recursiveCopy) popDeferred() (copyJob, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.deferred) == 0 {
		return copyJob{}, false
	}
	job := r.deferred[0]
	r.deferred = r.deferred[1:]
	return job, true
}

// notStarted records a job that was not started because the copy is stopping.
// A deferred job already failed once, so it counts as failed with that error.
func (r *recursiveCopy) notStarted(job copyJob) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if job.err != nil {
		r.failures = append(r.failures, job)
		return
	}
	r.notAttempted++
}

// errCancelledMidCopy fails a file whose copy was cancelled because the copy
// stopped. On DBFS it may have left a partial target.
var errCancelledMidCopy = errors.New("cancelled: the copy stopped before this file finished")

// errPartialTarget fails a retried file whose target exists with another size.
var errPartialTarget = errors.New("target exists with a different size; an earlier attempt may have left a partial file. Rerun with --overwrite to replace it")

// copyOne copies one file, records the outcome and returns its error.
func (r *recursiveCopy) copyOne(ctx context.Context, job copyJob) error {
	// The walk already created the target directory, so skip the filer's
	// per-file parent-directory check.
	skipped, err := r.c.writeFile(ctx, job.sourcePath, job.targetPath, filer.CreateParentDirectories)

	class := classFile
	switch {
	case err != nil:
		class = classifyError(ctx, err)
	case skipped && job.err != nil:
		// Without --overwrite, a retry finds whatever an earlier attempt of
		// ours left behind. On Volumes a single-shot upload is one atomic PUT,
		// so that is the complete file. A DBFS write streams blocks into an
		// open handle, so a failed attempt can leave a truncated prefix, and
		// a size check catches it. A partial of the same size is not
		// detected, but a truncated prefix is shorter unless the failure
		// came after the last block. Any error here fails the file rather
		// than guessing that it was copied, though a 401, 403 or cancellation
		// still stops the copy.
		if err = r.checkExistingTarget(ctx, job); err != nil {
			class = classifyError(ctx, err)
			if class == classRetryable {
				class = classFile
			}
		}
	}
	if err == nil {
		// Failing to print means the output is gone (e.g. a closed pipe);
		// every other file would fail the same way.
		if err = r.c.emitResult(ctx, job.sourcePath, job.targetPath, skipped); err != nil {
			class = classFatal
		}
	}
	if class == classFatal && ctx.Err() != nil {
		// Cancelled because the copy is stopping; not this file's fault. A
		// deferred file keeps its earlier error.
		if job.err == nil {
			job.err = errCancelledMidCopy
		}
		r.notStarted(job)
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case err == nil:
		r.lastSuccess = r.now()
		r.failingSince = time.Time{}
		r.consecutiveRetryable = 0
		r.lastRetryAfter = 0
		if r.breakerOpen {
			r.breakerOpen = false
			log.Infof(ctx, "fs cp: breaker closed: %s was copied; starting new files again", job.sourcePath)
		}
	case class == classRetryable && r.failingSince.IsZero():
		r.failingSince = r.now()
	}
	if class == classRetryable {
		r.consecutiveRetryable++
		r.lastRetryAfter = max(r.lastRetryAfter, retryAfter(err))
		if !r.breakerOpen && r.consecutiveRetryable >= r.c.concurrency {
			r.breakerOpen = true
			r.probeBackoff = r.probeBackoffMin
			log.Infof(ctx, "fs cp: breaker opened after %d retryable failures in a row (last: %s); trying one file at a time", r.consecutiveRetryable, describeError(err))
		}
	}
	switch {
	case err == nil && skipped:
		r.skipped++
	case err == nil:
		r.copied++
	case class == classRetryable && !job.final:
		log.Debugf(ctx, "fs cp: deferring %s: %s", job.sourcePath, describeError(err))
		job.err = err
		r.deferred = append(r.deferred, job)
	default:
		if class == classRetryable {
			log.Debugf(ctx, "fs cp: %s failed on its retry: %s", job.sourcePath, describeError(err))
		}
		job.err = err
		r.failures = append(r.failures, job)
		if class == classFatal && r.fatalErr == nil {
			r.fatalErr = err
			r.cancel()
		}
	}
	return err
}

// checkExistingTarget returns errPartialTarget if the target's size differs from
// the source's, or the error from either Stat.
func (r *recursiveCopy) checkExistingTarget(ctx context.Context, job copyJob) error {
	source, err := r.c.sourceFiler.Stat(ctx, job.sourcePath)
	if err != nil {
		return err
	}
	target, err := r.c.targetFiler.Stat(ctx, job.targetPath)
	if err != nil {
		return err
	}
	if source.Size() != target.Size() {
		return errPartialTarget
	}
	return nil
}

// retryWalk runs op, a directory listing or Mkdir, retrying a retryable error
// in place with the probe backoff for up to the stall timeout from the first
// attempt. The walk is sequential, so this makes one call at a time.
func (r *recursiveCopy) retryWalk(ctx context.Context, what string, op func() error) error {
	start := r.now()
	backoff := r.probeBackoffMin
	for retried := false; ; retried = true {
		err := op()
		if err == nil {
			// A recovered call is progress, so its retries do not use up the
			// stall timeout of the files that follow.
			if retried {
				r.progress()
			}
			return nil
		}
		// No retries while a stalled copy only counts the files left.
		if classifyError(ctx, err) != classRetryable || r.isStalled() {
			return err
		}
		remaining := r.stallTimeout - r.now().Sub(start)
		if remaining <= 0 {
			return err
		}
		delay := min(max(backoff, retryAfter(err)), r.probeBackoffMax, remaining)
		backoff = min(backoff*2, r.probeBackoffMax)
		log.Debugf(ctx, "fs cp: retrying %s in %s after: %s", what, delay, describeError(err))
		if r.sleep(ctx, delay) != nil {
			return err
		}
	}
}

func (r *recursiveCopy) isStalled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stalled
}

// walkRetryFiler retries the calls the walk makes on the source: a Stat of
// the root and a ReadDir of every directory.
type walkRetryFiler struct {
	filer.Filer
	r *recursiveCopy
}

func (w *walkRetryFiler) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	var entries []fs.DirEntry
	err := w.r.retryWalk(ctx, "list "+name, func() error {
		var err error
		entries, err = w.Filer.ReadDir(ctx, name)
		return err
	})
	return entries, err
}

func (w *walkRetryFiler) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	var info fs.FileInfo
	err := w.r.retryWalk(ctx, "stat "+name, func() error {
		var err error
		info, err = w.Filer.Stat(ctx, name)
		return err
	})
	return info, err
}

// finish prints the end-of-run summary to stderr and returns the command's
// error. A complete copy gets one line, which JSON output omits. An incomplete
// copy gets the counts, the reason it stopped early, if it did, and the first
// failures.
func (r *recursiveCopy) finish(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// The copy cancels its own context only after a fatal error, so any other
	// cancellation came from the caller.
	cancelled := ctx.Err() != nil && r.fatalErr == nil
	notCopied := len(r.failures) + r.notAttempted
	if notCopied == 0 && r.fatalErr == nil && !cancelled && !r.stalled {
		if !r.c.outputJSON {
			line := "Copied " + plural(r.copied, "file")
			if r.skipped > 0 {
				line += fmt.Sprintf(", skipped %d that already existed", r.skipped)
			}
			cmdio.LogString(ctx, line+".")
		}
		return nil
	}

	notAttempted := strconv.Itoa(r.notAttempted)
	if r.walkCutShort {
		notAttempted = "at least " + notAttempted
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Copy incomplete: %d copied, %d skipped (already existed), %d failed, %s not attempted.",
		r.copied, r.skipped, len(r.failures), notAttempted)
	// The same order as the returned error, so the reason matches it.
	switch {
	case r.fatalErr != nil:
		b.WriteString("\nStopped early because of the error shown below.")
	case cancelled:
		b.WriteString("\nStopped: the copy was cancelled.")
	case r.stalled:
		fmt.Fprintf(&b, "\nStopped: files kept failing and none was copied for %s, so no new files were started.", formatDuration(r.stallTimeout))
	}
	if len(r.failures) > 0 {
		b.WriteString("\nFailed:")
		for _, job := range r.failures[:min(len(r.failures), maxListedFailures)] {
			source, target := r.c.fullPaths(job.sourcePath, job.targetPath)
			fmt.Fprintf(&b, "\n  %s -> %s: %s", source, target, describeError(job.err))
		}
		if n := len(r.failures) - maxListedFailures; n > 0 {
			fmt.Fprintf(&b, "\n  ... and %d more", n)
		}
	}
	cmdio.LogString(ctx, b.String())

	switch {
	case r.fatalErr != nil:
		return r.fatalErr
	case cancelled:
		return fmt.Errorf("copy cancelled: %w", ctx.Err())
	case r.stalled:
		return fmt.Errorf("copy stopped: files kept failing and none was copied for %s", formatDuration(r.stallTimeout))
	}
	if notCopied == 1 {
		return errors.New("1 file was not copied")
	}
	return fmt.Errorf("%d files were not copied", notCopied)
}

// formatDuration writes d as "5 minutes" or "2 seconds" when it is a whole
// number of them.
func formatDuration(d time.Duration) string {
	switch {
	case d >= time.Minute && d%time.Minute == 0:
		return plural(int(d/time.Minute), "minute")
	case d >= time.Second && d%time.Second == 0:
		return plural(int(d/time.Second), "second")
	}
	return d.String()
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}
