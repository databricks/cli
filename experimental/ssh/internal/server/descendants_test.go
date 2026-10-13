package server

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeProc writes a /proc-like tree. Each process is described by its ppid, pgrp and state.
type fakeProcess struct {
	comm  string
	ppid  int
	pgrp  int
	state string
}

func fakeProc(t *testing.T, processes map[int]fakeProcess) string {
	t.Helper()
	root := t.TempDir()
	for pid, p := range processes {
		dir := filepath.Join(root, strconv.Itoa(pid))
		require.NoError(t, os.MkdirAll(dir, 0o755))
		// The real format has 50+ fields; only the four leading ones are read.
		line := strconv.Itoa(pid) + " (" + p.comm + ") " + p.state + " " +
			strconv.Itoa(p.ppid) + " " + strconv.Itoa(p.pgrp) + " 0 0 -1 4194304\n"
		require.NoError(t, os.WriteFile(filepath.Join(dir, "stat"), []byte(line), 0o644))
	}
	return root
}

func TestParseProcStat(t *testing.T) {
	t.Run("parses state, ppid and pgrp", func(t *testing.T) {
		got, err := parseProcStat("4242 (databricks) S 4200 4242 4242 0 -1 4194304 1 0")
		require.NoError(t, err)
		assert.Equal(t, procStat{state: "S", ppid: 4200, pgrp: 4242}, got)
	})

	// The comm field is unquoted and may contain spaces and parentheses, so the fields
	// after it can only be located from the last ')'.
	t.Run("handles a comm with spaces and parentheses", func(t *testing.T) {
		got, err := parseProcStat("7 (weird ) name) R 3 9 9 0 -1 0")
		require.NoError(t, err)
		assert.Equal(t, procStat{state: "R", ppid: 3, pgrp: 9}, got)
	})

	t.Run("rejects a line without comm", func(t *testing.T) {
		_, err := parseProcStat("4242 S 4200 4242")
		assert.ErrorContains(t, err, "no comm field")
	})

	t.Run("rejects a truncated line", func(t *testing.T) {
		_, err := parseProcStat("4242 (databricks) S 4200")
		assert.ErrorContains(t, err, "expected state, ppid and pgrp")
	})
}

func TestDetachedDescendants(t *testing.T) {
	// The shape the server sees at teardown: the notebook (100) is the subreaper, the
	// server (200) leads its own group, sshd (300) is in the server's group, and the
	// detached work (400, 500) has been reparented onto the notebook with its own groups.
	const notebook, server, sshd = 100, 200, 300

	t.Run("returns detached work only", func(t *testing.T) {
		root := fakeProc(t, map[int]fakeProcess{
			notebook: {comm: "python", ppid: 1, pgrp: notebook, state: "S"},
			server:   {comm: "databricks", ppid: notebook, pgrp: server, state: "S"},
			sshd:     {comm: "sshd", ppid: server, pgrp: server, state: "S"},
			400:      {comm: "tmux: server", ppid: notebook, pgrp: 400, state: "S"},
			500:      {comm: "train.py", ppid: notebook, pgrp: 500, state: "R"},
		})

		pids, err := detachedDescendants(root, server)
		require.NoError(t, err)
		assert.Equal(t, []int{400, 500}, pids)
	})

	t.Run("excludes the server's own process group", func(t *testing.T) {
		// A login shell that sshd put in its own group, but that is still parented by
		// sshd rather than adopted by the notebook, is not detached work.
		root := fakeProc(t, map[int]fakeProcess{
			notebook: {comm: "python", ppid: 1, pgrp: notebook, state: "S"},
			server:   {comm: "databricks", ppid: notebook, pgrp: server, state: "S"},
			sshd:     {comm: "sshd", ppid: server, pgrp: server, state: "S"},
			400:      {comm: "bash", ppid: sshd, pgrp: 400, state: "S"},
		})

		pids, err := detachedDescendants(root, server)
		require.NoError(t, err)
		assert.Empty(t, pids)
	})

	t.Run("skips zombies", func(t *testing.T) {
		root := fakeProc(t, map[int]fakeProcess{
			notebook: {comm: "python", ppid: 1, pgrp: notebook, state: "S"},
			server:   {comm: "databricks", ppid: notebook, pgrp: server, state: "S"},
			400:      {comm: "gone", ppid: notebook, pgrp: 400, state: "Z"},
		})

		pids, err := detachedDescendants(root, server)
		require.NoError(t, err)
		assert.Empty(t, pids)
	})

	// The notebook died first, so the server was reparented to PID 1. Nothing is anchored
	// any more and there is no sibling set to enumerate.
	t.Run("returns nothing once the notebook is gone", func(t *testing.T) {
		root := fakeProc(t, map[int]fakeProcess{
			1:      {comm: "systemd", ppid: 0, pgrp: 1, state: "S"},
			server: {comm: "databricks", ppid: 1, pgrp: server, state: "S"},
			400:    {comm: "tmux: server", ppid: 1, pgrp: 400, state: "S"},
		})

		pids, err := detachedDescendants(root, server)
		require.NoError(t, err)
		assert.Empty(t, pids)
	})

	t.Run("ignores unreadable and non-process entries", func(t *testing.T) {
		root := fakeProc(t, map[int]fakeProcess{
			notebook: {comm: "python", ppid: 1, pgrp: notebook, state: "S"},
			server:   {comm: "databricks", ppid: notebook, pgrp: server, state: "S"},
			400:      {comm: "tmux: server", ppid: notebook, pgrp: 400, state: "S"},
		})
		require.NoError(t, os.MkdirAll(filepath.Join(root, "self"), 0o755))
		// A process that exited between the readdir and the stat read.
		require.NoError(t, os.MkdirAll(filepath.Join(root, "999"), 0o755))

		pids, err := detachedDescendants(root, server)
		require.NoError(t, err)
		assert.Equal(t, []int{400}, pids)
	})

	t.Run("fails when own stat is missing", func(t *testing.T) {
		_, err := detachedDescendants(t.TempDir(), server)
		assert.ErrorContains(t, err, "failed to read own process stat")
	})
}

// The fixtures above are hand-written, so this pins the field offsets against a real
// /proc/<pid>/stat. The server only ever runs on Linux compute.
func TestParseProcStatAgainstRealProcfs(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("procfs is Linux-only")
	}

	got, err := readProcStat(procRoot, os.Getpid())
	require.NoError(t, err)
	assert.Equal(t, os.Getppid(), got.ppid)
	assert.NotZero(t, got.pgrp)
	assert.NotEmpty(t, got.state)
}

func TestFormatPids(t *testing.T) {
	assert.Equal(t, "1, 22, 333", formatPids([]int{1, 22, 333}))
	assert.Empty(t, formatPids(nil))
}
