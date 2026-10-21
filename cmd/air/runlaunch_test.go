package aircmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	sdkauth "github.com/databricks/databricks-sdk-go/config/experimental/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestCliLaunchDir(t *testing.T) {
	dir := cliLaunchDir("/Workspace/Users/me@example.com", "my-exp", "")
	assert.True(t, strings.HasPrefix(dir, "/Workspace/Users/me@example.com/.air/cli_launch/my-exp/my-exp_"), dir)
	// run name overrides the leaf; the unique suffix keeps successive dirs distinct.
	withRun := cliLaunchDir("/base", "exp", "run1")
	assert.True(t, strings.HasPrefix(withRun, "/base/.air/cli_launch/exp/run1_"), withRun)
	assert.NotEqual(t, dir, cliLaunchDir("/Workspace/Users/me@example.com", "my-exp", ""))
}

func newFakeWorkspaceClient(t *testing.T) *databricks.WorkspaceClient {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	return w
}

func TestUserWorkspaceDir(t *testing.T) {
	w := newFakeWorkspaceClient(t)
	t.Setenv(userWorkspaceDirEnv, "")
	dir, err := userWorkspaceDir(t.Context(), w)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(dir, "/Workspace/Users/"), dir)

	// The env override wins without an API call.
	t.Setenv(userWorkspaceDirEnv, "/Workspace/custom")
	dir, err = userWorkspaceDir(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "/Workspace/custom", dir)
}

func fakeJWT(t *testing.T, issuer, subject string) string {
	payload, err := json.Marshal(map[string]string{"iss": issuer, "sub": subject})
	require.NoError(t, err)
	enc := base64.RawURLEncoding
	return enc.EncodeToString([]byte(`{"alg":"none"}`)) + "." + enc.EncodeToString(payload) + ".sig"
}

func oauthConfig(host, authType, accessToken string) *databricks.Config {
	ts := sdkauth.TokenSourceFn(func(context.Context) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: accessToken}, nil
	})
	return &databricks.Config{Host: host, Credentials: config.NewTokenSourceStrategy(authType, ts)}
}

// The token's user is only a guess that lets validation start early: userWorkspaceDir decides.
func TestProspectiveLaunchPathsTokenGuess(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	var attributes []string
	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(req testserver.Request) any {
		attributes = append(attributes, req.URL.Query().Get("attributes"))
		return testserver.TestUser
	})
	testserver.AddDefaultHandlers(server)
	issuer := server.URL + "/oidc"
	scimDir := "/Workspace/Users/" + testserver.TestUser.UserName
	otherDir := "/Workspace/Users/other@example.com"
	otherToken := fakeJWT(t, issuer, "other@example.com")

	for _, tc := range []struct {
		name      string
		cfg       *databricks.Config
		override  string
		validated []string
	}{
		{"token names the SCIM user", oauthConfig(server.URL, auth.AuthTypeDatabricksCli, fakeJWT(t, issuer, testserver.TestUser.UserName)), "", []string{scimDir}},
		{"token names another user", oauthConfig(server.URL, auth.AuthTypeOAuthM2M, otherToken), "", []string{otherDir, scimDir}},
		{"token from another issuer", oauthConfig(server.URL, auth.AuthTypeDatabricksCli, fakeJWT(t, "https://accounts.test/oidc", "other@example.com")), "", []string{scimDir}},
		// Other auth types may not cache Token, so they skip the guess.
		{"other auth type", oauthConfig(server.URL, auth.AuthTypeAzureCli, otherToken), "", []string{scimDir}},
		{"pat", &databricks.Config{Host: server.URL, Token: otherToken}, "", []string{scimDir}},
		{"override", oauthConfig(server.URL, auth.AuthTypeDatabricksCli, otherToken), "/Workspace/custom", []string{"/Workspace/custom"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(userWorkspaceDirEnv, tc.override)
			w, err := databricks.NewWorkspaceClient(tc.cfg)
			require.NoError(t, err)

			var launchDirs []string
			base, launchDir, commandPath, err := prospectiveLaunchPaths(t.Context(), w, baseRunConfig(), func(launchDir, _ string) error {
				launchDirs = append(launchDirs, launchDir)
				if strings.HasPrefix(launchDir, otherDir+"/") {
					return errors.New("validated an unconfirmed user")
				}
				return nil
			})
			require.NoError(t, err)
			require.Len(t, launchDirs, len(tc.validated))
			for i, dir := range tc.validated {
				assert.True(t, strings.HasPrefix(launchDirs[i], dir+"/.air/cli_launch/"), launchDirs[i])
			}
			assert.Equal(t, tc.validated[len(tc.validated)-1], base)
			assert.Equal(t, launchDirs[len(launchDirs)-1], launchDir)
			assert.Equal(t, launchDir+"/command.sh", commandPath)
		})
	}
	require.NotEmpty(t, attributes)
	for _, a := range attributes {
		assert.Equal(t, "userName", a)
	}
}

func TestUserWorkspaceDirProfilePrecedence(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(configPath, []byte(`
[__settings__]
default_profile = second

[first]
host = https://first.test
databricks_internal_user_workspace_dir = /Workspace/first

[second]
host = https://second.test
databricks_internal_user_workspace_dir = /Workspace/second
`), 0o600))

	t.Run("explicit profile", func(t *testing.T) {
		w := newFakeWorkspaceClient(t)
		w.Config.ConfigFile = configPath
		w.Config.Profile = "first"
		t.Setenv(userWorkspaceDirEnv, "")
		dir, err := userWorkspaceDir(t.Context(), w)
		require.NoError(t, err)
		assert.Equal(t, "/Workspace/first", dir)
	})

	t.Run("default profile", func(t *testing.T) {
		w := newFakeWorkspaceClient(t)
		w.Config.ConfigFile = configPath
		w.Config.Profile = ""
		t.Setenv(userWorkspaceDirEnv, "")
		dir, err := userWorkspaceDir(t.Context(), w)
		require.NoError(t, err)
		assert.Equal(t, "/Workspace/second", dir)
	})

	t.Run("environment wins", func(t *testing.T) {
		w := newFakeWorkspaceClient(t)
		w.Config.ConfigFile = configPath
		w.Config.Profile = "first"
		t.Setenv(userWorkspaceDirEnv, "/Workspace/env")
		dir, err := userWorkspaceDir(t.Context(), w)
		require.NoError(t, err)
		assert.Equal(t, "/Workspace/env", dir)
	})
}

func TestEnsureExperimentDirectory(t *testing.T) {
	ctx := t.Context()
	w := newFakeWorkspaceClient(t)

	// Empty means default (always exists) — no API call, no error.
	require.NoError(t, ensureExperimentDirectory(ctx, w, ""))

	// A missing path is created.
	require.NoError(t, ensureExperimentDirectory(ctx, w, "/Workspace/Users/me/exp"))

	// An existing directory is accepted as-is.
	require.NoError(t, w.Workspace.MkdirsByPath(ctx, "/Workspace/Users/me/existing"))
	require.NoError(t, ensureExperimentDirectory(ctx, w, "/Workspace/Users/me/existing"))

	// A path that exists but is a file is rejected.
	fc, err := filer.NewWorkspaceFilesClient(w, "/Workspace/Users/me")
	require.NoError(t, err)
	require.NoError(t, fc.Write(ctx, "afile", strings.NewReader("x")))
	err = ensureExperimentDirectory(ctx, w, "/Workspace/Users/me/afile")
	require.ErrorContains(t, err, "is not a directory")
}
