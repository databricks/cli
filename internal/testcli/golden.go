package testcli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/elliotchance/orderedmap/v3"
	"github.com/stretchr/testify/require"
)

var OverwriteMode = os.Getenv("TESTS_OUTPUT") == "OVERWRITE"

func ReadFile(t testutil.TestingT, ctx context.Context, filename string) string {
	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		return ""
	}
	require.NoError(t, err)
	// On CI, on Windows \n in the file somehow end up as \r\n
	return NormalizeNewlines(string(data))
}

func captureOutput(t testutil.TestingT, ctx context.Context, args []string) string {
	t.Logf("run args: [%s]", strings.Join(args, ", "))
	r := NewRunner(t, ctx, args...)
	stdout, stderr, err := r.Run()
	require.NoError(t, err)
	out := stderr.String() + stdout.String()
	return ReplaceOutput(t, ctx, out)
}

func WriteFile(t testutil.TestingT, filename, data string) {
	t.Logf("Overwriting %s", filename)
	err := os.WriteFile(filename, []byte(data), 0o644)
	require.NoError(t, err)
}

func RequireOutput(t testutil.TestingT, ctx context.Context, args []string, expectedFilename string) {
	_, filename, _, _ := runtime.Caller(1)
	dir := filepath.Dir(filename)
	expectedPath := filepath.Join(dir, expectedFilename)
	expected := ReadFile(t, ctx, expectedPath)

	out := captureOutput(t, ctx, args)

	if out != expected {
		actual := fmt.Sprintf("Output from %v", args)
		testdiff.AssertEqualTexts(t, expectedFilename, actual, expected, out)

		if OverwriteMode {
			WriteFile(t, expectedPath, out)
		}
	}
}

func RequireOutputJQ(t testutil.TestingT, ctx context.Context, args []string, expectedFilename string, ignorePaths []string) {
	_, filename, _, _ := runtime.Caller(1)
	dir := filepath.Dir(filename)
	expectedPath := filepath.Join(dir, expectedFilename)
	expected := ReadFile(t, ctx, expectedPath)

	out := captureOutput(t, ctx, args)

	if out != expected {
		actual := fmt.Sprintf("Output from %v", args)
		testdiff.AssertEqualJQ(t.(*testing.T), expectedFilename, actual, expected, out, ignorePaths)

		if OverwriteMode {
			WriteFile(t, expectedPath, out)
		}
	}
}

var (
	uuidRegex        = regexp.MustCompile(`[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}`)
	numIdRegex       = regexp.MustCompile(`[0-9]{3,}`)
	privatePathRegex = regexp.MustCompile(`(/tmp|/private)(/.*)/([a-zA-Z0-9]+)`)
)

func ReplaceOutput(t testutil.TestingT, ctx context.Context, out string) string {
	out = NormalizeNewlines(out)
	replacements := GetReplacementsMap(ctx)
	if replacements == nil {
		t.Fatal("WithReplacementsMap was not called")
	}
	for key, value := range replacements.AllFromFront() {
		out = strings.ReplaceAll(out, key, value)
	}

	out = uuidRegex.ReplaceAllString(out, "<UUID>")
	out = numIdRegex.ReplaceAllString(out, "<NUMID>")
	out = privatePathRegex.ReplaceAllString(out, "/tmp/.../$3")

	return out
}

type key int

const (
	replacementsMapKey = key(1)
)

func WithReplacementsMap(ctx context.Context) (context.Context, *orderedmap.OrderedMap[string, string]) {
	value := ctx.Value(replacementsMapKey)
	if value != nil {
		if existingMap, ok := value.(*orderedmap.OrderedMap[string, string]); ok {
			return ctx, existingMap
		}
	}

	newMap := orderedmap.NewOrderedMap[string, string]()
	ctx = context.WithValue(ctx, replacementsMapKey, newMap)
	return ctx, newMap
}

func GetReplacementsMap(ctx context.Context) *orderedmap.OrderedMap[string, string] {
	value := ctx.Value(replacementsMapKey)
	if value != nil {
		if existingMap, ok := value.(*orderedmap.OrderedMap[string, string]); ok {
			return existingMap
		}
	}
	return nil
}

func setKV(replacements *orderedmap.OrderedMap[string, string], key, value string) {
	if key == "" || value == "" {
		return
	}
	replacements.Set(key, value)
}

