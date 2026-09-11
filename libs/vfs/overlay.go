package vfs

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"path"
	"time"
)

// Overlay returns a Path that behaves like base but also serves a set of in-memory
// files, letting a caller inject generated content into a tree (e.g. the bundle sync
// root) without writing to disk. Overlaid files participate in Open/Stat/ReadDir/
// ReadFile and fs.WalkDir; the real tree is untouched.
//
// Names are slash-separated and relative to the root, per fs.ValidPath: an absolute
// name, an empty name, or one escaping the root with ".." is rejected. A name must
// not collide with a real entry in base; on collision the overlaid file wins for
// direct access.
func Overlay(base Path, files map[string][]byte) (Path, error) {
	// Copy so later mutations of the caller's map don't leak in.
	overlay := make(map[string][]byte, len(files))
	// dirs maps each ancestor directory to its overlaid children (base-name -> isFile),
	// so ReadDir and fs.WalkDir surface the synthetic entries even when the parent
	// directory itself exists only in the overlay.
	dirs := make(map[string]map[string]bool)

	for name, data := range files {
		clean := path.Clean(name)
		// Reject anything that isn't a file rooted in the tree. An absolute path would
		// also make the ancestor walk below spin forever: path.Dir("/") is "/", never
		// ".". Clean("") is "." (the root itself), which is a directory, not a file.
		if clean == "." || !fs.ValidPath(clean) {
			return nil, fmt.Errorf("overlay: invalid file name %q: must be a relative slash-separated path inside the root", name)
		}
		overlay[clean] = data
		// Register clean under its parent as a file, then each ancestor dir under its
		// own parent as a directory, up to ".".
		child := clean
		isFile := true
		for child != "." {
			parent := path.Dir(child)
			addOverlayChild(dirs, parent, path.Base(child), isFile)
			child = parent
			isFile = false
		}
	}
	return &overlayPath{base: base, files: overlay, dirs: dirs}, nil
}

// addOverlayChild records child under dir in the ancestor-directory index.
func addOverlayChild(dirs map[string]map[string]bool, dir, child string, isFile bool) {
	entries, ok := dirs[dir]
	if !ok {
		entries = make(map[string]bool)
		dirs[dir] = entries
	}
	// Don't downgrade a dir entry to file if seen both ways; files never collide.
	if isFile || !entries[child] {
		entries[child] = isFile
	}
}

type overlayPath struct {
	base  Path
	files map[string][]byte
	// dirs maps a directory name to its overlaid child base-names (value true = the
	// child is an overlaid file, false = an overlaid subdirectory).
	dirs map[string]map[string]bool
}

func (o *overlayPath) Open(name string) (fs.File, error) {
	if data, ok := o.files[path.Clean(name)]; ok {
		return newMemFile(path.Base(name), data), nil
	}
	return o.base.Open(name)
}

func (o *overlayPath) Stat(name string) (fs.FileInfo, error) {
	if data, ok := o.files[path.Clean(name)]; ok {
		return memFileInfo{name: path.Base(name), size: int64(len(data))}, nil
	}
	return o.base.Stat(name)
}

func (o *overlayPath) ReadFile(name string) ([]byte, error) {
	if data, ok := o.files[path.Clean(name)]; ok {
		return append([]byte(nil), data...), nil
	}
	return o.base.ReadFile(name)
}

func (o *overlayPath) ReadDir(name string) ([]fs.DirEntry, error) {
	clean := path.Clean(name)
	baseEntries, err := o.base.ReadDir(name)
	// A dir that exists only in the overlay (e.g. .air_snapshots) has no real
	// counterpart; tolerate a not-exist error when we have overlaid children for it.
	overlaid := o.dirs[clean]
	if err != nil && overlaid == nil {
		return nil, err
	}

	seen := make(map[string]bool, len(baseEntries))
	entries := make([]fs.DirEntry, 0, len(baseEntries)+len(overlaid))
	for _, e := range baseEntries {
		seen[e.Name()] = true
		entries = append(entries, e)
	}
	for child, isFile := range overlaid {
		if seen[child] {
			continue
		}
		if isFile {
			full := child
			if clean != "." {
				full = clean + "/" + child
			}
			entries = append(entries, memDirEntry{name: child, size: int64(len(o.files[full]))})
		} else {
			entries = append(entries, memDirEntry{name: child, dir: true})
		}
	}
	return entries, nil
}

func (o *overlayPath) Parent() Path   { return o.base.Parent() }
func (o *overlayPath) Native() string { return o.base.Native() }

// memFile is an in-memory fs.File for an overlaid file.
type memFile struct {
	info   memFileInfo
	reader *bytes.Reader
}

func newMemFile(name string, data []byte) *memFile {
	return &memFile{
		info:   memFileInfo{name: name, size: int64(len(data))},
		reader: bytes.NewReader(data),
	}
}

func (f *memFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *memFile) Read(p []byte) (int, error) { return f.reader.Read(p) }
func (f *memFile) Close() error               { return nil }

type memFileInfo struct {
	name string
	size int64
}

func (i memFileInfo) Name() string       { return i.name }
func (i memFileInfo) Size() int64        { return i.size }
func (i memFileInfo) Mode() fs.FileMode  { return 0o644 }
func (i memFileInfo) ModTime() time.Time { return time.Time{} }
func (i memFileInfo) IsDir() bool        { return false }
func (i memFileInfo) Sys() any           { return nil }

type memDirEntry struct {
	name string
	dir  bool
	size int64
}

func (e memDirEntry) Name() string { return e.name }
func (e memDirEntry) IsDir() bool  { return e.dir }
func (e memDirEntry) Type() fs.FileMode {
	if e.dir {
		return fs.ModeDir
	}
	return 0
}

func (e memDirEntry) Info() (fs.FileInfo, error) {
	return memFileInfo{name: e.name, size: e.size}, nil
}

var _ io.Reader = (*memFile)(nil)
