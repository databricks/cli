package filer

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"sync"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// This readahead cache is designed to optimize file system operations by caching the results of
// directory reads (ReadDir) and file/directory metadata reads (Stat). This cache aims to eliminate
// redundant operations and improve performance by storing the results of these operations and
// reusing them when possible. Additionally, the cache performs readahead on ReadDir calls,
// proactively caching information about files and subdirectories to speed up future access.
//
// The cache maintains two primary maps: one for ReadDir results and another for Stat results.
// When a directory read or a stat operation is requested, the cache first checks if the result
// is already available. If it is, the cached result is returned immediately. If not, the
// operation is queued for execution, and the result is stored in the cache once the operation
// completes. In cases where the result is not immediately available, the caller may need to wait
// for the cache entry to be populated. However, because the queue is processed in order by a
// fixed number of worker goroutines, we are guaranteed that the required cache entry will be
// populated and available once the queue processes the corresponding task.
//
// The cache uses a worker pool to process the queued operations concurrently. This is
// implemented using a fixed number of worker goroutines that continually pull tasks from a
// queue and execute them. The queue itself is logically unbounded in the sense that it needs to
// accommodate all the new tasks that may be generated dynamically during the execution of ReadDir
// calls. Specifically, a single ReadDir call can add an unknown number of new Stat and ReadDir
// tasks to the queue because each directory entry may represent a file or subdirectory that
// requires further processing.
//
// For practical reasons, we are not using an unbounded queue but a channel with a maximum size
// of 10,000. This helps prevent excessive memory usage and ensures that the system remains
// responsive under load. If we encounter real examples of subtrees with more than 10,000
// elements, we can consider addressing this limitation in the future. For now, this approach
// balances the need for readahead efficiency with practical constraints.
//
// It is crucial to note that each ReadDir and Stat call is executed only once. The result of a
// Stat call can be served from the cache if the information was already returned by an earlier
// ReadDir call. This helps to avoid redundant operations and ensures that the system remains
// efficient even under high load.

const (
	kMaxQueueSize = 10_000

	// Number of worker goroutines to process the queue.
	// These workers share the same HTTP client and therefore the same rate limiter.
	// If this number is increased, the rate limiter should be modified as well.
	kNumCacheWorkers = 1
)

// queueFullError is returned when the queue is at capacity.
type queueFullError struct {
	name string
}

// Error returns the error message.
func (e queueFullError) Error() string {
	return fmt.Sprintf("queue is at capacity (%d); cannot enqueue work for %q", kMaxQueueSize, e.name)
}

// Common type for all cacheable calls.
type cacheEntry struct {
	// Channel to signal that the operation has completed.
	done chan struct{}

	// The (cleaned) name of the file or directory being operated on.
	name string

	// Return values of the operation.
	err error
}

// String returns the path of the file or directory being operated on.
func (e *cacheEntry) String() string {
	return e.name
}

// Mark this entry as errored.
func (e *cacheEntry) markError(err error) {
	e.err = err
	close(e.done)
}

// readDirEntry is the cache entry for a [ReadDir] call.
type readDirEntry struct {
	cacheEntry

	// Return values of a [ReadDir] call.
	entries []fs.DirEntry
}

// Create a new readDirEntry.
func newReadDirEntry(name string) *readDirEntry {
	return &readDirEntry{cacheEntry: cacheEntry{done: make(chan struct{}), name: name}}
}

// Execute the operation and signal completion.
func (e *readDirEntry) execute(ctx context.Context, c *cache) {
	t1 := time.Now()
	e.entries, e.err = c.f.ReadDir(ctx, e.name)
	t2 := time.Now()
	log.Tracef(ctx, "readdir for %s took %f", e.name, t2.Sub(t1).Seconds())

	// Finalize the read call by adding all directory entries to the stat cache.
	c.completeReadDir(e.name, e.entries)

	// Signal that the operation has completed.
	// The return value can now be used by routines waiting on it.
	close(e.done)
}

