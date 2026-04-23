package ucm

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/folders"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/config"
	"github.com/spf13/cobra"
)

// Default refs for the diff verb. Tuned for GitHub-style PR CI where
// origin/main is the target branch and HEAD is the topic branch tip.
const (
	defaultDiffBase = "origin/main"
	defaultDiffHead = "HEAD"
)

// diffRunner is the in-memory form of the ucm diff invocation. Kept as a
// struct so tests can drive it without plumbing cobra flags.
type diffRunner struct {
	base   string
	head   string
	output flags.Output

	// workDir is the cwd to run git commands from and the anchor for the
	// working-tree ucm.yml lookup. Set by the cobra RunE from os.Getwd.
	workDir string

	// gitRun shells out to `git <args...>` in workDir and returns stdout.
	// Factored out so tests can inject a fake git.
	gitRun func(args ...string) ([]byte, error)
}

// diffResult is the structured output of a diff run. Emitted verbatim when
// -o json; rendered to text otherwise.
type diffResult struct {
	ChangedResources []string `json:"changed_resources"`
}

func newDiffCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Detect which ucm stacks changed since a base git ref. Intended for CI matrices.",
		Long: `Detect which ucm resources changed between two git refs.

Compares ucm.yml at --base (default origin/main) with ucm.yml at --head
(default HEAD, i.e. the current working tree if clean) and prints the set of
resource keys that were added, removed, or modified. Intended for CI matrix
generation — pair with -o json and feed into a matrix strategy.

Common invocations:
  databricks ucm diff                        # Diff HEAD vs origin/main
  databricks ucm diff --base main            # Diff against local main
  databricks ucm diff --base v1.2.3 -o json  # JSON output for CI`,
		Args: root.NoArgs,
	}

	var base, head string
	cmd.Flags().StringVar(&base, "base", defaultDiffBase, "Git ref to diff against.")
	cmd.Flags().StringVar(&head, "head", defaultDiffHead, "Git ref (or HEAD) to diff from. HEAD reads the working tree.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if !logdiag.IsSetup(ctx) {
			ctx = logdiag.InitContext(ctx)
			cmd.SetContext(ctx)
		}

		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve working directory: %w", err)
		}

		r := &diffRunner{
			base:    base,
			head:    head,
			output:  diffOutputType(cmd),
			workDir: wd,
			gitRun:  defaultGitRun(wd),
		}
		return r.run(cmd.OutOrStdout())
	}

	return cmd
}

// diffOutputType mirrors planOutputType: returns OutputText when the flag is
// not wired (e.g. in unit tests invoking the command directly).
func diffOutputType(cmd *cobra.Command) flags.Output {
	if cmd.Flag("output") == nil {
		return flags.OutputText
	}
	return root.OutputType(cmd)
}

// defaultGitRun returns a gitRun closure rooted at dir. The -C flag makes the
// subshell independent of the parent process's cwd.
func defaultGitRun(dir string) func(args ...string) ([]byte, error) {
	return func(args ...string) ([]byte, error) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
		}
		return out, nil
	}
}

func (r *diffRunner) run(out io.Writer) error {
	headRoot, err := r.loadHead()
	if err != nil {
		return fmt.Errorf("load ucm.yml at head: %w", err)
	}

	baseRoot, err := r.loadBase()
	if err != nil {
		return fmt.Errorf("load ucm.yml at %s: %w", r.base, err)
	}

	changed := diffResources(baseRoot, headRoot)
	result := diffResult{ChangedResources: changed}

	switch r.output {
	case flags.OutputJSON:
		return writeDiffJSON(out, result)
	default:
		writeDiffText(out, result, r.base)
		return nil
	}
}

// loadHead reads ucm.yml from the working tree when --head is HEAD, or from
// git when --head names another ref. The working-tree path is preferred since
// it reflects uncommitted edits — matches how bundle validate works.
func (r *diffRunner) loadHead() (*config.Root, error) {
	if r.head == defaultDiffHead {
		return loadWorkingTreeUcm(r.workDir)
	}
	return r.loadFromGit(r.head)
}

// loadBase reads ucm.yml at --base via git show. When the file does not
// exist at the base ref (e.g. first PR introducing ucm.yml) an empty Root is
// returned so every resource at head shows up as "added".
func (r *diffRunner) loadBase() (*config.Root, error) {
	return r.loadFromGit(r.base)
}

// loadWorkingTreeUcm walks up from dir to find a ucm.yml (mirrors the engine's
// root traversal) and returns its typed + dyn tree.
func loadWorkingTreeUcm(dir string) (*config.Root, error) {
	for _, name := range config.FileNames {
		root, err := folders.FindDirWithLeaf(dir, name)
		if err != nil {
			continue
		}
		file, err := config.FileNames.FindInPath(root)
		if err != nil {
			return nil, err
		}
		cfg, diags := config.Load(file)
		if diags.HasError() {
			return nil, diags.Error()
		}
		return cfg, nil
	}
	return nil, fmt.Errorf("unable to locate ucm root: %s not found", config.FileNames[0])
}