func PrepareReplacements(t testutil.TestingT, replacements *orderedmap.OrderedMap[string, string], w *databricks.WorkspaceClient) {
	// in some clouds (gcp) w.Config.Host includes "https://" prefix in others it's really just a host (azure)
	host := strings.TrimPrefix(strings.TrimPrefix(w.Config.Host, "http://"), "https://")
	setKV(replacements, host, "$DATABRICKS_HOST")
	setKV(replacements, w.Config.ClusterID, "$DATABRICKS_CLUSTER_ID")
	setKV(replacements, w.Config.WarehouseID, "$DATABRICKS_WAREHOUSE_ID")
	setKV(replacements, w.Config.ServerlessComputeID, "$DATABRICKS_SERVERLESS_COMPUTE_ID")
	setKV(replacements, w.Config.MetadataServiceURL, "$DATABRICKS_METADATA_SERVICE_URL")
	setKV(replacements, w.Config.AccountID, "$DATABRICKS_ACCOUNT_ID")
	setKV(replacements, w.Config.Token, "$DATABRICKS_TOKEN")
	setKV(replacements, w.Config.Username, "$DATABRICKS_USERNAME")
	setKV(replacements, w.Config.Password, "$DATABRICKS_PASSWORD")
	setKV(replacements, w.Config.Profile, "$DATABRICKS_CONFIG_PROFILE")
	setKV(replacements, w.Config.ConfigFile, "$DATABRICKS_CONFIG_FILE")
	setKV(replacements, w.Config.GoogleServiceAccount, "$DATABRICKS_GOOGLE_SERVICE_ACCOUNT")
	setKV(replacements, w.Config.GoogleCredentials, "$GOOGLE_CREDENTIALS")
	setKV(replacements, w.Config.AzureResourceID, "$DATABRICKS_AZURE_RESOURCE_ID")
	setKV(replacements, w.Config.AzureClientSecret, "$ARM_CLIENT_SECRET")
	// setKV(replacements, w.Config.AzureClientID, "$ARM_CLIENT_ID")
	setKV(replacements, w.Config.AzureClientID, "$USERNAME")
	setKV(replacements, w.Config.AzureTenantID, "$ARM_TENANT_ID")
	setKV(replacements, w.Config.ActionsIDTokenRequestURL, "$ACTIONS_ID_TOKEN_REQUEST_URL")
	setKV(replacements, w.Config.ActionsIDTokenRequestToken, "$ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	setKV(replacements, w.Config.AzureEnvironment, "$ARM_ENVIRONMENT")
	setKV(replacements, w.Config.ClientID, "$DATABRICKS_CLIENT_ID")
	setKV(replacements, w.Config.ClientSecret, "$DATABRICKS_CLIENT_SECRET")
	setKV(replacements, w.Config.DatabricksCliPath, "$DATABRICKS_CLI_PATH")
	setKV(replacements, w.Config.AuthType, "$DATABRICKS_AUTH_TYPE")
}

func PrepareReplacementsUser(t testutil.TestingT, replacements *orderedmap.OrderedMap[string, string], u iam.User) {
	// There could be exact matches or overlap between different name fields, so sort them by length
	// to ensure we match the largest one first and map them all to the same token
	names := []string{
		u.DisplayName,
		u.UserName,
		iamutil.GetShortUserName(&u),
		u.Name.FamilyName,
		u.Name.GivenName,
	}
	if u.Name != nil {
		names = append(names, u.Name.FamilyName)
		names = append(names, u.Name.GivenName)
	}
	for _, val := range u.Emails {
		names = append(names, val.Value)
	}
	stableSortReverseLength(names)

	for _, name := range names {
		setKV(replacements, name, "$USERNAME")
	}

	for ind, val := range u.Groups {
		setKV(replacements, val.Value, fmt.Sprintf("$USER.Groups[%d]", ind))
	}

	setKV(replacements, u.Id, "$USER.Id")

	for ind, val := range u.Roles {
		setKV(replacements, val.Value, fmt.Sprintf("$USER.Roles[%d]", ind))
	}

	// Schemas []UserSchema `json:"schemas,omitempty"`
}

func stableSortReverseLength(strs []string) {
	slices.SortStableFunc(strs, func(a, b string) int {
		return len(b) - len(a)
	})
}

func NormalizeNewlines(input string) string {
	output := strings.ReplaceAll(input, "\r\n", "\n")
	return strings.ReplaceAll(output, "\r", "\n")
}
