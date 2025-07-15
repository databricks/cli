package testdiff

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"golang.org/x/mod/semver"
)

const (
	testerName = "[USERNAME]"
)

var (
	uuidRegex        = regexp.MustCompile(`[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}`)
	numIdRegex       = regexp.MustCompile(`[0-9]{3,}`)
	privatePathRegex = regexp.MustCompile(`(/tmp|/private)(/.*)/([a-zA-Z0-9]+)`)
	// Version could v0.0.0-dev+21e1aacf518a or just v0.0.0-dev (the latter is currently the case on Windows)
	devVersionRegex = regexp.MustCompile(`0\.0\.0-dev(\+[a-f0-9]{10,16})?`)
)

type Replacement struct {
	Old   *regexp.Regexp
	New   string
	Order int
}

type ReplacementsContext struct {
	Repls []Replacement
}

func (r *ReplacementsContext) Clone() ReplacementsContext {
	return ReplacementsContext{Repls: slices.Clone(r.Repls)}
}

func (r *ReplacementsContext) Replace(s string) string {
	// QQQ Should probably only replace whole words
	// Sort replacements stably by Order to guarantee deterministic application sequence.
	// A cloned slice is used to avoid mutating the original order held in the context.
	repls := slices.Clone(r.Repls)
	sort.SliceStable(repls, func(i, j int) bool {
		return repls[i].Order < repls[j].Order
	})
	for _, repl := range repls {
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
			encodedStrNew := trimQuotes(string(encodedNew))
			encodedStrOld := trimQuotes(string(encodedOld))
			if encodedStrNew != new || encodedStrOld != old {
				r.appendLiteral(encodedStrOld, encodedStrNew)
			}
		}
	}

	r.appendLiteral(old, new)
}

func trimQuotes(s string) string {
	if len(s) > 0 && s[0] == '"' {
		s = s[1:]
	}
	if len(s) > 0 && s[len(s)-1] == '"' {
		s = s[:len(s)-1]
	}
	return s
}

func (r *ReplacementsContext) SetPath(old, new string) {
	if old != "" && old != "." {
		// Converts C:\Users\DENIS~1.BIL -> C:\Users\denis.bilenko
		oldEvalled, err1 := filepath.EvalSymlinks(old)
		if err1 == nil && oldEvalled != old {
			r.SetPathNoEval(oldEvalled, new)
		}
	}

	r.SetPathNoEval(old, new)
}

func (r *ReplacementsContext) SetPathNoEval(old, new string) {
	r.Set(old, new)

	if runtime.GOOS != "windows" {
		return
	}

	// Support both forward and backward slashes
	m1 := strings.ReplaceAll(old, "\\", "/")
	if m1 != old {
		r.Set(m1, new)
	}

	m2 := strings.ReplaceAll(old, "/", "\\")
	if m2 != old && m2 != m1 {
		r.Set(m2, new)
	}
}

func (r *ReplacementsContext) SetPathWithParents(old, new string) {
	r.SetPath(old, new)
	r.SetPath(filepath.Dir(old), new+"_PARENT")
}

func PrepareReplacementsWorkspaceConfig(t testutil.TestingT, r *ReplacementsContext, cfg *config.Config) {
	t.Helper()
	// in some clouds (gcp) w.Config.Host includes "https://" prefix in others it's really just a host (azure)
	host := strings.TrimPrefix(strings.TrimPrefix(cfg.Host, "http://"), "https://")
	r.Set("https://"+host, "[DATABRICKS_URL]")
	r.Set("http://"+host, "[DATABRICKS_URL]")
	r.Set(host, "[DATABRICKS_HOST]")
	r.Set(cfg.ClusterID, "[DATABRICKS_CLUSTER_ID]")
	r.Set(cfg.WarehouseID, "[DATABRICKS_WAREHOUSE_ID]")
	r.Set(cfg.ServerlessComputeID, "[DATABRICKS_SERVERLESS_COMPUTE_ID]")
	r.Set(cfg.AccountID, "[DATABRICKS_ACCOUNT_ID]")
	r.Set(cfg.Username, "[DATABRICKS_USERNAME]")
	r.SetPath(cfg.Profile, "[DATABRICKS_CONFIG_PROFILE]")
	r.Set(cfg.ConfigFile, "[DATABRICKS_CONFIG_FILE]")
	r.Set(cfg.GoogleServiceAccount, "[DATABRICKS_GOOGLE_SERVICE_ACCOUNT]")
	r.Set(cfg.AzureResourceID, "[DATABRICKS_AZURE_RESOURCE_ID]")
	r.Set(cfg.AzureClientID, testerName)
	r.Set(cfg.AzureTenantID, "[ARM_TENANT_ID]")
	r.Set(cfg.AzureEnvironment, "[ARM_ENVIRONMENT]")
	r.Set(cfg.ClientID, "[DATABRICKS_CLIENT_ID]")
	r.SetPath(cfg.DatabricksCliPath, "[DATABRICKS_CLI_PATH]")
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

	for _, val := range u.Groups {
		r.Set(val.Value, "[USERGROUP]")
	}

	r.Set(u.Id, "[USERID]")

	for _, val := range u.Roles {
		r.Set(val.Value, "[USERROLE]")
	}
}

func PrepareReplacementsUUID(t testutil.TestingT, r *ReplacementsContext) {
	t.Helper()
	r.append(uuidRegex, "[UUID]")
}

func PrepareReplacementsNumber(t testutil.TestingT, r *ReplacementsContext) {
	t.Helper()
	r.append(numIdRegex, "[NUMID]")
}

func PrepareReplacementsTemporaryDirectory(t testutil.TestingT, r *ReplacementsContext) {
	t.Helper()
	r.append(privatePathRegex, "/tmp/.../$3")
}

func PrepareReplacementsDevVersion(t testutil.TestingT, r *ReplacementsContext) {
	t.Helper()
	r.append(devVersionRegex, "[DEV_VERSION]")
}

func PrepareReplacementSdkVersion(t testutil.TestingT, r *ReplacementsContext) {
	t.Helper()
	r.Set(databricks.Version(), "[SDK_VERSION]")
}

func goVersion() string {
	gv := runtime.Version()
	ssv := strings.ReplaceAll(gv, "go", "v")
	sv := semver.Canonical(ssv)
	return strings.TrimPrefix(sv, "v")
}

func PrepareReplacementsGoVersion(t testutil.TestingT, r *ReplacementsContext) {
	t.Helper()
	r.Set(goVersion(), "[GO_VERSION]")
}
