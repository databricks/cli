package phases

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
)

const gitFreshnessTimeout = 30 * time.Second

// CheckGitBranchFreshness asks for approval when the current branch is behind its upstream.
func CheckGitBranchFreshness(ctx context.Context, b *bundle.Bundle) error {
	if b.Config.Experimental == nil || !config.IsExplicitlyEnabled(b.Config.Experimental.CheckGitBranchFreshness) {
		return nil
	}

	checkCtx, cancel := context.WithTimeout(ctx, gitFreshnessTimeout)
	defer cancel()

	branch, upstream, behind, err := gitBranchFreshness(checkCtx, b.BundleRoot.Native())
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if checkCtx.Err() != nil {
			err = checkCtx.Err()
		}
		logdiag.LogDiag(ctx, diag.Warningf("could not check Git branch freshness: %s; run git fetch and verify the branch's upstream", err)[0])
		return nil
	}
	if behind == 0 {
		return nil
	}

	message := fmt.Sprintf("Git branch %q is %d commit(s) behind its upstream %q", branch, behind, upstream)
	logdiag.LogDiag(ctx, diag.Warningf("%s", message)[0])
	if b.AutoApprove {
		return nil
	}
	if !cmdio.IsPromptSupported(ctx) {
		return fmt.Errorf("%s; the current console does not support prompting. To proceed, use --auto-approve (which also approves other deployment actions)", message)
	}
	approved, err := cmdio.AskYesOrNo(ctx, "Deploy from this branch anyway?")
	if err != nil {
		return fmt.Errorf("confirming deployment from a stale Git branch: %w", err)
	}
	if !approved {
		return errors.New("deployment cancelled")
	}
	return nil
}

func gitBranchFreshness(ctx context.Context, dir string) (string, string, int, error) {
	branch, err := gitFreshnessCommand(ctx, dir, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return "", "", 0, fmt.Errorf("cannot determine the current branch: %w", err)
	}
	// https://git-scm.com/docs/git-config#Documentation/git-config.txt-branchltnamegtmerge
	remote, err := gitFreshnessCommand(ctx, dir, "config", "--get", "branch."+branch+".remote")
	if err != nil {
		return "", "", 0, fmt.Errorf("cannot determine the upstream of branch %q: %w", branch, err)
	}
	refs, err := gitFreshnessCommand(ctx, dir, "config", "--get-all", "branch."+branch+".merge")
	if err != nil {
		return "", "", 0, fmt.Errorf("cannot determine the upstream of branch %q: %w", branch, err)
	}
	ref, _, _ := strings.Cut(refs, "\n")
	if remote == "" || ref == "" {
		return "", "", 0, fmt.Errorf("cannot determine the upstream of branch %q", branch)
	}
	upstream := strings.TrimPrefix(ref, "refs/heads/")
	if remote != "." {
		remotes, err := gitFreshnessCommand(ctx, dir, "remote")
		if err != nil {
			return "", "", 0, fmt.Errorf("cannot list Git remotes: %w", err)
		}
		// branch.<name>.remote can be a URL containing credentials, not just a remote name.
		if slices.Contains(strings.Split(remotes, "\n"), remote) {
			upstream = remote + "/" + upstream
		}
	}
	if _, err := gitFreshnessCommand(ctx, dir, "fetch", "--quiet", "--", remote, ref); err != nil {
		return "", "", 0, fmt.Errorf("git fetch failed: %w", err)
	}
	// The tracking ref may not be updated if the remote's fetch refspec excludes it.
	count, err := gitFreshnessCommand(ctx, dir, "rev-list", "--count", "HEAD..FETCH_HEAD")
	if err != nil {
		return "", "", 0, fmt.Errorf("cannot compare branch %q with its upstream: %w", branch, err)
	}
	behind, err := strconv.Atoi(count)
	if err != nil || behind < 0 {
		return "", "", 0, fmt.Errorf("invalid Git commit count %q", count)
	}
	return branch, upstream, behind, nil
}

func gitFreshnessCommand(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.WaitDelay = time.Second
	// https://git-scm.com/docs/git#Documentation/git.txt-codeGITTERMINALPROMPTcode
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
