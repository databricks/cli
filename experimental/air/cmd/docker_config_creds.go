package aircmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
)

// Docker Hub is keyed under this exact legacy string in ~/.docker/config.json.
const dockerHubAuthKey = "https://index.docker.io/v1/"

// credHelperTimeout bounds a docker-credential-<helper> invocation so a hung
// helper never delays registration.
const credHelperTimeout = 10 * time.Second

// dockerConfig is the subset of ~/.docker/config.json we read.
type dockerConfig struct {
	Auths       map[string]struct{ Auth string } `json:"auths"`
	CredsStore  string                           `json:"credsStore"`
	CredHelpers map[string]string                `json:"credHelpers"`
}

// dockerConfigPath returns ~/.docker/config.json, honoring DOCKER_CONFIG.
func dockerConfigPath(ctx context.Context) (string, error) {
	if override, ok := env.Lookup(ctx, "DOCKER_CONFIG"); ok && override != "" {
		return filepath.Join(override, "config.json"), nil
	}
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".docker", "config.json"), nil
}

// registryKey maps a normalized image URL to the key used in docker config.
// Docker Hub uses the legacy key; everything else is the bare hostname.
func registryKey(imageURL string) string {
	host, _, _ := strings.Cut(imageURL, "/")
	switch host {
	case "docker.io", "index.docker.io", "registry-1.docker.io":
		return dockerHubAuthKey
	}
	return host
}

// decodeDockerAuth decodes a base64 "username:password" auth field.
func decodeDockerAuth(authB64 string) (user, secret string, ok bool) {
	decoded, err := base64.StdEncoding.DecodeString(authB64)
	if err != nil {
		return "", "", false
	}
	u, s, found := strings.Cut(string(decoded), ":")
	if !found || u == "" || s == "" {
		return "", "", false
	}
	return u, s, true
}

// invokeCredHelper runs `docker-credential-<helper> get` for registry and parses
// its JSON. A missing helper, non-zero exit, or timeout yields ok=false; this is
// never an error (we fall through to the next credential source).
func invokeCredHelper(ctx context.Context, helper, registry string) (user, secret string, ok bool) {
	ctx, cancel := context.WithTimeout(ctx, credHelperTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker-credential-"+helper, "get")
	cmd.Stdin = strings.NewReader(registry)
	out, err := cmd.Output()
	if err != nil {
		log.Debugf(ctx, "docker-credential-%s get failed: %v", helper, err)
		return "", "", false
	}

	var payload struct {
		Username string `json:"Username"`
		Secret   string `json:"Secret"`
	}
	if err := json.Unmarshal(out, &payload); err != nil || payload.Username == "" || payload.Secret == "" {
		return "", "", false
	}
	return payload.Username, payload.Secret, true
}

// readDockerCredentials looks up registry credentials for imageURL from the local
// Docker config, mirroring Docker's own resolution order: per-registry helper,
// then the global credsStore, then the inline base64 auth. imageURL must already
// be normalized so the first path segment is the registry host. Returns ok=false
// when no credentials are available; it never errors, since a missing or
// unreadable config just means the caller falls back to the public-image path.
func readDockerCredentials(ctx context.Context, imageURL string) (user, secret string, ok bool) {
	path, err := dockerConfigPath(ctx)
	if err != nil {
		return "", "", false
	}
	// A missing or unreadable config is not an error: fall through to the
	// public-image path.
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", false
	}

	var cfg dockerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Debugf(ctx, "could not parse %s: %v", path, err)
		return "", "", false
	}

	registry := registryKey(imageURL)

	// Per-registry helper takes precedence over everything else.
	if helper := cfg.CredHelpers[registry]; helper != "" {
		if u, s, ok := invokeCredHelper(ctx, helper, registry); ok {
			return u, s, true
		}
	}

	// Global credential store. Consult it before the inline auth field, which may
	// be stale data left from a pre-credsStore `docker login`.
	if cfg.CredsStore != "" {
		if u, s, ok := invokeCredHelper(ctx, cfg.CredsStore, registry); ok {
			return u, s, true
		}
	}

	// Inline base64 auth as the final fallback.
	if entry, present := cfg.Auths[registry]; present && entry.Auth != "" {
		if u, s, ok := decodeDockerAuth(entry.Auth); ok {
			return u, s, true
		}
	}

	return "", "", false
}
