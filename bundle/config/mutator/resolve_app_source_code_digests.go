package mutator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"maps"
	pathlib "path"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
)

type resolveAppSourceCodeDigests struct{}

// ResolveAppSourceCodeDigests sets source_code_digest on every app whose source the bundle
// deploys (lifecycle.started: true): a digest of the files file sync uploads to its
// source_code_path. The digest is recorded in the deployment state, so a deploy that
// changes nothing but those files still plans an update and creates a new app deployment.
// It is a content digest because a fresh checkout has no sync snapshot and uploads every
// file, changed or not.
func ResolveAppSourceCodeDigests() bundle.Mutator {
	return &resolveAppSourceCodeDigests{}
}

func (*resolveAppSourceCodeDigests) Name() string {
	return "ResolveAppSourceCodeDigests"
}

func (*resolveAppSourceCodeDigests) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	prefixes := make(map[string]string)
	for name, app := range b.Config.Resources.Apps {
		if prefix, ok := appSourcePrefix(b, app); ok {
			prefixes[name] = prefix
		}
	}
	if len(prefixes) == 0 {
		return nil
	}

	// Listing the sync files walks the tree, so only do it once the loop above found
	// an app that needs a digest.
	syncable, err := listSyncableRelPaths(ctx, b)
	if err != nil {
		return diag.FromErr(fmt.Errorf("list sync files: %w", err))
	}

	var diags diag.Diagnostics
	// Sorted so diagnostics from several apps come out in a stable order.
	for _, name := range slices.Sorted(maps.Keys(prefixes)) {
		digest, err := appSourceCodeDigest(b.SyncRoot, prefixes[name], syncable)
		if err != nil {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("failed to compute the source code digest of app %s: %s", name, err),
				Locations: b.Config.GetLocations("resources.apps." + name + ".source_code_path"),
			})
			continue
		}
		b.Config.Resources.Apps[name].SourceCodeDigest = digest
	}
	return diags
}

// appSourcePrefix returns the path prefix, relative to the sync root, of the files that file
// sync uploads to the app's source_code_path: "app/" for a subdirectory, "" for the sync
// root itself. It reports false for an app whose source the bundle does not deploy from
// local files: lifecycle.started is not true (the bundle creates no app deployment), the
// app uses git_source, or source_code_path is a workspace path outside the bundle's files.
func appSourcePrefix(b *bundle.Bundle, app *resources.App) (string, bool) {
	if app.Lifecycle == nil || app.Lifecycle.Started == nil || !*app.Lifecycle.Started {
		return "", false
	}
	// TranslatePaths has already rewritten a local source_code_path to its workspace path.
	root := pathlib.Clean(b.RemoteFilePath())
	sourceCodePath := pathlib.Clean(app.SourceCodePath)
	if sourceCodePath == root {
		return "", true
	}
	dir, ok := strings.CutPrefix(sourceCodePath, root+"/")
	return dir + "/", ok
}

// appSourceCodeDigest hashes the name and content of every file in syncable that starts
// with prefix. syncable holds sorted paths relative to root. Names are hashed without the
// prefix, so the digest depends only on the content of the source.
func appSourceCodeDigest(root fs.FS, prefix string, syncable []string) (string, error) {
	h := sha256.New()
	for _, rel := range syncable {
		name, ok := strings.CutPrefix(rel, prefix)
		if !ok {
			continue
		}
		hash, err := hashFile(root, rel)
		if err != nil {
			return "", fmt.Errorf("hash %q: %w", rel, err)
		}
		h.Write([]byte(name))
		h.Write([]byte{0})
		h.Write([]byte(hash))
		h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
