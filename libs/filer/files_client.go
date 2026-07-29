package filer

import (
	"cmp"
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"path"
	"slices"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/sdk-go/core/apierr"
	files "github.com/databricks/sdk-go/files/v2"
	"github.com/databricks/sdk-go/options/client"
	"golang.org/x/sync/errgroup"
)

// cloudResponseHeaderTimeout bounds the wait for a cloud storage response header
// on a large-file part transfer. It matches the files/v2 engine's own default.
const cloudResponseHeaderTimeout = 60 * time.Second

// httpStatus returns the HTTP status code of err if it is (or wraps) an
// [apierr.APIError], and -1 otherwise. The filer keys its error mapping off the
// HTTP status rather than the SDK's canonical codes.Code: the Files API reports
// "path already exists" as 409 Conflict, which the SDK maps to codes.Aborted
// (not codes.AlreadyExists), so matching on the code would miss it.
func httpStatus(err error) int {
	if aerr, ok := errors.AsType[*apierr.APIError](err); ok {
		return aerr.HTTPStatusCode()
	}
	return -1
}

// As of 19th Feb 2024, the Files API backend has a rate limit of 10 concurrent
// requests and 100 QPS. We limit the number of concurrent requests to 5 to
// avoid hitting the rate limit.
const maxFilesRequestsInFlight = 5

// Type that implements fs.FileInfo for the Files API.
// This is required for the filer.Stat() method.
type filesApiFileInfo struct {
	absPath      string
	isDir        bool
	fileSize     int64
	lastModified int64
}

func (info filesApiFileInfo) Name() string {
	return path.Base(info.absPath)
}

func (info filesApiFileInfo) Size() int64 {
	return info.fileSize
}

func (info filesApiFileInfo) Mode() fs.FileMode {
	mode := fs.ModePerm
	if info.isDir {
		mode |= fs.ModeDir
	}
	return mode
}

func (info filesApiFileInfo) ModTime() time.Time {
	return time.UnixMilli(info.lastModified)
}

func (info filesApiFileInfo) IsDir() bool {
	return info.isDir
}

func (info filesApiFileInfo) Sys() any {
	return nil
}

// Type that implements fs.DirEntry for the Files API.
// This is required for the filer.ReadDir() method.
type filesApiDirEntry struct {
	i filesApiFileInfo
}

func (e filesApiDirEntry) Name() string {
	return e.i.Name()
}

func (e filesApiDirEntry) IsDir() bool {
	return e.i.IsDir()
}

func (e filesApiDirEntry) Type() fs.FileMode {
	return e.i.Mode()
}

func (e filesApiDirEntry) Info() (fs.FileInfo, error) {
	return e.i, nil
}

// uploadConcurrency bounds the concurrent part uploads of a large-file
// (multipart) write. Parts go to cloud storage rather than the rate-limited
// Files API, so this can fan out wider than the file-level copy parallelism.
const uploadConcurrency = 64

// multipartUploadEnvVar gates whether large Volumes writes are split into parts
// by the files/v2 upload engine. The engine is new, so multipart is off by
// default: when unset or not truthy, Write sends a single-shot PUT (via the
// files/v2 UploadFile endpoint), leaving fs cp and bundle behavior unchanged.
const multipartUploadEnvVar = "DATABRICKS_EXPERIMENTAL_MULTIPART_UPLOAD"

// MultipartUploadEnabled reports whether large files written to UC Volumes are
// split into parts by the files/v2 upload engine, gated by
// DATABRICKS_EXPERIMENTAL_MULTIPART_UPLOAD (off by default). Both paths use the
// files/v2 client; the flag only selects the multipart engine over a single-shot
// PUT.
func MultipartUploadEnabled(ctx context.Context) bool {
	enabled, _ := env.GetBool(ctx, multipartUploadEnvVar)
	return enabled
}

// uploadProgressKey is the context key for an optional large-file upload
// progress callback.
type uploadProgressKey struct{}

// WithUploadProgress returns a context carrying a progress callback for
// large-file (multipart) uploads. FilesClient.Write forwards it to the upload
// engine; it has no effect on writes that do not go through the engine (small
// files, non-seekable streams, non-Volumes targets, or when multipart upload is
// disabled).
func WithUploadProgress(ctx context.Context, fn files.ProgressFunc) context.Context {
	return context.WithValue(ctx, uploadProgressKey{}, fn)
}

func uploadProgressFromContext(ctx context.Context) files.ProgressFunc {
	fn, _ := ctx.Value(uploadProgressKey{}).(files.ProgressFunc)
	return fn
}

