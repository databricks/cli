package aircmd

// Garbage collection for v2 snapshot objects (design section 4). It is off unless
// DATABRICKS_AIR_SNAPSHOT_GC is set, and it only ever touches the v2 layout under the snapshot
// directory: manifests named *.air.json and objects/<32 hex>.tar.gz.
//
// Safety rests on two windows. An object is reused only while younger than v2ReuseWindow (R).
// It is deleted only when older than v2GCAge (A) and not referenced by a manifest younger than
// v2ManifestLease (L). Because A - R is a day, an object reused at time t cannot be deleted
// before t + (A - R). The submit that reused it writes its manifest within that day.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
)

const (
	// snapshotGCEnv opts in to GC after a submit. It is unset by default.
	snapshotGCEnv = "DATABRICKS_AIR_SNAPSHOT_GC"

	v2ReuseWindow   = 7 * 24 * time.Hour
	v2GCAge         = 8 * 24 * time.Hour
	v2ManifestLease = 30 * 24 * time.Hour
	v2GCInterval    = 24 * time.Hour
	v2GCTimeBox     = 3 * time.Second
	v2GCMarkerName  = ".air-gc.marker"
)

// v2ObjectFile matches the object files GC may delete, relative to objects/.
var v2ObjectFile = regexp.MustCompile(`^[0-9a-f]{32}\.tar\.gz$`)

// v2GCPlan is what a GC run would delete. Names are relative to the snapshot directory.
type v2GCPlan struct {
	Manifests []string
	Objects   []string
	// ObjectsKept counts objects that are referenced or too young to delete.
	ObjectsKept int
}

// planV2GC lists the v2 layout in store and decides what a GC run at now would delete. It
// deletes nothing. Any listing or manifest that cannot be read fails the plan, since an
// unread manifest could reference any object.
func planV2GC(ctx context.Context, store filer.Filer, now time.Time) (v2GCPlan, error) {
	var plan v2GCPlan
	entries, err := store.ReadDir(ctx, ".")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return plan, err
	}
	referenced := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), v2ManifestSuffix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			return v2GCPlan{}, err
		}
		if now.Sub(info.ModTime()) >= v2ManifestLease {
			plan.Manifests = append(plan.Manifests, e.Name())
			continue
		}
		ids, err := readV2ManifestIDs(ctx, store, e.Name())
		if err != nil {
			return v2GCPlan{}, err
		}
		for _, id := range ids {
			referenced[id] = true
		}
	}

	objects, err := store.ReadDir(ctx, v2ObjectsDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return v2GCPlan{}, err
	}
	for _, o := range objects {
		if o.IsDir() || !v2ObjectFile.MatchString(o.Name()) {
			continue
		}
		info, err := o.Info()
		if err != nil {
			return v2GCPlan{}, err
		}
		age := now.Sub(info.ModTime())
		id := strings.TrimSuffix(o.Name(), ".tar.gz")
		if referenced[id] || age < v2GCAge {
			plan.ObjectsKept++
			continue
		}
		plan.Objects = append(plan.Objects, path.Join(v2ObjectsDir, o.Name()))
	}
	return plan, nil
}

// readV2ManifestIDs returns the member ids of the manifest at name.
func readV2ManifestIDs(ctx context.Context, store filer.Filer, name string) ([]string, error) {
	rc, err := store.Read(ctx, name)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var manifest v2Manifest
	if err := json.NewDecoder(rc).Decode(&manifest); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(manifest.Members))
	for _, m := range manifest.Members {
		ids = append(ids, m.ID)
	}
	return ids, nil
}

// applyV2GC deletes the planned names. Each object is checked again just before its delete, and
// it is kept if a submit has rewritten it since the plan. A failed delete is returned after the
// rest are attempted; partial progress is safe because each delete is independently justified.
func applyV2GC(ctx context.Context, store filer.Filer, plan v2GCPlan, now time.Time) error {
	var firstErr error
	for _, name := range plan.Manifests {
		// Recheck the lease: a submit may have renewed the manifest since the plan.
		info, err := store.Stat(ctx, name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			firstErr = errOr(firstErr, err)
			continue
		}
		if now.Sub(info.ModTime()) < v2ManifestLease {
			continue
		}
		if err := store.Delete(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			firstErr = errOr(firstErr, err)
		}
	}
	for _, name := range plan.Objects {
		info, err := store.Stat(ctx, name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			firstErr = errOr(firstErr, err)
			continue
		}
		if now.Sub(info.ModTime()) < v2GCAge {
			continue
		}
		if err := store.Delete(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			firstErr = errOr(firstErr, err)
		}
	}
	return firstErr
}

// errOr returns first if set, else err.
func errOr(first, err error) error {
	if first != nil {
		return first
	}
	return err
}

// gcEnabled reports whether GC is opted in. Unset, or any value env.GetBool treats as false, is off.
func gcEnabled(ctx context.Context) bool {
	on, _ := env.GetBool(ctx, snapshotGCEnv)
	return on
}

// gcDue reports whether the last GC run is at least v2GCInterval old, by the marker's mtime.
func gcDue(ctx context.Context, store filer.Filer, now time.Time) (bool, error) {
	info, err := store.Stat(ctx, v2GCMarkerName)
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return now.Sub(info.ModTime()) >= v2GCInterval, nil
}

// runV2GC is the best-effort GC run after a submit: it runs at most once per v2GCInterval per
// store, and all its errors are logged at debug. It never fails the submit. The caller bounds
// ctx with v2GCTimeBox; if the box expires while planning, nothing is deleted.
func runV2GC(ctx context.Context, store filer.Filer, now time.Time) {
	due, err := gcDue(ctx, store, now)
	if err != nil {
		log.Debugf(ctx, "snapshot GC skipped: %v", err)
		return
	}
	if !due {
		return
	}
	if err := store.Write(ctx, v2GCMarkerName, bytes.NewReader(nil), filer.OverwriteIfExists, filer.CreateParentDirectories); err != nil {
		log.Debugf(ctx, "snapshot GC skipped: %v", err)
		return
	}
	plan, err := planV2GC(ctx, store, now)
	if err != nil {
		log.Debugf(ctx, "snapshot GC planning failed; nothing deleted: %v", err)
		return
	}
	if err := applyV2GC(ctx, store, plan, now); err != nil {
		log.Debugf(ctx, "snapshot GC incomplete: %v", err)
	}
	log.Debugf(ctx, "snapshot GC deleted %d manifests and %d objects, kept %d objects", len(plan.Manifests), len(plan.Objects), plan.ObjectsKept)
}

// gcAfterSubmit runs GC for the snapshot directory of a submitted run, when GC is opted in. It
// runs after the run exists, and is bounded by v2GCTimeBox.
func gcAfterSubmit(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig, base string) {
	if !gcEnabled(ctx) || cfg.CodeSource == nil || cfg.CodeSource.Snapshot == nil {
		return
	}
	artifactPath := path.Join(base, ".air", "repo_snapshots")
	if cfg.CodeSource.Snapshot.RemoteVolume != nil {
		artifactPath = *cfg.CodeSource.Snapshot.RemoteVolume
	}
	store, _, err := snapshotUploadFiler(ctx, w, artifactPath)
	if err != nil {
		log.Debugf(ctx, "snapshot GC skipped: %v", err)
		return
	}
	gcCtx, cancel := context.WithTimeout(ctx, v2GCTimeBox)
	defer cancel()
	runV2GC(gcCtx, store, time.Now())
}
