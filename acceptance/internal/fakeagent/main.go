// Command fakeagent stands in for the claude, codex, gemini and copilot CLIs in the aitools acceptance tests. The
// harness builds it once per run and copies it under each agent's name, and a test puts those on PATH, so no test runs
// a CLI from the host.
//
// Run as an agent, it appends its argv to the file named by $FAKE_AGENT_LOG and fails, as a CLI does for a command it
// does not have. No aitools command needs an agent CLI to succeed in these tests, so a test that runs one fails too.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/env"
)

func main() {
	ctx := context.Background()
	program := strings.TrimSuffix(strings.ToLower(filepath.Base(os.Args[0])), ".exe")
	args := os.Args[1:]
	if err := logArgv(ctx, append([]string{program}, args...)); err != nil {
		fail(err)
	}
	fail(fmt.Errorf("fakeagent: %s does not take %q", program, args))
}

// fail reports err on stderr and exits 1, as a CLI does when a command fails.
func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// logArgv appends one invocation to the log: the program on a line of its own, then each argument on a line of its own,
// indented. The arguments are not quoted, so a path or JSON in an argument reads as it was written.
func logArgv(ctx context.Context, argv []string) error {
	f, err := os.OpenFile(env.Get(ctx, "FAKE_AGENT_LOG"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(argv[0] + "\n")
	for _, arg := range argv[1:] {
		b.WriteString("\t" + arg + "\n")
	}
	_, err = f.WriteString(b.String())
	return errors.Join(err, f.Close())
}