// FilesClient implements the [Filer] interface for the Files API backend.
type FilesClient struct {
	client *files.Client

	// File operations will be relative to this path.
	root WorkspaceRootPath

	// Large files are uploaded with the multipart engine. The limiter and transfer
	// client are shared across every Write on this filer, so concurrent uploads
	// (e.g. fs cp -r) draw from one bounded budget and one connection pool.
	limiter        files.Limiter
	transferClient *http.Client
}

func NewFilesClient(ctx context.Context, w *databricks.WorkspaceClient, root string) (Filer, error) {
	c, err := newFilesAPIClient(ctx, w.Config)
	if err != nil {
		return nil, err
	}

	return &FilesClient{
		client: c,

		root: NewWorkspaceRootPath(root),

		limiter:        files.NewLimiter(uploadConcurrency),
		transferClient: newTransferClient(uploadConcurrency),
	}, nil
}

// newTransferClient returns an HTTP client for the cloud-leg part transfers of a
// large-file upload, sized for n concurrent transfers so idle connections are
// reused rather than re-dialed (Go's default of 2 per host would force
// re-dialing). It attaches no Databricks credentials (presigned URLs are
// self-authenticating) and sets no whole-request timeout, which would abort a
// legitimately long transfer; the upload is bounded by its context instead.
// files/v2 builds an equivalent client internally when one is not supplied; this
// filer supplies a shared one so all writes on it draw from a single pool.
func newTransferClient(n int) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = cloudResponseHeaderTimeout
	transport.MaxIdleConnsPerHost = n
	transport.MaxIdleConns = max(transport.MaxIdleConns, n)
	return &http.Client{Transport: transport}
}

// newFilesAPIClient builds the files/v2 client from the CLI's already resolved
// config, reusing its auth instead of re-reading a profile. cfg is passed by
// pointer because config.Config embeds a sync.Mutex and must not be copied.
func newFilesAPIClient(ctx context.Context, cfg *config.Config) (*files.Client, error) {
	copts := []client.Option{
		client.WithHost(cfg.Host),
		client.WithCredentials(configCredentials{cfg: cfg}),
		client.WithoutProfileResolution(),
	}
	// The workspace routing header is needed on unified ("SPOG") hosts; the CLI's
	// "none" sentinel means "no workspace ID", so it is not forwarded.
	if id := cfg.WorkspaceID; id != "" && id != auth.WorkspaceIDNone {
		copts = append(copts, client.WithWorkspaceID(id))
	}
	return files.NewClient(ctx, copts...)
}

func (w *FilesClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	// Check that target path exists if CreateParentDirectories mode is not set
	if !slices.Contains(mode, CreateParentDirectories) {
		dir := path.Dir(absPath)
		_, err := w.client.GetDirectoryMetadata(ctx, &files.GetDirectoryMetadataRequest{DirectoryPath: &dir})
		if err != nil {
			// This API returns a 404 if the directory doesn't exist.
			if httpStatus(err) == http.StatusNotFound {
				return noSuchDirectoryError{dir}
			}
			return err
		}
	}

	overwrite := slices.Contains(mode, OverwriteIfExists)

	// When the multipart flag is enabled, seekable uploads go through the files/v2
	// upload engine, which sends small files in a single PUT and splits large ones
	// into parts. Non-seekable streams and the flag-off case use the single-shot
	// UploadFile endpoint below: the former to avoid buffering the whole stream in
	// memory, the latter to keep the default behavior a plain PUT. The engine
	// recovers an io.ReaderAt for concurrent positioned reads when the source
	// provides one (a local file).
	if MultipartUploadEnabled(ctx) && isSeekable(reader) {
		opts := []files.UploadOption{
			files.WithOverwrite(overwrite),
			files.WithLimiter(w.limiter),
			files.WithTransferClient(w.transferClient),
		}
		// A caller (fs cp) can attach a progress callback via the context to
		// render an upload bar; it is absent for other writers (e.g. bundle).
		if fn := uploadProgressFromContext(ctx); fn != nil {
			opts = append(opts, files.WithProgress(fn))
		}
		_, uerr := w.client.Upload(ctx, absPath, reader, opts...)
		return mapUploadError(uerr, absPath)
	}

	_, err = w.client.UploadFile(ctx, &files.UploadFileRequest{
		FilePath:  &absPath,
		Contents:  io.NopCloser(reader),
		Overwrite: &overwrite,
	})
	if err == nil {
		return nil
	}

	// This API returns 409 if a file already exists at the path.
	if httpStatus(err) == http.StatusConflict {
		return fileAlreadyExistsError{absPath}
	}
	return err
}

