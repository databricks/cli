package apps

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/log"
)

// gitOnboardResult is the outcome of the interactive Git onboarding prompt that
// runs when `apps init` is started outside any Git repository.
type gitOnboardResult struct {
	// resolved is true when onboarding took ownership of the destination (name,
	// destDir, in-place). When false the caller falls through to the normal
	// name/location prompts and passive detection.
	resolved bool
	appName  string
	destDir  string
	inPlace  bool

	// source overrides the scaffolded git-backed config. It is nil for the
	// clone path, where the freshly cloned origin is picked up by the normal
	// post-scaffold detection instead.
	source *gitScaffoldSource

	// createRepo, when set, is executed after scaffolding to create the GitHub
	// repository from the generated files and push the first commit.
	createRepo *githubRepoCreate
}

// githubRepoCreate captures a deferred `gh repo create` for the "create a new
// repo" path. sourceDir is filled in by the caller once the scaffold
// destination is known.
type githubRepoCreate struct {
	ownerRepo string // "owner/name"
	public    bool
	sourceDir string
}

// runGitOnboarding runs the interactive "no Git repository" flow. It returns a
// zero (unresolved) result — leaving the caller's normal name/location prompts
// in charge — when the user is already in a repo, declines, or a step can't be
// completed. It never fails the scaffold for an optional Git step.
func runGitOnboarding(ctx context.Context, outputDir string) (gitOnboardResult, error) {
	// Only offered when the user has no Git repo set up at all; a repo that
	// already exists (with or without a remote) is left untouched.
	if isInGitRepo(ctx, ".", cmdctx.WorkspaceClient(ctx)) {
		return gitOnboardResult{}, nil
	}

	deploy, err := prompt.PromptDeployFromGit(ctx)
	if err != nil {
		return gitOnboardResult{}, err
	}
	if !deploy {
		return gitOnboardResult{}, nil
	}

	choice, err := prompt.PromptCreateOrExistingRepo(ctx)
	if err != nil {
		return gitOnboardResult{}, err
	}

	switch choice {
	case prompt.GitRepoExisting:
		return onboardExistingRepo(ctx, outputDir)
	case prompt.GitRepoCreate:
		return onboardCreateRepo(ctx, outputDir)
	default:
		return gitOnboardResult{}, nil
	}
}

// onboardExistingRepo clones a user-provided repository into the current
// directory and scaffolds the app in place. Cloning into "." requires an empty
// directory, so a non-empty cwd falls back to the normal flow.
func onboardExistingRepo(ctx context.Context, outputDir string) (gitOnboardResult, error) {
	if outputDir != "" {
		log.Warnf(ctx, "--output-dir is not compatible with cloning into the current directory; continuing without Git setup")
		return gitOnboardResult{}, nil
	}
	if !dirIsEmpty(".") {
		log.Warnf(ctx, "the current directory is not empty; cloning an existing repo requires an empty directory. Continuing without Git setup")
		return gitOnboardResult{}, nil
	}

	rawURL, err := prompt.PromptRepoURL(ctx, "Existing repository URL", "the app is cloned into the current directory and deployed from this repo")
	if err != nil {
		return gitOnboardResult{}, err
	}
	if rawURL == "" {
		return gitOnboardResult{}, nil
	}

	appName, err := prompt.DeriveInPlaceAppName(".")
	if err != nil {
		return gitOnboardResult{}, fmt.Errorf("cloning here needs a directory whose name is a valid app name: %w", err)
	}

	if err := gitCloneInto(ctx, rawURL, "."); err != nil {
		return gitOnboardResult{}, fmt.Errorf("clone %s: %w", rawURL, err)
	}
	// Guard against clobbering: the app is scaffolded in place over the clone,
	// so the repository must be empty (a bare initial repo). A repo with files
	// (README, LICENSE, ...) would have its contents overwritten by the scaffold.
	if !dirHasOnlyGit(".") {
		return gitOnboardResult{}, fmt.Errorf("cloned repository %s is not empty; scaffolding would overwrite its files — start from an empty repository", rawURL)
	}
	prompt.PrintAnswered(ctx, "Cloned", rawURL)

	// The freshly cloned origin is detected by the normal post-scaffold pass,
	// so no source override is needed here.
	return gitOnboardResult{
		resolved: true,
		appName:  appName,
		destDir:  ".",
		inPlace:  true,
	}, nil
}