// Wait for the operation to complete and return the result.
func (e *readDirEntry) wait(ctx context.Context) ([]fs.DirEntry, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-e.done:
		// Note: return a copy of the slice to prevent the caller from modifying the cache.
		// The underlying elements are values (see [wsfsDirEntry]) so a shallow copy is sufficient.
		return slices.Clone(e.entries), e.err
	}
}

// statEntry is the cache entry for a [Stat] call.
type statEntry struct {
	cacheEntry

	// Return values of a [Stat] call.
	info fs.FileInfo
}

// Create a new stat entry.
func newStatEntry(name string) *statEntry {
	return &statEntry{cacheEntry: cacheEntry{done: make(chan struct{}), name: name}}
}

// Execute the operation and signal completion.
func (e *statEntry) execute(ctx context.Context, c *cache) {
	t1 := time.Now()
	e.info, e.err = c.f.Stat(ctx, e.name)
	t2 := time.Now()
	log.Tracef(ctx, "stat for %s took %f", e.name, t2.Sub(t1).Seconds())

	// Signal that the operation has completed.
	// The return value can now be used by routines waiting on it.
	close(e.done)
}

// Wait for the operation to complete and return the result.
func (e *statEntry) wait(ctx context.Context) (fs.FileInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-e.done:
		return e.info, e.err
	}
}

// Mark the stat entry as done.
func (e *statEntry) markDone(info fs.FileInfo, err error) {
	e.info = info
	e.err = err
	close(e.done)
}

// executable is the interface all cacheable calls must implement.
type executable interface {
	fmt.Stringer

	execute(ctx context.Context, c *cache)
}

// cache stores all entries for cacheable Workspace File System calls.
// We care about caching only [ReadDir] and [Stat] calls.
type cache struct {
	f Filer
	m sync.Mutex

	readDir map[string]*readDirEntry
	stat    map[string]*statEntry

	// Queue of operations to execute.
	queue chan executable

	// For tracking the number of active goroutines.
	wg sync.WaitGroup
}

func newWorkspaceFilesReadaheadCache(f Filer) *cache {
	c := &cache{
		f: f,

		readDir: make(map[string]*readDirEntry),
		stat:    make(map[string]*statEntry),

		queue: make(chan executable, kMaxQueueSize),
	}

	ctx := context.Background()
	for range kNumCacheWorkers {
		c.wg.Add(1)
		go c.work(ctx)
	}

	return c
}

// work until the queue is closed.
func (c *cache) work(ctx context.Context) {
	defer c.wg.Done()

	for e := range c.queue {
		e.execute(ctx, c)
	}
}

// enqueue adds an operation to the queue.
// If the context is canceled, an error is returned.
// If the queue is full, an error is returned.
//
// Its caller is holding the lock so it cannot block.
func (c *cache) enqueue(ctx context.Context, e executable) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c.queue <- e:
		return nil
	default:
		return queueFullError{e.String()}
	}
}

func (c *cache) completeReadDirForDir(name string, dirEntry fs.DirEntry) {
	// Add to the stat cache if not already present.
	if _, ok := c.stat[name]; !ok {
		e := newStatEntry(name)
		e.markDone(dirEntry.Info())
		c.stat[name] = e
	}

	// Queue a [ReadDir] call for the directory if not already present.
	if _, ok := c.readDir[name]; !ok {
		// Create a new cache entry and queue the operation.
		e := newReadDirEntry(name)
		err := c.enqueue(context.Background(), e)
		if err != nil {
			e.markError(err)
		}

		// Add the entry to the cache, even if has an error.
		c.readDir[name] = e
	}
}

func (c *cache) completeReadDirForFile(name string, dirEntry fs.DirEntry) {
	// Skip if this entry is already in the cache.
	if _, ok := c.stat[name]; ok {
		return
	}

	// Create a new cache entry.
	e := newStatEntry(name)

	// Depending on the object type, we either have to perform a real
	// stat call, or we can use the [fs.DirEntry] info directly.
	switch dirEntry.(wsfsDirEntry).ObjectType {
	case workspace.ObjectTypeNotebook:
		// Note: once the list API returns `repos_export_format` we can avoid this additional stat call.
		// This is the only (?) case where this implementation is tied to the workspace filer.

		// Queue a [Stat] call for the file.
		err := c.enqueue(context.Background(), e)
		if err != nil {
			e.markError(err)
		}
	default:
		// Use the [fs.DirEntry] info directly.
		e.markDone(dirEntry.Info())
	}

	// Add the entry to the cache, even if has an error.
	c.stat[name] = e
}

