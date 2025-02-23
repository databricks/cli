package sync

import (
	"path"

	"golang.org/x/exp/maps"
)

// List of operations to apply to synchronize local file systems changes to WSFS.
type diff struct {
	delete []string
	rmdir  []string
	mkdir  []string
	put    []string
}

func (d diff) IsEmpty() bool {
	return len(d.put) == 0 && len(d.delete) == 0
}

// Compute operations required to make files in WSFS reflect current local files.
// Takes into account changes since the last sync iteration.
func computeDiff(after, before *SnapshotState) diff {
	d := &diff{
		delete: make([]string, 0),
		rmdir:  make([]string, 0),
		mkdir:  make([]string, 0),
		put:    make([]string, 0),
	}
	d.addRemovedFiles(after, before)
	d.addFilesWithRemoteNameChanged(after, before)
	d.addNewFiles(after, before)
	d.addUpdatedFiles(after, before)
	return *d
}

// Add operators for tracked files that no longer exist.
func (d *diff) addRemovedFiles(after, before *SnapshotState) {
	for localName, remoteName := range before.LocalToRemoteNames {
		if _, ok := after.LocalToRemoteNames[localName]; !ok {
			d.delete = append(d.delete, remoteName)
		}
	}

	// Remove directories that would no longer contain any files.
	beforeDirs := MakeDirSet(maps.Keys(before.LocalToRemoteNames))
	afterDirs := MakeDirSet(maps.Keys(after.LocalToRemoteNames))
	d.rmdir = beforeDirs.Remove(afterDirs).Slice()
}

// Cleanup previous remote files for files that had their remote targets change. For
// example this is possible if you convert a normal python script to a notebook.
func (d *diff) addFilesWithRemoteNameChanged(after, before *SnapshotState) {
	for localName, beforeRemoteName := range before.LocalToRemoteNames {
		afterRemoteName, ok := after.LocalToRemoteNames[localName]
		if ok && afterRemoteName != beforeRemoteName {
			d.delete = append(d.delete, beforeRemoteName)
		}
	}
}

// Add operators for files that were not being tracked before.
func (d *diff) addNewFiles(after, before *SnapshotState) {
	for localName := range after.LastModifiedTimes {
		if _, ok := before.LastModifiedTimes[localName]; !ok {
			d.put = append(d.put, localName)
		}
	}

	// Add directories required for these new files.
	beforeDirs := MakeDirSet(maps.Keys(before.LocalToRemoteNames))
	afterDirs := MakeDirSet(maps.Keys(after.LocalToRemoteNames))
	d.mkdir = afterDirs.Remove(beforeDirs).Slice()
}

// Add operators for files which had their contents updated.
func (d *diff) addUpdatedFiles(after, before *SnapshotState) {
	for localName, modTime := range after.LastModifiedTimes {
		prevModTime, ok := before.LastModifiedTimes[localName]
		if ok && modTime.After(prevModTime) {
			d.put = append(d.put, localName)
		}
	}
}

// groupedMkdir returns a slice of slices of paths to create.
// Because the underlying mkdir calls create intermediate directories,
// we can group them together to reduce the total number of calls.
// This returns a slice of a slice for parity with [groupedRmdir].
func (d diff) groupedMkdir() [][]string {
	// Compute the set of prefixes of all paths to create.
	prefixes := make(map[string]bool)
	for _, name := range d.mkdir {
		dir := path.Dir(name)
		for dir != "." && dir != "/" {
			prefixes[dir] = true
			dir = path.Dir(dir)
		}
	}

	var out []string

	// Collect all paths that are not a prefix of another path.
	for _, name := range d.mkdir {
		if !prefixes[name] {
			out = append(out, name)
		}
	}

	return [][]string{out}
}

// groupedRmdir returns a slice of slices of paths to delete.
// The outer slice is ordered such that each inner slice can be
// deleted in parallel, as long as it is processed in order.
// The first entry will contain leaf directories, the second entry
// will contain intermediate directories, and so on.
func (d diff) groupedRmdir() [][]string {
	// Compute the number of times each directory is a prefix of another directory.
	prefixes := make(map[string]int)
	for _, dir := range d.rmdir {
		prefixes[dir] = 0
	}
	for _, dir := range d.rmdir {
		dir = path.Dir(dir)
		for dir != "." && dir != "/" {
			// Increment the prefix count for this directory, only if it
			// it one of the directories we are deleting.
			if _, ok := prefixes[dir]; ok {
				prefixes[dir]++
			}
			dir = path.Dir(dir)
		}
	}

	var out [][]string

	for len(prefixes) > 0 {
		var toDelete []string

		// Find directories which are not a prefix of another directory.
		// These are the directories we can delete.
		for dir, count := range prefixes {
			if count == 0 {
				toDelete = append(toDelete, dir)
				delete(prefixes, dir)
			}
		}

		// Remove these directories from the prefixes map.
		for _, dir := range toDelete {
			dir = path.Dir(dir)
			for dir != "." && dir != "/" {
				// Decrement the prefix count for this directory, only if it
				// it one of the directories we are deleting.
				if _, ok := prefixes[dir]; ok {
					prefixes[dir]--
				}
				dir = path.Dir(dir)
			}
		}

		// Add these directories to the output.
		out = append(out, toDelete)
	}

	return out
}
