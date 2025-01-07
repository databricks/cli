package auth

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetHostDoesNotFailWithNoDatabrickscfg(t *testing.T) {
	ctx := context.Background()
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", "./imaginary-file/databrickscfg")
	err := setHostAndAccountId(ctx, profile.DefaultProfiler, "foo", &authArguments{host: "test"}, []string{})
	assert.NoError(t, err)
}

func TestSetHost(t *testing.T) {
	authArguments := authArguments{}
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")
	ctx, _ := cmdio.SetupTest(context.Background())

	// Test error when both flag and argument are provided
	authArguments.host = "val from --host"
	err := setHostAndAccountId(ctx, profile.DefaultProfiler, "profile-1", &authArguments, []string{"val from [HOST]"})
	assert.EqualError(t, err, "please only provide a host as an argument or a flag, not both")

	// Test setting host from flag
	authArguments.host = "val from --host"
	err = setHostAndAccountId(ctx, profile.DefaultProfiler, "profile-1", &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "val from --host", authArguments.host)

	// Test setting host from argument
	authArguments.host = ""
	err = setHostAndAccountId(ctx, profile.DefaultProfiler, "profile-1", &authArguments, []string{"val from [HOST]"})
	assert.NoError(t, err)
	assert.Equal(t, "val from [HOST]", authArguments.host)

	// Test setting host from profile
	authArguments.host = ""
	err = setHostAndAccountId(ctx, profile.DefaultProfiler, "profile-1", &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.host1.com", authArguments.host)

	// Test setting host from profile
	authArguments.host = ""
	err = setHostAndAccountId(ctx, profile.DefaultProfiler, "profile-2", &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.host2.com", authArguments.host)

	// Test host is not set. Should prompt.
	authArguments.host = ""
	err = setHostAndAccountId(ctx, profile.DefaultProfiler, "", &authArguments, []string{})
	assert.EqualError(t, err, "the command is being run in a non-interactive environment, please specify a host using --host")
}

func TestSetAccountId(t *testing.T) {
	var authArguments authArguments
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")
	ctx, _ := cmdio.SetupTest(context.Background())

	// Test setting account-id from flag
	authArguments.accountId = "val from --account-id"
	err := setHostAndAccountId(ctx, profile.DefaultProfiler, "account-profile", &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://accounts.cloud.databricks.com", authArguments.host)
	assert.Equal(t, "val from --account-id", authArguments.accountId)

	// Test setting account_id from profile
	authArguments.accountId = ""
	err = setHostAndAccountId(ctx, profile.DefaultProfiler, "account-profile", &authArguments, []string{})
	require.NoError(t, err)
	assert.Equal(t, "https://accounts.cloud.databricks.com", authArguments.host)
	assert.Equal(t, "id-from-profile", authArguments.accountId)

	// Neither flag nor profile account-id is set, should prompt
	authArguments.accountId = ""
	authArguments.host = "https://accounts.cloud.databricks.com"
	err = setHostAndAccountId(ctx, profile.DefaultProfiler, "", &authArguments, []string{})
	assert.EqualError(t, err, "the command is being run in a non-interactive environment, please specify an account ID using --account-id")
}
