package env

import (
	"context"
	"errors"
	"os"
	"runtime"
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
