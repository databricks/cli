package sync

import (
	"path"

	"golang.org/x/exp/maps"
)

type diff struct {
	delete []string
	rmdir  []string
	mkdir  []string
	put    []string
}

func (d diff) IsEmpty() bool {
	return len(d.put) == 0 && len(d.delete) == 0
}

// TODO: mark directories that are to be removed.

// TODO: add logging everywhere for debuggility / tracing.
func computeDiff(after *SyncFiles, before *SyncFiles) *diff {
	d := &diff{
		delete: make([]string, 0),
		rmdir:  make([]string, 0),
		mkdir:  make([]string, 0),
		put:    make([]string, 0),
	}
	d.accountForRemovedFiles(after, before)
	d.accountForChangedRemoteNames(after, before)
	d.accountForNewFiles(after, before)
	d.accountForUpdatedFiles(after, before)
	return d
}

func (d *diff) accountForRemovedFiles(after *SyncFiles, before *SyncFiles) {
	for localName, remoteName := range before.LocalToRemoteNames {
		if _, ok := after.LocalToRemoteNames[localName]; !ok {
			d.delete = append(d.delete, remoteName)
		}
	}

	beforeDirs := MakeDirSet(maps.Keys(before.LocalToRemoteNames))
	afterDirs := MakeDirSet(maps.Keys(after.LocalToRemoteNames))
	d.rmdir = append(d.rmdir, beforeDirs.Remove(afterDirs).Slice()...)
}

func (d *diff) accountForChangedRemoteNames(after *SyncFiles, before *SyncFiles) {
	for localName, beforeRemoteName := range before.LocalToRemoteNames {
		afterRemoteName, ok := after.LocalToRemoteNames[localName]
		if !ok || afterRemoteName == beforeRemoteName {
			continue
		}
		d.delete = append(d.delete, beforeRemoteName)
	}
}

func (d *diff) accountForNewFiles(after *SyncFiles, before *SyncFiles) {
	for localName, _ := range after.LastModifiedTimes {
		if _, ok := before.LastModifiedTimes[localName]; !ok {
			d.put = append(d.put, localName)
		}
	}

	beforeDirs := MakeDirSet(maps.Keys(before.LocalToRemoteNames))
	afterDirs := MakeDirSet(maps.Keys(after.LocalToRemoteNames))
	d.mkdir = append(d.mkdir, afterDirs.Remove(beforeDirs).Slice()...)
}

func (d *diff) accountForUpdatedFiles(after *SyncFiles, before *SyncFiles) {
	for localName, modTime := range after.LastModifiedTimes {
		prevModTime, ok := before.LastModifiedTimes[localName]
		if !ok || !modTime.After(prevModTime) {
			continue
		}
		d.put = append(d.put, localName)
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
