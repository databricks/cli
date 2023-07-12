package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const githubUrl = "https://github.com"
const databricksOrg = "databricks"

type cloneOptions struct {
	// Branch or tag to clone
	Reference string

	// URL for the repository
	RepositoryUrl string

	// Local path to clone repository at
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
		return strings.Join([]string{githubUrl, s}, "/")
	}

	// case: only repository name is provided
	return strings.Join([]string{githubUrl, databricksOrg, s}, "/")
}

func (opts cloneOptions) args() []string {
	args := []string{"clone", opts.RepositoryUrl, opts.TargetPath, "--depth=1", "--no-tags"}
	if opts.Reference != "" {
		args = append(args, "--branch", opts.Reference)
	}
	return args
}

func Clone(ctx context.Context, url, reference, targetPath string) error {
	fullUrl := expandUrl(url)
	opts := cloneOptions{
		Reference:     reference,
		RepositoryUrl: fullUrl,
		TargetPath:    targetPath,
	}

	cmd := exec.CommandContext(ctx, "git", opts.args()...)
	var cmdErr bytes.Buffer
	cmd.Stderr = &cmdErr

	// start git clone
	err := cmd.Start()
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("please install git CLI to clone a repository: %w", err)
	}
	if err != nil {
		return err
	}

	// wait for git clone to complete
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("git clone failed: %w. %s", err, cmdErr.String())
	}
	return nil
}
