package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/require"
)

func TestRewriteRefreshRecoveryPreservesCauseAndGuidance(t *testing.T) {
	previous := config.DefaultHostMetadataResolverFactory
	metadataCalls := 0
	config.DefaultHostMetadataResolverFactory = func(*config.Config) config.HostMetadataResolver {
		metadataCalls++
		return func(context.Context, string) (*config.HostMetadata, error) {
			return nil, errors.New("unexpected guidance metadata call")
		}
	}
	t.Cleanup(func() { config.DefaultHostMetadataResolverFactory = previous })
	cases := []struct{ name, host, account, profile, command string }{
		{name: "profile", profile: "test-profile", command: "databricks auth login --profile test-profile"},
		{name: "no-profile", command: "databricks auth login"},
		{name: "unnamed-workspace", host: "https://workspace.example.test", command: "databricks auth login --host https://workspace.example.test"},
		{name: "unnamed-account", host: "https://accounts.example.test", account: "account-fixture", command: "databricks auth login --host https://accounts.example.test --account-id account-fixture"},
		{name: "unnamed-unified-host", host: "https://unified.example.test", account: "account-fixture", command: "databricks auth login --host https://unified.example.test --account-id account-fixture"},
		{name: "profile-precedence", host: "https://workspace.example.test", account: "account-fixture", profile: "test-profile", command: "databricks auth login --profile test-profile"},
		{name: "invalid-workspace-host", host: "invalid-host-sentinel", command: "databricks auth login"},
		{name: "invalid-account-host", host: "https://invalid.example.test/", account: "account-fixture", command: "databricks auth login"},
		{name: "account-without-host", account: "account-fixture", command: "databricks auth login"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			cause := fmt.Errorf("outer: %w", storage.ErrRefreshRecoveryRequired)
			rewritten, err := RewriteAuthError(t.Context(), test.host, test.account, test.profile, cause)
			if !rewritten || !errors.Is(err, storage.ErrRefreshRecoveryRequired) || !strings.Contains(err.Error(), "uncertain outcome; automatic refresh is blocked") {
				t.Fatal("recovery guidance lost its typed cause")
			}
			_, command, found := strings.Cut(err.Error(), "\n  $ ")
			require.True(t, found)
			require.Equal(t, test.command, command)
			if strings.Contains(err.Error(), "refresh token is invalid") || strings.Contains(err.Error(), "report this issue") {
				t.Fatal("uncertain refresh was mislabeled")
			}
		})
	}
	require.Zero(t, metadataCalls, "static recovery guidance must not resolve metadata")
	cause := errors.New("unrelated failure")
	rewritten, err := RewriteAuthError(t.Context(), "", "", "", cause)
	if rewritten {
		t.Fatal("unrelated error behavior changed")
	}
	require.Same(t, cause, err)
}