func (c *cache) completeReadDir(dir string, entries []fs.DirEntry) {
	c.m.Lock()
	defer c.m.Unlock()

	for _, e := range entries {
		name := path.Join(dir, e.Name())

		if e.IsDir() {
			c.completeReadDirForDir(name, e)
		} else {
			c.completeReadDirForFile(name, e)
		}
	}
}

// Cleanup closes the queue and waits for all goroutines to exit.
func (c *cache) Cleanup() {
	close(c.queue)
	c.wg.Wait()
}

// Write passes through to the underlying Filer.
func (c *cache) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	return c.f.Write(ctx, name, reader, mode...)
}

// Read passes through to the underlying Filer.
func (c *cache) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	return c.f.Read(ctx, name)
}

// Delete passes through to the underlying Filer.
func (c *cache) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	return c.f.Delete(ctx, name, mode...)
}

// Mkdir passes through to the underlying Filer.
func (c *cache) Mkdir(ctx context.Context, name string) error {
	return c.f.Mkdir(ctx, name)
}

// ReadDir returns the entries in a directory.
// If the directory is already in the cache, the cached value is returned.
func (c *cache) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	name = path.Clean(name)

	// Lock before R/W access to the cache.
	c.m.Lock()

	// If the directory is already in the cache, wait for and return the cached value.
	if e, ok := c.readDir[name]; ok {
		c.m.Unlock()
		return e.wait(ctx)
	}

	// Otherwise, create a new cache entry and queue the operation.
	e := newReadDirEntry(name)
	err := c.enqueue(ctx, e)
	if err != nil {
		c.m.Unlock()
		return nil, err
	}

	c.readDir[name] = e
	c.m.Unlock()

	// Wait for the operation to complete.
	return e.wait(ctx)
}

// statFromReadDir returns the file info for a file or directory.
// If the file info is already in the cache, the cached value is returned.
func (c *cache) statFromReadDir(ctx context.Context, name string, entry *readDirEntry) (fs.FileInfo, error) {
	_, err := entry.wait(ctx)
	if err != nil {
		return nil, err
	}

	// Upon completion of a [ReadDir] call, all directory entries are added to the stat cache and
	// enqueue a [Stat] call if necessary (entries for notebooks are incomplete and require a
	// real stat call).
	//
	// This means that the file or directory we're trying to stat, either
	//
	//   - is present in the stat cache
	//   - doesn't exist.
	//
	c.m.Lock()
	e, ok := c.stat[name]
	c.m.Unlock()
	if ok {
		return e.wait(ctx)
	}

	return nil, FileDoesNotExistError{name}
}

// Stat returns the file info for a file or directory.
// If the file info is already in the cache, the cached value is returned.
func (c *cache) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	name = path.Clean(name)

	// Lock before R/W access to the cache.
	c.m.Lock()

	// If the file info is already in the cache, wait for and return the cached value.
	if e, ok := c.stat[name]; ok {
		c.m.Unlock()
		return e.wait(ctx)
	}

	// If the parent directory is in the cache (or queued to be read),
	// wait for it to complete to avoid redundant stat calls.
	dir := path.Dir(name)
	if dir != name {
		if e, ok := c.readDir[dir]; ok {
			c.m.Unlock()
			return c.statFromReadDir(ctx, name, e)
		}
	}

	// Otherwise, create a new cache entry and queue the operation.
	e := newStatEntry(name)
	err := c.enqueue(ctx, e)
	if err != nil {
		c.m.Unlock()
		return nil, err
	}

	c.stat[name] = e
	c.m.Unlock()

	// Wait for the operation to complete.
	return e.wait(ctx)
}