// isSeekable reports whether r can be seeked, without moving it: a no-op
// Seek(0, io.SeekCurrent) returns the current offset for a working seeker and
// errors for a broken one. Probing this way (rather than seeking to the end and
// back) guarantees the reader keeps its position, so a false result never leaves
// it parked at EOF for the single-shot fallback, which reads from the current
// offset and would otherwise upload a truncated object. The engine sizes the
// stream itself once it takes over.
func isSeekable(r io.Reader) bool {
	s, ok := r.(io.Seeker)
	if !ok {
		return false
	}
	_, err := s.Seek(0, io.SeekCurrent)
	return err == nil
}

// mapUploadError translates the upload engine's already-exists sentinel into the
// filer's error so skip-if-exists logic (which checks fs.ErrExist) keeps working.
// A nil error passes through unchanged.
func mapUploadError(err error, absPath string) error {
	if errors.Is(err, files.ErrAlreadyExists) {
		return fileAlreadyExistsError{absPath}
	}
	return err
}

func (w *FilesClient) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	resp, err := w.client.DownloadFile(ctx, &files.DownloadFileRequest{FilePath: &absPath})

	// Return early on success.
	if err == nil {
		return resp.Contents, nil
	}

	// This API returns a 404 if the specified path does not exist.
	if httpStatus(err) == http.StatusNotFound {
		// Check if the path is a directory. If so, return not a file error.
		if _, err := w.statDir(ctx, name); err == nil {
			return nil, notAFile{absPath}
		}

		// No file or directory exists at the specified path. Return no such file error.
		return nil, fileDoesNotExistError{absPath}
	}

	return nil, err
}

func (w *FilesClient) deleteFile(ctx context.Context, name string) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	// Illegal to delete the root path.
	if absPath == w.root.rootPath {
		return cannotDeleteRootError{}
	}

	_, err = w.client.DeleteFile(ctx, &files.DeleteFileRequest{FilePath: &absPath})

	// Return early on success.
	if err == nil {
		return nil
	}

	// This files delete API returns a 404 if the specified path does not exist.
	if httpStatus(err) == http.StatusNotFound {
		return fileDoesNotExistError{absPath}
	}

	return err
}

func (w *FilesClient) deleteDirectory(ctx context.Context, name string) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	// Illegal to delete the root path.
	if absPath == w.root.rootPath {
		return cannotDeleteRootError{}
	}

	_, err = w.client.DeleteDirectory(ctx, &files.DeleteDirectoryRequest{DirectoryPath: &absPath})

	// Return early on success.
	if err == nil {
		return nil
	}

	// The directory delete API returns a 400 if the directory is not empty. That
	// status is generic, so confirm the specific reason before mapping it.
	if aerr, ok := errors.AsType[*apierr.APIError](err); ok && aerr.HTTPStatusCode() == http.StatusBadRequest {
		if info := aerr.Details().ErrorInfo; info != nil && info.Reason == "FILES_API_DIRECTORY_IS_NOT_EMPTY" {
			return directoryNotEmptyError{absPath}
		}
		return err
	}

	// On GCS-backed storage a directory created implicitly is just a key prefix
	// with no standalone object, so it disappears when its last child is
	// deleted. The delete API then returns 404 for that already-vanished
	// directory; treat it as a not-found error so recursive delete can consider
	// its goal satisfied.
	if httpStatus(err) == http.StatusNotFound {
		return noSuchDirectoryError{absPath}
	}
	return err
}

