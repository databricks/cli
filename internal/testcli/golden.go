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

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/elliotchance/orderedmap/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wI2L/jsondiff"
)

func ReadFile(t testutil.TestingT, ctx context.Context, filename string) string {
	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		return ""
	}
	require.NoError(t, err)
	return string(data)
}

func WriteFile(t testutil.TestingT, ctx context.Context, filename, data string) {
	t.Logf("Overwriting %s", filename)
	err := os.WriteFile(filename, []byte(data), 0o644)
	require.NoError(t, err)
}

func captureOutput(t testutil.TestingT, ctx context.Context, args []string) string {
	t.Logf("run args: [%s]", strings.Join(args, ", "))
	r := NewRunner(t, ctx, args...)
	stdout, stderr, err := r.Run()
	require.NoError(t, err)
	out := stderr.String() + stdout.String()
	return ReplaceOutput(t, ctx, out)
}

func assertEqualTexts(t testutil.TestingT, filename1, filename2, expected, out string) {
	if len(out) < 1000 && len(expected) < 1000 {
		// This shows full strings + diff which could be useful when debugging newlines
		assert.Equal(t, expected, out)
	} else {
		// only show diff for large texts
		diff := testutil.Diff(filename1, filename2, expected, out)
		t.Errorf("Diff:\n" + diff)
	}
}

func logDiff(t testutil.TestingT, filename1, filename2, expected, out string) {
	diff := testutil.Diff(filename1, filename2, expected, out)
	t.Logf("Diff:\n" + diff)
}

func RequireOutput(t testutil.TestingT, ctx context.Context, args []string, expectedFilename string) {
	_, filename, _, _ := runtime.Caller(1)
	dir := filepath.Dir(filename)
	expectedPath := filepath.Join(dir, expectedFilename)
	expected := ReadFile(t, ctx, expectedPath)

	out := captureOutput(t, ctx, args)

	if out != expected {
		actual := fmt.Sprintf("Output from %v", args)
		assertEqualTexts(t, expectedFilename, actual, expected, out)

		if os.Getenv("TESTS_OUTPUT") == "OVERWRITE" {
			WriteFile(t, ctx, expectedPath, out)
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
		patch, err := jsondiff.CompareJSON([]byte(expected), []byte(out))
		actual := fmt.Sprintf("Output from %v", args)
		if err != nil {
			t.Logf("CompareJSON error for %s vs %s: %s (fallback to textual comparison)", args, expectedFilename, err)
			assertEqualTexts(t, expectedFilename, actual, expected, out)
		} else {
			logDiff(t, expectedFilename, actual, expected, out)
			ignoredDiffs := []string{}
			erroredDiffs := []string{}
			for _, op := range patch {
				if matchesPrefixes(ignorePaths, op.Path) {
					ignoredDiffs = append(ignoredDiffs, fmt.Sprintf("%7s %s %v", op.Type, op.Path, op.Value))
				} else {
					erroredDiffs = append(erroredDiffs, fmt.Sprintf("%7s %s %v", op.Type, op.Path, op.Value))
				}
			}
			if len(ignoredDiffs) > 0 {
				t.Logf("Ignored differences between %s and %s:\n ==> %s", expectedFilename, args, strings.Join(ignoredDiffs, "\n ==> "))
			}
			if len(erroredDiffs) > 0 {
				t.Errorf("Unexpected differences between %s and %s:\n ==> %s", expectedFilename, args, strings.Join(erroredDiffs, "\n ==> "))
			}
		}

		if os.Getenv("TESTS_OUTPUT") == "OVERWRITE" {
			WriteFile(t, ctx, filepath.Join(dir, expectedFilename), out)
		}
	}
}

func matchesPrefixes(prefixes []string, path string) bool {
	for _, p := range prefixes {
		if p == path {
			return true
		}
		if strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
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
