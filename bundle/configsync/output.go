package configsync

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/databricks/cli/bundle"
)

// FileChange represents a change to a bundle configuration file
type FileChange struct {
	Path            string `json:"path"`
	OriginalContent string `json:"originalContent"`
	ModifiedContent string `json:"modifiedContent"`
}

// DiffOutput represents the complete output of the config-remote-sync command
type DiffOutput struct {
	Files   []FileChange `json:"files"`
	Changes Changes      `json:"changes"`
}

// ErrSourceChanged indicates that the source no longer matches the snapshot
// used to plan remote edits. The command reloads and replans on this error.
var ErrSourceChanged = errors.New("source configuration changed")

// ErrSourceRecoveryRequired indicates that an interrupted multi-file commit
// could not be rolled back automatically and left a recovery file on disk.
var ErrSourceRecoveryRequired = errors.New("source configuration recovery required")

// SaveFiles writes all file changes to disk.
func SaveFiles(ctx context.Context, b *bundle.Bundle, files []FileChange) error {
	return saveFiles(ctx, files, os.Rename)
}

type stagedFileChange struct {
	change         FileChange
	existed        bool
	mode           fs.FileMode
	stagedPath     string
	backupPath     string
	committed      bool
	preserveBackup bool
}

func writeStagedFile(path string, content []byte, mode fs.FileMode) (string, error) {
	directory := filepath.Dir(path)
	file, err := os.CreateTemp(directory, "."+filepath.Base(path)+".config-remote-sync-*")
	if err != nil {
		return "", fmt.Errorf("creating staged file for %s: %w", path, err)
	}
	stagedPath := file.Name()
	succeeded := false
	defer func() {
		_ = file.Close()
		if !succeeded {
			_ = os.Remove(stagedPath)
		}
	}()

	if err := file.Chmod(mode); err != nil {
		return "", fmt.Errorf("setting permissions on staged file for %s: %w", path, err)
	}
	if _, err := file.Write(content); err != nil {
		return "", fmt.Errorf("writing staged file for %s: %w", path, err)
	}
	if err := file.Sync(); err != nil {
		return "", fmt.Errorf("syncing staged file for %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("closing staged file for %s: %w", path, err)
	}
	succeeded = true
	return stagedPath, nil
}

func validateSourceSnapshot(staged *stagedFileChange) error {
	content, err := os.ReadFile(staged.change.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) && !staged.existed {
			if staged.change.OriginalContent == "" {
				return nil
			}
			return fmt.Errorf("%w: source file %s disappeared while remote changes were being resolved", ErrSourceChanged, staged.change.Path)
		}
		return fmt.Errorf("reading current source file %s: %w", staged.change.Path, err)
	}
	if !staged.existed || !bytes.Equal(content, []byte(staged.change.OriginalContent)) {
		return fmt.Errorf("%w: source file %s changed while remote changes were being resolved", ErrSourceChanged, staged.change.Path)
	}
	return nil
}

func commitError(primary, rollback error) error {
	if rollback == nil {
		return primary
	}
	return errors.Join(primary, fmt.Errorf("%w: %w", ErrSourceRecoveryRequired, rollback))
}

func rollbackStagedFiles(stagedFiles []*stagedFileChange, rename func(string, string) error) error {
	var rollbackErrors []error
	for _, staged := range slices.Backward(stagedFiles) {

		if !staged.committed {
			continue
		}

		current, err := os.ReadFile(staged.change.Path)
		if err != nil {
			staged.preserveBackup = staged.existed
			rollbackErrors = append(rollbackErrors, fmt.Errorf("reading %s during rollback: %w", staged.change.Path, err))
			continue
		}
		if !bytes.Equal(current, []byte(staged.change.ModifiedContent)) {
			staged.preserveBackup = staged.existed
			rollbackErrors = append(rollbackErrors, fmt.Errorf("refusing to overwrite a concurrent edit to %s during rollback", staged.change.Path))
			continue
		}

		if staged.existed {
			if err := rename(staged.backupPath, staged.change.Path); err != nil {
				staged.preserveBackup = true
				rollbackErrors = append(rollbackErrors, fmt.Errorf("restoring %s from recovery file %s: %w", staged.change.Path, staged.backupPath, err))
				continue
			}
			staged.backupPath = ""
		} else if err := os.Remove(staged.change.Path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("removing newly created file %s: %w", staged.change.Path, err))
			continue
		}
		staged.committed = false
	}
	return errors.Join(rollbackErrors...)
}

func saveFiles(ctx context.Context, files []FileChange, rename func(string, string) error) error {
	ordered := slices.Clone(files)
	slices.SortFunc(ordered, func(left, right FileChange) int {
		return cmp.Compare(left.Path, right.Path)
	})

	seen := make(map[string]struct{}, len(ordered))
	stagedFiles := make([]*stagedFileChange, 0, len(ordered))
	defer func() {
		for _, staged := range stagedFiles {
			if staged.stagedPath != "" {
				_ = os.Remove(staged.stagedPath)
			}
			if staged.backupPath != "" && !staged.preserveBackup {
				_ = os.Remove(staged.backupPath)
			}
		}
	}()

	for _, change := range ordered {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, ok := seen[change.Path]; ok {
			return fmt.Errorf("duplicate file change for %s", change.Path)
		}
		seen[change.Path] = struct{}{}

		if err := os.MkdirAll(filepath.Dir(change.Path), 0o755); err != nil {
			return fmt.Errorf("creating source directory for %s: %w", change.Path, err)
		}

		staged := &stagedFileChange{change: change, mode: 0o644}
		info, err := os.Lstat(change.Path)
		switch {
		case err == nil:
			if info.Mode()&fs.ModeSymlink != 0 {
				return fmt.Errorf("refusing to replace symbolic link %s", change.Path)
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("source path %s is not a regular file", change.Path)
			}
			staged.existed = true
			staged.mode = info.Mode().Perm()
		case errors.Is(err, fs.ErrNotExist):
			staged.existed = false
		default:
			return fmt.Errorf("inspecting source file %s: %w", change.Path, err)
		}

		if err := validateSourceSnapshot(staged); err != nil {
			return err
		}
		staged.stagedPath, err = writeStagedFile(change.Path, []byte(change.ModifiedContent), staged.mode)
		if err != nil {
			return err
		}
		if staged.existed {
			staged.backupPath, err = writeStagedFile(change.Path, []byte(change.OriginalContent), staged.mode)
			if err != nil {
				return err
			}
		}
		stagedFiles = append(stagedFiles, staged)
	}

	for _, staged := range stagedFiles {
		if err := validateSourceSnapshot(staged); err != nil {
			return err
		}
	}

	for _, staged := range stagedFiles {
		if err := ctx.Err(); err != nil {
			rollbackErr := rollbackStagedFiles(stagedFiles, rename)
			return commitError(err, rollbackErr)
		}
		if err := validateSourceSnapshot(staged); err != nil {
			rollbackErr := rollbackStagedFiles(stagedFiles, rename)
			return commitError(err, rollbackErr)
		}
		if err := rename(staged.stagedPath, staged.change.Path); err != nil {
			rollbackErr := rollbackStagedFiles(stagedFiles, rename)
			return commitError(fmt.Errorf("committing source file %s: %w", staged.change.Path, err), rollbackErr)
		}
		staged.stagedPath = ""
		staged.committed = true
	}

	return nil
}