// loadFromGit resolves the ucm.yml path relative to the git repo root, then
// uses `git show <ref>:<path>` to fetch its contents at that ref. A missing
// file at the ref is treated as an empty config (added resources only).
func (r *diffRunner) loadFromGit(ref string) (*config.Root, error) {
	repoRoot, err := r.gitRun("rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("resolve git repo root: %w", err)
	}
	repo := strings.TrimSpace(string(repoRoot))

	for _, name := range config.FileNames {
		relPath, err := findRelUcmPath(repo, r.workDir, name)
		if err != nil {
			continue
		}
		raw, err := r.gitRun("show", ref+":"+filepath.ToSlash(relPath))
		if err != nil {
			// File may not exist at this ref yet — fall through to the next
			// candidate and, failing that, return an empty Root.
			continue
		}
		cfg, diags := config.LoadFromBytes(relPath, raw)
		if diags.HasError() {
			return nil, diags.Error()
		}
		return cfg, nil
	}
	// No ucm.yml/.yaml found at ref: return an empty Root so diff reports
	// everything at head as newly added.
	return &config.Root{}, nil
}

// findRelUcmPath locates leaf under workDir (or a parent up to repo) and
// returns its path relative to repo. Returns an error if the file is not
// found — callers fall through to the next candidate name.
func findRelUcmPath(repo, workDir, leaf string) (string, error) {
	dir, err := folders.FindDirWithLeaf(workDir, leaf)
	if err != nil {
		return "", err
	}
	full := filepath.Join(dir, leaf)
	rel, err := filepath.Rel(repo, full)
	if err != nil {
		return "", err
	}
	return rel, nil
}

// diffResources returns the sorted list of dotted resource keys that changed
// (added, removed, or modified) between base and head. Keys are prefixed with
// "resources." to match the bundle plan convention.
func diffResources(base, head *config.Root) []string {
	var changed []string
	kinds := resourceKinds(base, head)

	for _, kind := range kinds {
		baseEntries := resourceEntries(base, kind)
		headEntries := resourceEntries(head, kind)

		for _, name := range union(baseEntries, headEntries) {
			if !resourceEqual(baseEntries[name], headEntries[name]) {
				changed = append(changed, fmt.Sprintf("resources.%s.%s", kind, name))
			}
		}
	}
	sort.Strings(changed)
	return changed
}

// resourceKinds returns the sorted set of resource-kind keys ("catalogs",
// "schemas", ...) present at either side of the diff. Derived from the dyn
// tree rather than the Resources struct so the comparison survives schema
// evolution without code changes.
func resourceKinds(base, head *config.Root) []string {
	seen := map[string]struct{}{}
	collect := func(r *config.Root) {
		if r == nil {
			return
		}
		m, ok := r.Value().Get("resources").AsMap()
		if !ok {
			return
		}
		for _, k := range m.Keys() {
			seen[k.MustString()] = struct{}{}
		}
	}
	collect(base)
	collect(head)

	kinds := make([]string, 0, len(seen))
	for k := range seen {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	return kinds
}

// resourceEntries returns the map-of-resources under resources.<kind> as
// typed Go values. The nil-safe AsAny flatten keeps the comparison logic in
// diffResources trivial.
func resourceEntries(r *config.Root, kind string) map[string]any {
	out := map[string]any{}
	if r == nil {
		return out
	}
	v := r.Value().Get("resources").Get(kind)
	m, ok := v.AsMap()
	if !ok {
		return out
	}
	for _, pair := range m.Pairs() {
		out[pair.Key.MustString()] = pair.Value.AsAny()
	}
	return out
}

// union returns the sorted union of keys across two maps.
func union(a, b map[string]any) []string {
	seen := map[string]struct{}{}
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// resourceEqual compares two resource subtrees. Either side may be nil, which
// means the resource is absent on that side (added or removed). Uses
// reflect.DeepEqual on the AsAny flattening so YAML ordering and location
// metadata don't cause spurious diffs.
func resourceEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return reflect.DeepEqual(a, b)
}

// writeDiffText renders the per-resource "changed" list plus a trailing
// summary, or the no-change sentinel when nothing moved.
func writeDiffText(out io.Writer, r diffResult, base string) {
	if len(r.ChangedResources) == 0 {
		fmt.Fprintf(out, "No ucm resource changes since %s.\n", base)
		return
	}
	width := 0
	for _, key := range r.ChangedResources {
		if len(key) > width {
			width = len(key)
		}
	}
	for _, key := range r.ChangedResources {
		fmt.Fprintf(out, "%-*s  changed\n", width, key)
	}
}

// writeDiffJSON emits the structured result. The schema is stable (the
// "changed_resources" field) and consumed by CI matrix generators.
func writeDiffJSON(out io.Writer, r diffResult) error {
	if r.ChangedResources == nil {
		r.ChangedResources = []string{}
	}
	buf, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(buf))
	return err
}
