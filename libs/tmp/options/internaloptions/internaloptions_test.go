package internaloptions

import (
	"context"
	"log/slog"
	"testing"

	"github.com/databricks/cli/libs/tmp/auth"
)

// stubCredentials satisfies the credentials requirement of Resolve for tests
// that exercise other resolution behavior (e.g. logger defaulting).
type stubCredentials struct{}

func (stubCredentials) Name() string { return "stub" }

func (stubCredentials) AuthHeaders(context.Context) ([]auth.Header, error) { return nil, nil }

func TestClientOptionsResolve_DefaultLogger(t *testing.T) {
	c := &ClientOptions{Credentials: stubCredentials{}}
	if err := c.Resolve(); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if c.Logger == nil {
		t.Fatal("expected default logger to be set")
	}
	if c.Logger.Enabled(context.Background(), slog.LevelError) {
		t.Fatal("expected default logger to be disabled")
	}
}

func TestClientOptionsResolve_PreservesProvidedLogger(t *testing.T) {
	provided := slog.Default()
	c := &ClientOptions{Logger: provided, Credentials: stubCredentials{}}
	if err := c.Resolve(); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if c.Logger != provided {
		t.Fatal("expected provided logger to be preserved")
	}
}
