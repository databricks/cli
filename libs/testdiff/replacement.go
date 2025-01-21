package testdiff

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

const (
	testerName = "$USERNAME"
)

var (
	uuidRegex        = regexp.MustCompile(`[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}`)
	numIdRegex       = regexp.MustCompile(`[0-9]{3,}`)
	privatePathRegex = regexp.MustCompile(`(/tmp|/private)(/.*)/([a-zA-Z0-9]+)`)
)

type Replacement struct {
	Old *regexp.Regexp
	New string
}

type ReplacementsContext struct {
	Repls []Replacement
}

func (r *ReplacementsContext) Clone() ReplacementsContext {
	return ReplacementsContext{Repls: slices.Clone(r.Repls)}
}

func (r *ReplacementsContext) Replace(s string) string {
	// QQQ Should probably only replace whole words
	for _, repl := range r.Repls {
		s = repl.Old.ReplaceAllString(s, repl.New)
	}
	return s
}

func (r *ReplacementsContext) append(pattern *regexp.Regexp, replacement string) {
	r.Repls = append(r.Repls, Replacement{
		Old: pattern,
		New: replacement,
	})
}

func (r *ReplacementsContext) appendLiteral(old, new string) {
	r.append(
		// Transform the input strings such that they can be used as literal strings in regular expressions.
		regexp.MustCompile(regexp.QuoteMeta(old)),
		// Transform the replacement string such that `$` is interpreted as a literal dollar sign.
		// For more information about how the replacement string is used, see [regexp.Regexp.Expand].
		strings.ReplaceAll(new, `$`, `$$`),
	)
}

func (r *ReplacementsContext) Set(old, new string) {
	if old == "" || new == "" {
		return
	}

	// Always include both verbatim and json version of replacement.
	// This helps when the string in question contains \ or other chars that need to be quoted.
	// In that case we cannot rely that json(old) == '"{old}"' and need to add it explicitly.

	encodedNew, err := json.Marshal(new)
	if err == nil {
		encodedOld, err := json.Marshal(old)
		if err == nil {
			r.appendLiteral(string(encodedOld), string(encodedNew))
		}
	}

	r.appendLiteral(old, new)
}

func PrepareReplacementsWorkspaceClient(t testutil.TestingT, r *ReplacementsContext, w *databricks.WorkspaceClient) {
	t.Helper()
	// in some clouds (gcp) w.Config.Host includes "https://" prefix in others it's really just a host (azure)
	host := strings.TrimPrefix(strings.TrimPrefix(w.Config.Host, "http://"), "https://")
	r.Set("https://"+host, "$DATABRICKS_URL")
	r.Set("http://"+host, "$DATABRICKS_URL")
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
	r.Set(w.Config.AzureClientID, testerName)
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
	t.Helper()
	// There could be exact matches or overlap between different name fields, so sort them by length
	// to ensure we match the largest one first and map them all to the same token

	r.Set(u.UserName, testerName)
	r.Set(u.DisplayName, testerName)
	if u.Name != nil {
		r.Set(u.Name.FamilyName, testerName)
		r.Set(u.Name.GivenName, testerName)
	}

	for _, val := range u.Emails {
		r.Set(val.Value, testerName)
	}

	r.Set(iamutil.GetShortUserName(&u), testerName)

	for ind, val := range u.Groups {
		r.Set(val.Value, fmt.Sprintf("$USER.Groups[%d]", ind))
	}

	r.Set(u.Id, "$USER.Id")

	for ind, val := range u.Roles {
		r.Set(val.Value, fmt.Sprintf("$USER.Roles[%d]", ind))
	}
}

func PrepareReplacementsUUID(t testutil.TestingT, r *ReplacementsContext) {
	t.Helper()
	r.append(uuidRegex, "<UUID>")
}

func PrepareReplacementsNumber(t testutil.TestingT, r *ReplacementsContext) {
	t.Helper()
	r.append(numIdRegex, "<NUMID>")
}

func PrepareReplacementsTemporaryDirectory(t testutil.TestingT, r *ReplacementsContext) {
	t.Helper()
	r.append(privatePathRegex, "/tmp/.../$3")
}
