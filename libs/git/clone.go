package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
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
}

func (opts cloneOptions) args(shallow bool) []string {
	args := []string{"clone", opts.RepositoryUrl, opts.TargetPath, "--no-tags"}
	if opts.Reference != "" {
		args = append(args, "--branch", opts.Reference)
	}
	if shallow {
		args = append(args, "--depth=1")
	}
	return args
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
	}

	cmd := exec.CommandContext(ctx, "git", opts.args(true)...)
	var cmdErr bytes.Buffer
	cmd.Stderr = &cmdErr

	// start git clone
	err := cmd.Start()
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("please install git CLI to clone a repository: %w", err)
	}
	if err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// wait for git clone to complete
	err = cmd.Wait()
	// Git repos hosted via HTTP do not support shallow cloning. We try with
	// a deep clone this time
	if err != nil && strings.Contains(cmdErr.String(), "dumb http transport does not support shallow capabilities") {
		retryCmd := exec.CommandContext(ctx, "git", opts.args(false)...)
		var retryCmdErr bytes.Buffer
		retryCmd.Stderr = &retryCmdErr
		retryErr := retryCmd.Run()
		if retryErr != nil {
			return fmt.Errorf("git clone failed: %w. %s", retryErr, retryCmdErr.String())
		} else {
			return nil
		}
	}
	if err != nil {
		return fmt.Errorf("git clone failed: %w. %s", err, cmdErr.String())
	}
	return nil
}
