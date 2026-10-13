package server

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// procRoot is the procfs mount point. The functions below take the root as an argument
// so tests can run against a fixture tree instead of the live one.
const procRoot = "/proc"

// procStat is the part of /proc/<pid>/stat this package reads.
type procStat struct {
	state string
	ppid  int
	pgrp  int
}

// parseProcStat reads state, ppid and pgrp out of the contents of /proc/<pid>/stat.
func parseProcStat(content string) (procStat, error) {
	// Anchor on the last ')' instead of splitting from the left: the second field is the
	// executable name, and it can contain both spaces and parentheses.
	comm := strings.LastIndex(content, ")")
	if comm < 0 {
		return procStat{}, fmt.Errorf("no comm field in %q", content)
	}
	// state, ppid and pgrp are the three fields that follow comm.
	fields := strings.Fields(content[comm+1:])
	if len(fields) < 3 {
		return procStat{}, fmt.Errorf("expected state, ppid and pgrp after comm in %q", content)
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return procStat{}, fmt.Errorf("failed to parse ppid in %q: %w", content, err)
	}
	pgrp, err := strconv.Atoi(fields[2])
	if err != nil {
		return procStat{}, fmt.Errorf("failed to parse pgrp in %q: %w", content, err)
	}
	return procStat{state: fields[0], ppid: ppid, pgrp: pgrp}, nil
}

func readProcStat(root string, pid int) (procStat, error) {
	content, err := os.ReadFile(filepath.Join(root, strconv.Itoa(pid), "stat"))
	if err != nil {
		return procStat{}, err
	}
	return parseProcStat(string(content))
}

// detachedDescendants returns the pids of processes the SSH session started that left
// the server's process group - what tmux, setsid, disown and a plain background command
// all do.
//
// They surface as siblings of the server: the bootstrap notebook sets
// PR_SET_CHILD_SUBREAPER, so a process that orphans itself is reparented to the notebook
// rather than to PID 1. Taking the notebook's children and excluding the server's own
// process group therefore leaves exactly the detached work - the server's sshd children
// stay in its group, and the notebook starts nothing else.
func detachedDescendants(root string, selfPid int) ([]int, error) {
	self, err := readProcStat(root, selfPid)
	if err != nil {
		return nil, fmt.Errorf("failed to read own process stat: %w", err)
	}
	// The notebook is gone and the server has been reparented to PID 1: there is no
	// anchor left to enumerate against, and nothing left to keep alive.
	if self.ppid <= 1 {
		return nil, nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", root, err)
	}

	var pids []int
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			// Not a process directory.
			continue
		}
		stat, err := readProcStat(root, pid)
		if err != nil {
			// The process exited while we were walking, or its stat is unreadable.
			continue
		}
		if stat.ppid != self.ppid || stat.pgrp == self.pgrp {
			continue
		}
		// Zombies are already dead; the notebook's SIGCHLD handler collects them.
		if strings.HasPrefix(stat.state, "Z") {
			continue
		}
		pids = append(pids, pid)
	}
	slices.Sort(pids)
	return pids, nil
}

// formatPids renders pids for a log line, e.g. "1234, 1235".
func formatPids(pids []int) string {
	parts := make([]string, len(pids))
	for i, pid := range pids {
		parts[i] = strconv.Itoa(pid)
	}
	return strings.Join(parts, ", ")
}
