package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"path"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/log"
)

// shaLister fetches content SHAs for a workspace directory in one bulk call.
// The interface lets tests stub the API call without spinning up a fake
// filer.
type shaLister interface {
	ListWithSHAs(ctx context.Context, dirPath string) ([]filer.RemoteFileMetadata, error)
}

// RemoteFilter is the third layer of the sync pipeline. It takes the action
// plan produced by the snapshot diff and drops puts whose remote SHA already
// matches the local SHA — files the workspace already has, byte-for-byte.
//
// The expensive work (one bulk list, one SHA per skipped put candidate) only
// pays off when the snapshot diff has produced false-positive puts at scale.
// The caller decides when to invoke Apply; today that's only on a fresh
// snapshot (no prior local state).
type RemoteFilter struct {
	lister     shaLister
	remotePath string
}

func NewRemoteFilter(lister shaLister, remotePath string) *RemoteFilter {
	return &RemoteFilter{lister: lister, remotePath: remotePath}
}

// Apply returns a copy of d with put entries removed for files whose local
// SHA already matches the remote SHA. Errors fetching or computing SHAs are
// logged and treated as "do not skip" — the worst case is an unnecessary
// upload, which is the existing behavior.
func (f *RemoteFilter) Apply(ctx context.Context, d diff, files []fileset.File, localToRemote map[string]string) diff {
	if len(d.put) == 0 || f == nil || f.lister == nil {
		return d
	}

	remote, err := f.lister.ListWithSHAs(ctx, f.remotePath)
	if err != nil {
		log.Warnf(ctx, "could not fetch remote content SHAs from %s; uploading all candidate files: %s", f.remotePath, err)
		return d
	}
	if len(remote) == 0 {
		return d
	}

	remoteSHAByPath := make(map[string]string, len(remote))
	for _, e := range remote {
		if e.ContentSHA256Hex == "" {
			continue
		}
		remoteSHAByPath[e.Path] = e.ContentSHA256Hex
	}

	localByRelative := make(map[string]*fileset.File, len(files))
	for i := range files {
		localByRelative[files[i].Relative] = &files[i]
	}

	keep := make([]string, 0, len(d.put))
	skipped := 0
	for _, p := range d.put {
		if !f.canSkip(ctx, p, localByRelative, localToRemote, remoteSHAByPath) {
			keep = append(keep, p)
			continue
		}
		skipped++
	}

	if skipped > 0 {
		log.Debugf(ctx, "remote-filter: skipped %d/%d uploads matching workspace SHAs", skipped, len(d.put))
	}

	return diff{
		delete: d.delete,
		rmdir:  d.rmdir,
		mkdir:  d.mkdir,
		put:    keep,
	}
}

// canSkip reports whether the put for relativePath can be safely dropped:
// the workspace already has a file at the corresponding remote path with the
// same SHA-256 as the local file.
func (f *RemoteFilter) canSkip(
	ctx context.Context,
	relativePath string,
	localByRelative map[string]*fileset.File,
	localToRemote map[string]string,
	remoteSHAByPath map[string]string,
) bool {
	local, ok := localByRelative[relativePath]
	if !ok {
		return false
	}
	remoteName, ok := localToRemote[relativePath]
	if !ok {
		return false
	}
	remoteSHA, ok := remoteSHAByPath[path.Join(f.remotePath, remoteName)]
	if !ok {
		return false
	}
	localSHA, err := computeFileSHA(local)
	if err != nil {
		log.Debugf(ctx, "remote-filter: hashing %s failed; will upload: %s", relativePath, err)
		return false
	}
	return localSHA == remoteSHA
}

func computeFileSHA(f *fileset.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()
	h := sha256.New()
	if _, err := io.Copy(h, rc); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
