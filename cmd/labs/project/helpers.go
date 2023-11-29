package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/env"
)

func PathInLabs(ctx context.Context, dirs ...string) (string, error) {
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	prefix := []string{homeDir, ".databricks", "labs"}
	return filepath.Join(append(prefix, dirs...)...), nil
}

func tryLoadAndParseJSON[T any](jsonFile string) (*T, error) {
	raw, err := os.ReadFile(jsonFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", jsonFile, err)
	}
	var v T
	err = json.Unmarshal(raw, &v)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", jsonFile, err)
	}
	return &v, nil
}
