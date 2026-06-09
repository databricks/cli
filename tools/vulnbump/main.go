// Command vulnbump upgrades Go module dependencies that govulncheck reports as
// vulnerable to their fixed versions and prints a Markdown summary of the
// upgrades for use in a pull request body.
//
// It reads the output of `govulncheck -scan module -format json` from stdin and
// applies the upgrades to the module in the directory given as the first
// argument (default "."). End to end it is driven as:
//
//	govulncheck -scan module -format json | vulnbump .
//
// .github/workflows/bump-vuln-deps.yml runs it this way across every go.mod.
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// commandRunner runs a command in dir. It is injected so tests can exercise the
// end-to-end flow without shelling out to the Go toolchain or the network.
type commandRunner func(dir, name string, args ...string) error

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if err := run(dir, os.Stdin, os.Stdout, execCommand); err != nil {
		fmt.Fprintln(os.Stderr, "vulnbump:", err)
		os.Exit(1)
	}
}

// run reads govulncheck JSON from in, applies each bump in dir via runCmd, and
// writes the Markdown summary to out.
func run(dir string, in io.Reader, out io.Writer, runCmd commandRunner) error {
	bumps, err := parseBumps(in)
	if err != nil {
		return err
	}
	for _, b := range bumps {
		if err := runCmd(dir, "go", "get", b.Module+"@"+b.FixedVersion); err != nil {
			return err
		}
	}
	if len(bumps) > 0 {
		if err := runCmd(dir, "go", "mod", "tidy"); err != nil {
			return err
		}
	}
	_, err = io.WriteString(out, renderSummary(bumps))
	return err
}

// execCommand runs a command in dir, forwarding its output to stderr so it
// surfaces in the workflow log without polluting the Markdown summary on stdout.
func execCommand(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v in %s: %w", name, args, dir, err)
	}
	return nil
}
