package sync

import (
	"sort"
	"strings"

	"golang.org/x/exp/maps"
)

type diff struct {
	put    []string
	delete []string
}

func (d diff) IsEmpty() bool {
	return len(d.put) == 0 && len(d.delete) == 0
}

func (d diff) GroupDeletesByNestingLevel() [][]string {
	// Group the paths to delete by their nesting level.
	// We need a directory to be empty before we can remove it, so a file at
	// level 5 must be deleted before deleting its directory at level 4.
	deletesByLevel := make(map[int][]string)
	for _, remoteName := range d.delete {
		level := len(strings.Split(remoteName, "/"))
		deletesByLevel[level] = append(deletesByLevel[level], remoteName)
	}

	// Get a sorted list of nesting levels.
	levels := maps.Keys(deletesByLevel)
	sort.Ints(levels)

	// Return slice ordered by descending level.
	// Each slice contains paths at the same level.
	var out [][]string
	for i := len(levels) - 1; i >= 0; i-- {
		out = append(out, deletesByLevel[levels[i]])
	}

	return out
}
