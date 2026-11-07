package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"
)

const refreshIntentActor = "DATABRICKS_TEST_REFRESH_INTENT_ACTOR"

type refreshIntentSetup struct {
	Path        string
	Nonrotating bool
}
type refreshIntentMessage struct {
	Kind       string
	Generation int
}

func TestRefreshIntentProcessLoss(t *testing.T) {
	if os.Getenv(refreshIntentActor) == "1" {
		var setup refreshIntentSetup
		if err := json.NewDecoder(os.Stdin).Decode(&setup); err != nil {
			t.Fatal("closed actor setup failure")
		}
		store, err := NewFileStore(t.Context(), WithFileLocation(setup.Path))
		if err != nil {
			t.Fatal("closed actor store failure")
		}
		c := store.(*fileStore)
		result := intentToken("1")
		generation := 1
		if setup.Nonrotating {
			entry, err := c.Lookup("primary")
			if err != nil {
				t.Fatal("closed actor original entry failure")
			}
			result = entry
			generation = 0
		}
		c.beforeIntentCompletion = func() {
			if err := json.NewEncoder(os.Stdout).Encode(refreshIntentMessage{Kind: "cache-published", Generation: generation}); err != nil {
				os.Exit(3)
			}
			_, _ = io.Copy(io.Discard, os.Stdin)
		}
		calls := 0
		if err := intentRefresh(t, c, "primary", result, &calls); err != nil {
			t.Fatal("closed actor publication failure")
		}
		t.Fatal("actor passed the intentional process-loss boundary")
	}
	for _, nonrotating := range []bool{false, true} {
		t.Run(map[bool]string{false: "rotating", true: "nonrotating-identical-entry"}[nonrotating], func(t *testing.T) {
			c := intentFixture(t)
			original := intentToken("0")
			seedIntent(t, c, "primary", original)
			seedIntent(t, c, "copy", original)
			home := t.TempDir()
			command := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestRefreshIntentProcessLoss$", "-test.timeout=30s")
			command.Env = []string{refreshIntentActor + "=1", "HOME=" + home, "USERPROFILE=" + home, "PATH=" + os.Getenv("PATH"), "TMPDIR=" + os.TempDir(), "TMP=" + os.TempDir(), "TEMP=" + os.TempDir()}
			input, err := command.StdinPipe()
			if err != nil {
				t.Fatal("closed actor input failure")
			}
			output, err := command.StdoutPipe()
			if err != nil {
				t.Fatal("closed actor output failure")
			}
			command.Stderr = &bytes.Buffer{}
			if err := command.Start(); err != nil {
				t.Fatal("closed actor start failure")
			}
			waited := false
			t.Cleanup(func() {
				_ = input.Close()
				if !waited {
					_ = command.Process.Kill()
					_ = command.Wait()
				}
			})
			if err := json.NewEncoder(input).Encode(refreshIntentSetup{Path: c.fileLocation, Nonrotating: nonrotating}); err != nil {
				t.Fatal("closed actor send failure")
			}
			var message refreshIntentMessage
			if err := json.NewDecoder(output).Decode(&message); err != nil || message.Kind != "cache-published" {
				t.Fatal("closed actor did not reach exact publication boundary")
			}
			expected := 1
			if nonrotating {
				expected = 0
			}
			if message.Generation != expected {
				t.Fatal("closed actor generation mismatch")
			}
			if err := command.Process.Kill(); err != nil {
				t.Fatal("closed actor process loss failed")
			}
			err = command.Wait()
			waited = true
			if _, ok := errors.AsType[*exec.ExitError](err); !ok {
				t.Fatal("actor did not terminate by process loss")
			}
			store, err := NewFileStore(t.Context(), WithFileLocation(c.fileLocation))
			if err != nil {
				t.Fatal("fresh actor store failed")
			}
			fresh := store.(*fileStore)
			_, err = WithFileTransaction(t.Context(), fresh, func(view Store) error { return nil }, nil)
			if err != nil {
				t.Fatal("actual committed cache proof failed reconciliation")
			}
			j := readIntent(t, fresh)
			completed := 0
			for _, a := range j.Attempts {
				if a.Phase == "completed" {
					completed++
				}
			}
			if completed != 1 {
				t.Fatal("response proof was not completed exactly once")
			}
			calls := 0
			if !errors.Is(intentRefresh(t, fresh, "copy", intentToken("2"), &calls), ErrRefreshRecoveryRequired) || calls != 0 {
				t.Fatal("stale alias replayed after process loss")
			}
			entry, err := fresh.Lookup("primary")
			if err != nil {
				t.Fatal("fresh completed entry missing")
			}
			if err := intentRefresh(t, fresh, "primary", entry, &calls); err != nil || calls != 1 {
				t.Fatal("proven current epoch could not refresh")
			}
		})
	}
}
