package cmd_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// fuzzStubTransport short-circuits SDK HTTP requests without going through a
// real listener. The SDK treats 4xx as terminal (no retries), so LROs and
// paginated lists fail fast instead of looping.
type fuzzStubTransport struct{}

func (fuzzStubTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	body := `{"error_code":"FUZZ_STUB","message":"fuzz"}`
	return &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}, nil
}

// perInvocationTimeout bounds a single Execute call as a backstop against
// hangs. Some auto-generated commands poll long-running operations (e.g.
// apps create-space) that don't terminate even when fuzzStubTransport
// returns 4xx.
const perInvocationTimeout = 200 * time.Millisecond

// fuzzHarness holds the root command for a fuzz test plus a base context
// applied to each leaf before invocation. The base context carries the parts
// of CLI setup that auto-generated RunE bodies reach for: fake workspace and
// account clients, an initialized logdiag, and a discarding cmdio. The full
// cli tree is built so leaves inherit persistent flags (notably --output)
// from the root via cobra's parent-walking flag lookup. HTTP traffic from the
// SDK clients is short-circuited by a fuzzStubTransport that 400s every
// request.
type fuzzHarness struct {
	cli     *cobra.Command
	baseCtx context.Context
	leaves  []leafCommand
}

type leafCommand struct {
	cmd          *cobra.Command
	path         []string // tokens from cli root to this leaf, exclusive of root
	declaredArgs int
}

func newFuzzHarness(t *testing.T) *fuzzHarness {
	t.Helper()

	wc, err := databricks.NewWorkspaceClient((*databricks.Config)(&config.Config{
		Host:          "https://fuzz.invalid",
		Token:         "fuzz",
		AuthType:      "pat",
		HTTPTransport: fuzzStubTransport{},
	}))
	require.NoError(t, err)

	ac, err := databricks.NewAccountClient((*databricks.Config)(&config.Config{
		Host:          "https://fuzz.invalid",
		Token:         "fuzz",
		AuthType:      "pat",
		AccountID:     "00000000-0000-0000-0000-000000000000",
		HTTPTransport: fuzzStubTransport{},
	}))
	require.NoError(t, err)

	// Pre-install everything auto-generated RunE bodies reach for into a base
	// context. Cobra propagates this into subcommands during execute, so we
	// don't need a PersistentPreRunE to re-install it per invocation.
	baseCtx := t.Context()
	baseCtx = logdiag.InitContext(baseCtx)
	baseCtx = cmdctx.SetWorkspaceClient(baseCtx, wc)
	baseCtx = cmdctx.SetAccountClient(baseCtx, ac)
	baseCtx = cmdio.MockDiscard(baseCtx)

	cli := cmd.New(baseCtx)

	// Replace the real PersistentPreRunE (IO/logger/telemetry/user-agent init)
	// with a no-op so it doesn't clobber the context we just set up.
	cli.PersistentPreRunE = nil
	cli.PersistentPostRunE = nil

	// Zero out PreRunE on every leaf so MustWorkspaceClient / MustAccountClient
	// can't reach out for real credentials.
	leaves := collectLeaves(cli)
	for _, l := range leaves {
		l.cmd.PreRunE = nil
	}

	return &fuzzHarness{cli: cli, baseCtx: baseCtx, leaves: leaves}
}

// collectLeaves walks the tree under root and returns every command that has a
// RunE and no subcommands. The returned path excludes root's own name, so it's
// ready to hand to cli.SetArgs.
func collectLeaves(root *cobra.Command) []leafCommand {
	var out []leafCommand
	for _, child := range root.Commands() {
		collectLeavesInto(child, nil, &out)
	}
	return out
}

func collectLeavesInto(cmd *cobra.Command, parentPath []string, out *[]leafCommand) {
	path := append(append([]string{}, parentPath...), cmd.Name())
	children := cmd.Commands()
	if len(children) == 0 {
		if cmd.RunE != nil {
			*out = append(*out, leafCommand{cmd: cmd, path: path, declaredArgs: declaredArgCount(cmd)})
		}
		return
	}
	for _, child := range children {
		collectLeavesInto(child, path, out)
	}
}

