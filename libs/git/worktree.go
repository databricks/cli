package git

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/vfs"
)

func readLines(root vfs.Path, name string) ([]string, error) {
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

// readGitDir reads the value of the `.git` file in a worktree.
func readGitDir(root vfs.Path) (string, error) {
	lines, err := readLines(root, GitDirectoryName)
	if err != nil {
		return "", err
	}

	var gitDir string
	for _, line := range lines {
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}

		if parts[0] == "gitdir" {
			gitDir = strings.TrimSpace(parts[1])
		}
	}

	if gitDir == "" {
		return "", fmt.Errorf(`expected %q to contain a line with "gitdir: [...]"`, filepath.Join(root.Native(), GitDirectoryName))
	}

	return gitDir, nil
}

// readGitCommonDir reads the value of the `commondir` file in the `.git` directory of a worktree.
// This file typically contains "../.." to point to $GIT_COMMON_DIR.
func readGitCommonDir(gitDir vfs.Path) (string, error) {
	lines, err := readLines(gitDir, "commondir")
	if err != nil {
		return "", err
	}

	if len(lines) == 0 {
		return "", errors.New("file is empty")
	}

	return strings.TrimSpace(lines[0]), nil
}

// resolveGitDirs resolves the paths for $GIT_DIR and $GIT_COMMON_DIR.
// The path argument is the root of the checkout where (supposedly) a `.git` file or directory exists.
func resolveGitDirs(root vfs.Path) (vfs.Path, vfs.Path, error) {
	fileInfo, err := root.Stat(GitDirectoryName)
	if err != nil {
		// If the `.git` file or directory does not exist, then this is not a git repository.
		// Return paths that we know don't exist, so we do not need to perform nil checks in the caller.
		if errors.Is(err, fs.ErrNotExist) {
			gitDir := vfs.MustNew(filepath.Join(root.Native(), GitDirectoryName))
			return gitDir, gitDir, nil
		}
		return nil, nil, err
	}

	// If the path is a directory, then it is the main working tree.
	// Both $GIT_DIR and $GIT_COMMON_DIR point to the same directory.
	if fileInfo.IsDir() {
		gitDir := vfs.MustNew(filepath.Join(root.Native(), GitDirectoryName))
		return gitDir, gitDir, nil
	}

	// If the path is not a directory, then it is a worktree.
	// Read value for $GIT_DIR.
	gitDirValue, err := readGitDir(root)
	if err != nil {
		return nil, nil, err
	}

	// Resolve $GIT_DIR.
	var gitDir vfs.Path
	if filepath.IsAbs(gitDirValue) {
		gitDir = vfs.MustNew(gitDirValue)
	} else {
		gitDir = vfs.MustNew(filepath.Join(root.Native(), gitDirValue))
	}

	// Read value for $GIT_COMMON_DIR.
	gitCommonDirValue, err := readGitCommonDir(gitDir)
	if err != nil {
		return nil, nil, fmt.Errorf(`expected "commondir" file in worktree git folder at %q: %w`, gitDir.Native(), err)
	}

	// Resolve $GIT_COMMON_DIR.
	var gitCommonDir vfs.Path
	if filepath.IsAbs(gitCommonDirValue) {
		gitCommonDir = vfs.MustNew(gitCommonDirValue)
	} else {
		gitCommonDir = vfs.MustNew(filepath.Join(gitDir.Native(), gitCommonDirValue))
	}

	return gitDir, gitCommonDir, nil
}