// onboardCreateRepo gathers details for a new GitHub repo. With an authenticated
// gh CLI it predicts the repo URL and defers creation until after scaffolding;
// otherwise it falls back to writing the git-backed config from a URL the user
// supplies, leaving repo creation to them.
func onboardCreateRepo(ctx context.Context, outputDir string) (gitOnboardResult, error) {
	name, private, err := prompt.PromptNewRepoDetails(ctx, defaultRepoName())
	if err != nil {
		return gitOnboardResult{}, err
	}

	destDir := name
	if outputDir != "" {
		destDir = filepath.Join(outputDir, name)
	}

	// gh must be installed and authenticated to create the repo for the user.
	if login, ok := ghLogin(ctx); ok {
		ownerRepo := login + "/" + name
		src := &gitScaffoldSource{
			URL:            "https://github.com/" + ownerRepo,
			Provider:       "gitHub",
			Branch:         "main",
			SourceCodePath: "./",
		}
		return gitOnboardResult{
			resolved:   true,
			appName:    name,
			destDir:    destDir,
			source:     src,
			createRepo: &githubRepoCreate{ownerRepo: ownerRepo, public: !private},
		}, nil
	}

	// Fallback: no usable gh. Ask for the URL and write the git-backed config;
	// the user creates and pushes the repo themselves.
	log.Warnf(ctx, "gh CLI not installed or not authenticated — writing git-backed config without creating the repo. Install and authenticate gh (https://cli.github.com), then create the repo and push before deploying.")
	rawURL, err := prompt.PromptRepoURL(ctx, "Git repository URL", "create this repo and push the app before deploying")
	if err != nil {
		return gitOnboardResult{}, err
	}
	repoURL, provider := normalizeGitOrigin(rawURL)
	if repoURL == "" || provider == "" {
		log.Warnf(ctx, "could not determine a Git provider from %q; continuing without Git setup", rawURL)
		return gitOnboardResult{resolved: true, appName: name, destDir: destDir}, nil
	}
	return gitOnboardResult{
		resolved: true,
		appName:  name,
		destDir:  destDir,
		source:   &gitScaffoldSource{URL: repoURL, Provider: provider, Branch: "main", SourceCodePath: "./"},
	}, nil
}

// run creates the GitHub repository from the scaffolded files: it initializes a
// repo, commits, then `gh repo create ... --push`.
func (c *githubRepoCreate) run(ctx context.Context) error {
	init := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "add", "-A"},
		{"git", "commit", "-m", "Initial commit from databricks apps init"},
	}
	for _, args := range init {
		if err := runIn(ctx, c.sourceDir, args...); err != nil {
			return err
		}
	}
	visibility := "--private"
	if c.public {
		visibility = "--public"
	}
	return runIn(ctx, c.sourceDir, "gh", "repo", "create", c.ownerRepo, visibility, "--source", ".", "--remote", "origin", "--push")
}

// runIn runs a command in dir, wrapping failures with the combined output.
func runIn(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// gitCloneInto clones repoURL into dir (which must be empty for ".").
func gitCloneInto(ctx context.Context, repoURL, dir string) error {
	return runIn(ctx, ".", "git", "clone", repoURL, dir)
}

// ghLogin returns the authenticated GitHub username when the gh CLI is
// installed and logged in, and false otherwise.
func ghLogin(ctx context.Context) (string, bool) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", false
	}
	if err := exec.CommandContext(ctx, "gh", "auth", "status").Run(); err != nil {
		return "", false
	}
	out, err := exec.CommandContext(ctx, "gh", "api", "user", "--jq", ".login").Output()
	if err != nil {
		return "", false
	}
	login := strings.TrimSpace(string(out))
	if login == "" {
		return "", false
	}
	return login, true
}

// defaultRepoName suggests the current directory's basename as the repo/app
// name when it is a valid app name, else a generic default.
func defaultRepoName() string {
	if cwd, err := os.Getwd(); err == nil {
		base := filepath.Base(cwd)
		if prompt.ValidateProjectName(base) == nil {
			return base
		}
	}
	return "my-app"
}

// dirIsEmpty reports whether dir has no entries.
func dirIsEmpty(dir string) bool {
	entries, err := os.ReadDir(dir)
	return err == nil && len(entries) == 0
}

// dirHasOnlyGit reports whether dir contains nothing but a .git entry (i.e. a
// freshly cloned empty repository).
func dirHasOnlyGit(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Name() != ".git" {
			return false
		}
	}
	return true
}
