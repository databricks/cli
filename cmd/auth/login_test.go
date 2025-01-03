package auth

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/credentials/oauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetHostDoesNotFailWithNoDatabrickscfg(t *testing.T) {
	ctx := context.Background()
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", "./imaginary-file/databrickscfg")
	_, err := setHostAndAccountId(ctx, profile.DefaultProfiler, "foo", oauth.BasicOAuthArgument{Host: "test"}, []string{})
	assert.NoError(t, err)
}

func TestSetHost(t *testing.T) {
	var persistentAuth oauth.BasicOAuthArgument
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")
	ctx, _ := cmdio.SetupTest(context.Background())

	// Test error when both flag and argument are provided
	persistentAuth.Host = "val from --host"
	_, err := setHostAndAccountId(ctx, profile.DefaultProfiler, "profile-1", &persistentAuth, []string{"val from [HOST]"})
	assert.EqualError(t, err, "please only provide a host as an argument or a flag, not both")

	// Test setting host from flag
	persistentAuth.Host = "val from --host"
	res, err := setHostAndAccountId(ctx, profile.DefaultProfiler, "profile-1", &persistentAuth, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "val from --host", res.GetHost(ctx))

	// Test setting host from argument
	persistentAuth.Host = ""
	res, err = setHostAndAccountId(ctx, profile.DefaultProfiler, "profile-1", &persistentAuth, []string{"val from [HOST]"})
	assert.NoError(t, err)
	assert.Equal(t, "val from [HOST]", res.GetHost(ctx))

	// Test setting host from profile
	persistentAuth.Host = ""
	res, err = setHostAndAccountId(ctx, profile.DefaultProfiler, "profile-1", &persistentAuth, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.host1.com", res.GetHost(ctx))

	// Test setting host from profile
	persistentAuth.Host = ""
	res, err = setHostAndAccountId(ctx, profile.DefaultProfiler, "profile-2", &persistentAuth, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.host2.com", res.GetHost(ctx))

	// Test host is not set. Should prompt.
	persistentAuth.Host = ""
	_, err = setHostAndAccountId(ctx, profile.DefaultProfiler, "", &persistentAuth, []string{})
	assert.EqualError(t, err, "the command is being run in a non-interactive environment, please specify a host using --host")
}

func TestSetAccountId(t *testing.T) {
	var persistentAuth oauth.BasicOAuthArgument
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")
	ctx, _ := cmdio.SetupTest(context.Background())

	// Test setting account-id from flag
	persistentAuth.AccountID = "val from --account-id"
	res, err := setHostAndAccountId(ctx, profile.DefaultProfiler, "account-profile", &persistentAuth, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://accounts.cloud.databricks.com", res.GetHost(ctx))
	assert.Equal(t, "val from --account-id", res.GetAccountId(ctx))

	// Test setting account_id from profile
	persistentAuth.AccountID = ""
	res, err = setHostAndAccountId(ctx, profile.DefaultProfiler, "account-profile", &persistentAuth, []string{})
	require.NoError(t, err)
	assert.Equal(t, "https://accounts.cloud.databricks.com", res.GetHost(ctx))
	assert.Equal(t, "id-from-profile", res.GetAccountId(ctx))

	// Neither flag nor profile account-id is set, should prompt
	persistentAuth.AccountID = ""
	persistentAuth.Host = "https://accounts.cloud.databricks.com"
	_, err = setHostAndAccountId(ctx, profile.DefaultProfiler, "", &persistentAuth, []string{})
	assert.EqualError(t, err, "the command is being run in a non-interactive environment, please specify an account ID using --account-id")
}