// declaredArgCount returns how many positional placeholders follow the command
// name in cmd.Use (e.g. "update-default-warehouse-override NAME UPDATE_MASK TYPE" → 3).
func declaredArgCount(cmd *cobra.Command) int {
	fields := strings.Fields(cmd.Use)
	if len(fields) <= 1 {
		return 0
	}
	return len(fields) - 1
}

var testNameSanitizer = regexp.MustCompile(`[^A-Za-z0-9_./=#-]+`)

func sanitizeTestName(s string) string {
	return testNameSanitizer.ReplaceAllString(s, "_")
}

// run invokes a leaf's RunE directly, recovering panics as test failures.
// Non-panic errors are ignored — we only care about panics here.
//
// This bypasses cobra's full Execute path (flag parsing, PreRun chain,
// telemetry) for speed. Flag-defined-on-root concerns (e.g. root.OutputType
// reading --output) still work because the leaf is parented to the real root,
// and cobra.Command.Flag walks up the parent chain.
func (h *fuzzHarness) run(t *testing.T, leaf leafCommand, args []string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(h.baseCtx, perInvocationTimeout)
	defer cancel()
	leaf.cmd.SetContext(ctx)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic in %q with args=%#v: %v\n\n%s", strings.Join(leaf.path, "/"), args, r, debug.Stack())
		}
	}()

	// Honour cobra's arg validator. Counts it would reject are unreachable in
	// practice, so we don't fail on panics behind them.
	if leaf.cmd.Args != nil {
		if err := leaf.cmd.Args(leaf.cmd, args); err != nil {
			return
		}
	}
	_ = leaf.cmd.RunE(leaf.cmd, args)
}

// leafPathName formats a leaf's path as a slash-joined test name.
func leafPathName(leaf leafCommand) string {
	return strings.Join(leaf.path, "/")
}

// isAutoGenerated returns true for workspace/account leaves. We restrict
// count-fuzzing to auto-generated commands because that's where codegen
// regressions hide; manually written commands like `bundle` get PR review.
func isAutoGenerated(leaf leafCommand) bool {
	if len(leaf.path) == 0 {
		return false
	}
	if leaf.path[0] == "account" {
		return true
	}
	// Everything under a workspace service command (e.g. "warehouses") is
	// auto-generated. Manually-written commands live under named roots like
	// "bundle", "auth", "sync", "fs", etc. The heuristic: anything whose
	// root isn't in this block list is auto-generated.
	manualRoots := map[string]bool{
		"bundle":       true,
		"auth":         true,
		"sync":         true,
		"fs":           true,
		"api":          true,
		"cache":        true,
		"completion":   true,
		"configure":    true,
		"experimental": true,
		"labs":         true,
		"pipelines":    true,
		"psql":         true,
		"selftest":     true,
		"ssh":          true,
		"version":      true,
		"help":         true,
	}
	return !manualRoots[leaf.path[0]]
}

// fuzzableLeaves returns the auto-generated leaves we want to fuzz.
func (h *fuzzHarness) fuzzableLeaves() []leafCommand {
	var out []leafCommand
	for _, leaf := range h.leaves {
		if isAutoGenerated(leaf) {
			out = append(out, leaf)
		}
	}
	return out
}

// TestCountFuzz count-fuzzes every auto-generated workspace/account command.
// For each leaf it invokes RunE with positional-arg counts from 0 to
// declared+1. Guards against codegen regressions like
// https://github.com/databricks/cli/issues/5070, where interactive fallbacks
// in auto-generated commands access positional args out of bounds.
func TestCountFuzz(t *testing.T) {
	h := newFuzzHarness(t)
	leaves := h.fuzzableLeaves()

	for _, leaf := range leaves {
		t.Run(sanitizeTestName(leafPathName(leaf)), func(t *testing.T) {
			t.Parallel()
			for n := 0; n <= leaf.declaredArgs+1; n++ {
				args := make([]string, n)
				for i := range args {
					args[i] = "x"
				}
				h.run(t, leaf, args)
				if t.Failed() {
					return
				}
			}
		})
	}
}
