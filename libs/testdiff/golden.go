package testdiff

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
)

var OverwriteMode = os.Getenv("TESTS_OUTPUT") == "OVERWRITE"

func ReadFile(t testutil.TestingT, ctx context.Context, filename string) string {
	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		return ""
	}
	assert.NoError(t, err)
	// On CI, on Windows \n in the file somehow end up as \r\n
	return NormalizeNewlines(string(data))
}

func WriteFile(t testutil.TestingT, filename, data string) {
	t.Logf("Overwriting %s", filename)
	err := os.WriteFile(filename, []byte(data), 0o644)
	assert.NoError(t, err)
}

func AssertOutput(t testutil.TestingT, ctx context.Context, out, outTitle, expectedPath string) {
	expected := ReadFile(t, ctx, expectedPath)

	out = ReplaceOutput(t, ctx, out)

	if out != expected {
		AssertEqualTexts(t, expectedPath, outTitle, expected, out)

		if OverwriteMode {
			WriteFile(t, expectedPath, out)
		}
	}
}

func AssertOutputJQ(t testutil.TestingT, ctx context.Context, out, outTitle, expectedPath string, ignorePaths []string) {
	expected := ReadFile(t, ctx, expectedPath)

	out = ReplaceOutput(t, ctx, out)

	if out != expected {
		AssertEqualJQ(t.(*testing.T), expectedPath, outTitle, expected, out, ignorePaths)

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
	out = replacements.Replace(out)
	out = uuidRegex.ReplaceAllString(out, "<UUID>")
	out = numIdRegex.ReplaceAllString(out, "<NUMID>")
	out = privatePathRegex.ReplaceAllString(out, "/tmp/.../$3")

	return out
}

type key int

const (
	replacementsMapKey = key(1)
)

type Replacement struct {
	Old string
	New string
}

type ReplacementsContext struct {
	Repls []Replacement
}

func (r *ReplacementsContext) Replace(s string) string {
	// QQQ Should probably only replace whole words
	for _, repl := range r.Repls {
		s = strings.ReplaceAll(s, repl.Old, repl.New)
	}
	return s
}

func (r *ReplacementsContext) Set(old, new string) {
	if old == "" || new == "" {
		return
	}
	r.Repls = append(r.Repls, Replacement{Old: old, New: new})
}

func WithReplacementsMap(ctx context.Context) (context.Context, *ReplacementsContext) {
	value := ctx.Value(replacementsMapKey)
	if value != nil {
		if existingMap, ok := value.(*ReplacementsContext); ok {
			return ctx, existingMap
		}
	}

	newMap := &ReplacementsContext{}
	ctx = context.WithValue(ctx, replacementsMapKey, newMap)
	return ctx, newMap
}

func GetReplacementsMap(ctx context.Context) *ReplacementsContext {
	value := ctx.Value(replacementsMapKey)
	if value != nil {
		if existingMap, ok := value.(*ReplacementsContext); ok {
			return existingMap
		}
	}
	return nil
}

func PrepareReplacements(t testutil.TestingT, r *ReplacementsContext, w *databricks.WorkspaceClient) {
	// in some clouds (gcp) w.Config.Host includes "https://" prefix in others it's really just a host (azure)
	host := strings.TrimPrefix(strings.TrimPrefix(w.Config.Host, "http://"), "https://")
	r.Set(host, "$DATABRICKS_HOST")
	r.Set(w.Config.ClusterID, "$DATABRICKS_CLUSTER_ID")
	r.Set(w.Config.WarehouseID, "$DATABRICKS_WAREHOUSE_ID")
	r.Set(w.Config.ServerlessComputeID, "$DATABRICKS_SERVERLESS_COMPUTE_ID")
	r.Set(w.Config.MetadataServiceURL, "$DATABRICKS_METADATA_SERVICE_URL")
	r.Set(w.Config.AccountID, "$DATABRICKS_ACCOUNT_ID")
	r.Set(w.Config.Token, "$DATABRICKS_TOKEN")
	r.Set(w.Config.Username, "$DATABRICKS_USERNAME")
	r.Set(w.Config.Password, "$DATABRICKS_PASSWORD")
	r.Set(w.Config.Profile, "$DATABRICKS_CONFIG_PROFILE")
	r.Set(w.Config.ConfigFile, "$DATABRICKS_CONFIG_FILE")
	r.Set(w.Config.GoogleServiceAccount, "$DATABRICKS_GOOGLE_SERVICE_ACCOUNT")
	r.Set(w.Config.GoogleCredentials, "$GOOGLE_CREDENTIALS")
	r.Set(w.Config.AzureResourceID, "$DATABRICKS_AZURE_RESOURCE_ID")
	r.Set(w.Config.AzureClientSecret, "$ARM_CLIENT_SECRET")
	// r.Set(w.Config.AzureClientID, "$ARM_CLIENT_ID")
	r.Set(w.Config.AzureClientID, "$USERNAME")
	r.Set(w.Config.AzureTenantID, "$ARM_TENANT_ID")
	r.Set(w.Config.ActionsIDTokenRequestURL, "$ACTIONS_ID_TOKEN_REQUEST_URL")
	r.Set(w.Config.ActionsIDTokenRequestToken, "$ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	r.Set(w.Config.AzureEnvironment, "$ARM_ENVIRONMENT")
	r.Set(w.Config.ClientID, "$DATABRICKS_CLIENT_ID")
	r.Set(w.Config.ClientSecret, "$DATABRICKS_CLIENT_SECRET")
	r.Set(w.Config.DatabricksCliPath, "$DATABRICKS_CLI_PATH")
	// This is set to words like "path" that happen too frequently
	// r.Set(w.Config.AuthType, "$DATABRICKS_AUTH_TYPE")
}

func PrepareReplacementsUser(t testutil.TestingT, r *ReplacementsContext, u iam.User) {
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
		r.Set(name, "$USERNAME")
	}

	for ind, val := range u.Groups {
		r.Set(val.Value, fmt.Sprintf("$USER.Groups[%d]", ind))
	}

	r.Set(u.Id, "$USER.Id")

	for ind, val := range u.Roles {
		r.Set(val.Value, fmt.Sprintf("$USER.Roles[%d]", ind))
	}
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
