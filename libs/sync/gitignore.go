package sync

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/log"
)

func WriteGitIgnore(ctx context.Context, dir string) {
	gitignorePath := filepath.Join(dir, ".databricks", ".gitignore")
	file, err := os.OpenFile(gitignorePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return
		}
		log.Debugf(ctx, "Failed to create %s: %s", gitignorePath, err)
	}

	defer file.Close()
	_, err = file.WriteString("*\n")
	if err != nil {
		log.Debugf(ctx, "Error writing to %s: %s", gitignorePath, err)
	}
}
