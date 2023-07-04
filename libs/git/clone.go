package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
)

const GithubUrl = "https://github.com"
const DatabricksOrg = "databricks"

type cloneOptions struct {
	// Branch or tag to clone
	Reference string

	// URL for the repository
	RepositoryUrl string

	// Path to clone repository at
	TargetPath string
}

// Expands input into a repository URL. Handles three cases:
//  1. Full URL provided. This function does nothing.
//  2. Only org and repo-name are provided. Github is assumed as default provider in
//     this case Eg: databricks/cli -> https://github.com/databricks/cli.
//  3. Only repository is provided. Github is assumed as default provider. Databricks
//     is assumed as the repository owner. Eg: cli -> https://github.com/databricks/cli.
func expandUrl(s string) string {
	// case: full url
	if strings.HasPrefix(s, "git@") || strings.HasPrefix(s, "https://") {
		return s
	}

	// case: org and repository name are provided
	if strings.Contains(s, "/") {
		return strings.Join([]string{GithubUrl, s}, "/")
	}

	// case: only repository name is provided
	return strings.Join([]string{GithubUrl, DatabricksOrg, s}, "/")
}

func parseCloneOptions(url, targetPath string) cloneOptions {
	repoUrl := expandUrl(url)
	reference := "main"

	// Users can optionally specify a branch / tag by adding @my-branch to the end
	// of the URL. eg: https://github.com/databricks/cli.git@release-branch
	parts := strings.SplitN(repoUrl, "@", 2)
	if len(parts) == 2 && parts[1] != "" {
		reference = parts[1]
		repoUrl = parts[0]
	}
	return cloneOptions{
		Reference:     reference,
		RepositoryUrl: repoUrl,
		TargetPath:    targetPath,
	}

}

func (opts cloneOptions) args() []string {
	return []string{"clone", opts.RepositoryUrl, opts.TargetPath, "--branch", opts.Reference, "--depth=1", "--no-tags"}
}

func Clone(ctx context.Context, url, targetPath string) error {
	opts := parseCloneOptions(url, targetPath)
	cmd := exec.CommandContext(ctx, "git", opts.args()...)

	// Redirect exec command output
	cmd.Stderr = cmdio.Err(ctx)
	cmd.Stdout = cmdio.Out(ctx)
	cmd.Stdin = cmdio.In(ctx)

	// start git clone
	err := cmd.Start()
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("please install git CLI to download templates: %w", err)
	}
	if err != nil {
		return err
	}

	// wait for git clone to complete
	return cmd.Wait()
}