func (w *FilesClient) recursiveDelete(ctx context.Context, name string) error {
	filerFS := NewFS(ctx, w)
	var dirsToDelete []string
	var filesToDelete []string
	callback := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Files API does not allowing deleting non-empty directories. We instead
		// collect the directories to delete and delete them once all the files have
		// been deleted.
		if d.IsDir() {
			dirsToDelete = append(dirsToDelete, path)
			return nil
		}

		filesToDelete = append(filesToDelete, path)
		return nil
	}

	// Walk the directory and accumulate the files and directories to delete.
	err := fs.WalkDir(filerFS, name, callback)
	if err != nil {
		return err
	}

	// Delete the files in parallel.
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(maxFilesRequestsInFlight)

	for _, file := range filesToDelete {
		// Skip the file if the context has already been cancelled.
		select {
		case <-groupCtx.Done():
			continue
		default:
			// Proceed.
		}

		group.Go(func() error {
			return w.deleteFile(groupCtx, file)
		})
	}

	// Wait for the files to be deleted and return the first non-nil error.
	err = group.Wait()
	if err != nil {
		return err
	}

	// Delete the directories in reverse order to ensure that the parent
	// directories are deleted after the children. This is possible because
	// fs.WalkDir walks the directories in lexicographical order.
	for _, dir := range slices.Backward(dirsToDelete) {
		err := w.deleteDirectory(ctx, dir)
		// A directory may have already vanished after its last child was
		// deleted (see deleteDirectory for the GCS implicit-directory quirk).
		// The delete's goal is already satisfied, so tolerate it.
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *FilesClient) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	if slices.Contains(mode, DeleteRecursively) {
		return w.recursiveDelete(ctx, name)
	}

	// Issue a stat call to determine if the path is a file or directory.
	info, err := w.Stat(ctx, name)
	if err != nil {
		return err
	}

	// Issue the delete call for a directory
	if info.IsDir() {
		return w.deleteDirectory(ctx, name)
	}

	return w.deleteFile(ctx, name)
}

func (w *FilesClient) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	var entries []fs.DirEntry
	for entry, err := range w.client.ListDirectoryContentsIter(ctx, &files.ListDirectoryContentsRequest{
		DirectoryPath: &absPath,
	}) {
		if err != nil {
			// This API returns a 404 if the specified path does not exist.
			if httpStatus(err) == http.StatusNotFound {
				// Check if the path is a file. If so, return not a directory error.
				if _, ferr := w.statFile(ctx, name); ferr == nil {
					return nil, notADirectory{absPath}
				}

				// No file or directory exists at the specified path. Return no such directory error.
				return nil, noSuchDirectoryError{absPath}
			}
			return nil, err
		}

		entries = append(entries, filesApiDirEntry{
			i: filesApiFileInfo{
				absPath:      value(entry.Path),
				isDir:        value(entry.IsDirectory),
				fileSize:     int64(value(entry.FileSize)),
				lastModified: int64(value(entry.LastModified)),
			},
		})
	}

	// Sort by name for parity with os.ReadDir.
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return cmp.Compare(a.Name(), b.Name()) })
	return entries, nil
}

func (w *FilesClient) Mkdir(ctx context.Context, name string) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	_, err = w.client.CreateDirectory(ctx, &files.CreateDirectoryRequest{DirectoryPath: &absPath})

	// This API returns a 409 when a file already exists at the path (the create
	// is not idempotent over a file).
	if httpStatus(err) == http.StatusConflict {
		return fileAlreadyExistsError{absPath}
	}

	return err
}

// Get file metadata for a file using the Files API.
func (w *FilesClient) statFile(ctx context.Context, name string) (fs.FileInfo, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	resp, err := w.client.GetFileMetadata(ctx, &files.GetFileMetadataRequest{FilePath: &absPath})

	// If the HEAD requests succeeds, the file exists.
	if err == nil {
		return filesApiFileInfo{
			absPath:  absPath,
			isDir:    false,
			fileSize: value(resp.ContentLength),
		}, nil
	}

	// This API returns a 404 if the specified path does not exist.
	if httpStatus(err) == http.StatusNotFound {
		return nil, fileDoesNotExistError{absPath}
	}

	return nil, err
}

// Get file metadata for a directory using the Files API.
func (w *FilesClient) statDir(ctx context.Context, name string) (fs.FileInfo, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	_, err = w.client.GetDirectoryMetadata(ctx, &files.GetDirectoryMetadataRequest{DirectoryPath: &absPath})

	// If the HEAD requests succeeds, the directory exists.
	if err == nil {
		return filesApiFileInfo{absPath: absPath, isDir: true}, nil
	}

	// The directory metadata API returns a 404 if the specified path does not exist.
	if httpStatus(err) == http.StatusNotFound {
		return nil, noSuchDirectoryError{absPath}
	}

	return nil, err
}

func (w *FilesClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	// Assume that the path is a directory and issue a stat call.
	dirInfo, err := w.statDir(ctx, name)

	// If the file exists, return early.
	if err == nil {
		return dirInfo, nil
	}

	// Return early if the error is not a noSuchDirectoryError.
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	// Since the path is not a directory, assume that it is a file and issue a stat call.
	return w.statFile(ctx, name)
}

// value returns *p, or the zero value of T when p is nil.
func value[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}
