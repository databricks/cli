package dockercredentials

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func SetCredentialHelper(path, registryHost string) error {
	config, err := readDockerConfig(path)
	if err != nil {
		return err
	}

	helpers := map[string]string{}
	if raw, ok := config["credHelpers"]; ok {
		if err := json.Unmarshal(raw, &helpers); err != nil {
			return fmt.Errorf("read Docker config %s: %w", path, err)
		}
	}
	if helpers == nil {
		helpers = map[string]string{}
	}

	if helpers[registryHost] == HelperName {
		return nil
	}

	helpers[registryHost] = HelperName
	rawHelpers, err := json.Marshal(helpers)
	if err != nil {
		return err
	}
	config["credHelpers"] = rawHelpers

	if err := writeDockerConfig(path, config); err != nil {
		return err
	}
	return nil
}

func readDockerConfig(path string) (map[string]json.RawMessage, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]json.RawMessage{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Docker config %s: %w", path, err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("read Docker config %s: %w", path, err)
	}
	if config == nil {
		config = map[string]json.RawMessage{}
	}
	return config, nil
}

func writeDockerConfig(path string, config map[string]json.RawMessage) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create Docker config directory %s: %w", dir, err)
	}

	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')

	tmp, err := os.CreateTemp(dir, ".config.json.*")
	if err != nil {
		return fmt.Errorf("create temporary Docker config in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary Docker config %s: %w", tmpPath, err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set permissions on temporary Docker config %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary Docker config %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("write Docker config %s: %w", path, err)
	}
	return nil
}
