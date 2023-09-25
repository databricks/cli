package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/databricks/cli/libs/process"
)

// source: https://stackoverflow.com/questions/59081778/rules-for-special-characters-in-github-repository-name
var githubRepoRegex = regexp.MustCompile(`^[\w-\.]+$`)

const githubUrl = "https://github.com"
const databricksOrg = "databricks"

type cloneOptions struct {
	// Branch or tag to clone
	Reference string

	// URL for the repository
	RepositoryUrl string

	// Local path to clone repository at
	TargetPath string

	// If true, the repository is shallow cloned
	Shallow bool
}

func (opts cloneOptions) args() []string {
	args := []string{"clone", opts.RepositoryUrl, opts.TargetPath, "--no-tags"}
	if opts.Reference != "" {
		args = append(args, "--branch", opts.Reference)
	}
	if opts.Shallow {
		args = append(args, "--depth=1")
	}
	return args
}

func (opts cloneOptions) clone(ctx context.Context) error {
	// start and wait for git clone to complete
	_, err := process.Background(ctx, append([]string{"git"}, opts.args()...))
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("please install git CLI to clone a repository: %w", err)
	}
	if err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}
	return nil
}

func Clone(ctx context.Context, url, reference, targetPath string) error {
	// We assume only the repository name has been if input does not contain any
	// `/` characters and the url is only made up of alphanumeric characters and
	// ".", "_" and "-". This repository is resolved again databricks github account.
	fullUrl := url
	if githubRepoRegex.MatchString(url) {
		fullUrl = strings.Join([]string{githubUrl, databricksOrg, url}, "/")
	}

	opts := cloneOptions{
		Reference:     reference,
		RepositoryUrl: fullUrl,
		TargetPath:    targetPath,
		Shallow:       true,
	}

	err := opts.clone(ctx)
	// Git repos hosted via HTTP do not support shallow cloning. We try with
	// a deep clone this time
	if err != nil && strings.Contains(err.Error(), "dumb http transport does not support shallow capabilities") {
		opts.Shallow = false
		return opts.clone(ctx)
	}
	return err
}
