package env

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
)

var envContextKey int

func copyMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func getMap(ctx context.Context) map[string]string {
	if ctx == nil {
		return nil
	}
	m, ok := ctx.Value(&envContextKey).(map[string]string)
	if !ok {
		return nil
	}
	return m
}

func setMap(ctx context.Context, m map[string]string) context.Context {
	return context.WithValue(ctx, &envContextKey, m)
}

// Lookup key in the context or the the environment.
// Context has precedence.
func Lookup(ctx context.Context, key string) (string, bool) {
	m := getMap(ctx)

	// Return if the key is set in the context.
	v, ok := m[key]
	if ok {
		return v, true
	}

	// Fall back to the environment.
	return os.LookupEnv(key)
}

// Get key from the context or the environment.
// Context has precedence.
func Get(ctx context.Context, key string) string {
	v, _ := Lookup(ctx, key)
	return v
}

// GetBool gets a boolean value from the context or environment.
// Returns (value, true) if the key is set, or (false, false) if not set.
// It accepts various boolean-like values:
// - True: "1", "t", "T", "true", "TRUE", "True", "yes", "YES", "Yes", "on", "ON", "On"
// - False: "0", "f", "F", "false", "FALSE", "False", "no", "NO", "No", "off", "OFF", "Off", "" (empty string)
// Invalid values are treated as false but still return ok=true.
func GetBool(ctx context.Context, key string) (bool, bool) {
	v, ok := Lookup(ctx, key)
	if !ok {
		return false, false
	}

	// Empty string is treated as false
	if v == "" {
		return false, true
	}

	// Handle additional boolean-like values not covered by strconv.ParseBool
	switch strings.ToLower(v) {
	case "yes", "on":
		return true, true
	case "no", "off":
		return false, true
	}

	// Use strconv.ParseBool for standard boolean parsing
	// It handles: "1", "t", "T", "true", "TRUE", "True" (true)
	// and: "0", "f", "F", "false", "FALSE", "False" (false)
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, true
	}
	return b, true
}

// Set key on the context.
//
// Note: this does NOT mutate the processes' actual environment variables.
// It is only visible to other code that uses this package.
func Set(ctx context.Context, key, value string) context.Context {
	m := copyMap(getMap(ctx))
	m[key] = value
	return setMap(ctx, m)
}

func HomeEnvVar() string {
	if runtime.GOOS == "windows" {
		return "USERPROFILE"
	}
	return "HOME"
}

func WithUserHomeDir(ctx context.Context, value string) context.Context {
	return Set(ctx, HomeEnvVar(), value)
}

// ErrNoHomeEnv indicates the absence of $HOME env variable
var ErrNoHomeEnv = errors.New("$HOME is not set")

func UserHomeDir(ctx context.Context) (string, error) {
	home := Get(ctx, HomeEnvVar())
	if home == "" {
		return "", ErrNoHomeEnv
	}
	return home, nil
}

// All returns environment variables that are defined in both os.Environ
// and this package. `env.Set(ctx, x, y)` will override x from os.Environ.
func All(ctx context.Context) map[string]string {
	m := map[string]string{}
	for _, line := range os.Environ() {
		split := strings.SplitN(line, "=", 2)
		if len(split) != 2 {
			continue
		}
		m[split[0]] = split[1]
	}
	// override existing environment variables with the ones we set
	for k, v := range getMap(ctx) {
		m[k] = v
	}
	return m
}
