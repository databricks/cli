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

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	sdkapierr "github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/sdk-go/core/apierr"
	"github.com/databricks/sdk-go/core/apiretry"
)

// maxListedFailures is how many failed files the end-of-run summary lists.
const maxListedFailures = 10

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
//   - Once the copy stops, in-flight files are cancelled and reported as
//     failed. The walk goes on only to count the files left, which are
//     reported as not attempted.
type recursiveCopy struct {
	c *copy

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

	// fatalErr is the error that stopped the copy, if any.
	fatalErr error
	// walkCutShort is set when the walk ended on an error, so notAttempted
	// leaves out files it never reached.
	walkCutShort bool
}

func (r *recursiveCopy) run(ctx context.Context, sourceDir, targetDir string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	r.cancel = cancel
	r.slots = make(chan struct{}, r.c.concurrency)

	sourceFs := filer.NewFS(ctx, r.c.sourceFiler)
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
		return r.c.targetFiler.Mkdir(ctx, targetPath)
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

// start copies job in the background once a slot is free, or records it as not
// started if the copy is stopping.
func (r *recursiveCopy) start(ctx context.Context, job copyJob) {
	// Check again after waiting for a slot: the copy may have stopped.
	if r.stopping(ctx) {
		r.notStarted(job)
		return
	}
	r.slots <- struct{}{}
	if r.stopping(ctx) {
		<-r.slots
		r.notStarted(job)
		return
	}
	r.inflight.Go(func() {
		defer func() { <-r.slots }()
		r.copyOne(ctx, job)
	})
}

// stopping reports whether the copy should start no new files.
func (r *recursiveCopy) stopping(ctx context.Context) bool {
	// A fatal error cancels ctx.
	return ctx.Err() != nil
}

// walkFailed records an error that ended the walk. It stops the copy, unless
// the copy was already stopping and the walk was only counting files.
func (r *recursiveCopy) walkFailed(ctx context.Context, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.walkCutShort = true
	if ctx.Err() == nil && r.fatalErr == nil {
		r.fatalErr = err
		r.cancel()
	}
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

// copyOne copies one file and records the outcome.
func (r *recursiveCopy) copyOne(ctx context.Context, job copyJob) {
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
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
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
	if notCopied == 0 && r.fatalErr == nil && !cancelled {
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
	switch {
	case cancelled:
		b.WriteString("\nStopped: the copy was cancelled.")
	case r.fatalErr != nil:
		b.WriteString("\nStopped early because of the error shown below.")
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
	}
	if notCopied == 1 {
		return errors.New("1 file was not copied")
	}
	return fmt.Errorf("%d files were not copied", notCopied)
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}
