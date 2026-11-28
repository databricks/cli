package testdiff

import (
	"cmp"
	"encoding/json"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"golang.org/x/mod/semver"
)

const (
	testerName = "[USERNAME]"
)

var (
	uuidRegex = regexp.MustCompile(`[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}`)
	// Matches databricks-sdk-go/0.90.0
	sdkVersionRegex = regexp.MustCompile(`databricks-sdk-go/[0-9]+\.[0-9]+\.[0-9]+`)
	// Matches databricks-tf-provider/1.121.0
	tfProviderVersionRegex = regexp.MustCompile(`databricks-tf-provider/[0-9]+\.[0-9]+\.[0-9]+`)
)

type Replacement struct {
	Old      *regexp.Regexp
	New      string
	Order    int
	Distinct bool
	// WholeWord is set for the literal values that scripts add with add_repl. Their matches
	// must not cut into a longer alphanumeric run, see replaceWholeWord.
	WholeWord bool
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
	slices.SortStableFunc(repls, func(a, b Replacement) int {
		return cmp.Compare(a.Order, b.Order)
	})
	for _, repl := range repls {
		if repl.WholeWord {
			s = replaceWholeWord(repl.Old, repl.New, s)
			continue
		}
		if !repl.Distinct {
			s = repl.Old.ReplaceAllString(s, repl.New)
			continue
		}

		// Assign a stable [N] suffix per distinct matched value, in order of first
		// appearance. Number only when there is more than one distinct value: a lone
		// value (even if repeated) stays unsuffixed, so the common single-match output
		// is identical to the non-distinct case.
		seen := make(map[string]int)
		for _, match := range repl.Old.FindAllString(s, -1) {
			if _, ok := seen[match]; !ok {
				seen[match] = len(seen)
			}
		}
		if len(seen) <= 1 {
			s = repl.Old.ReplaceAllString(s, repl.New)
			continue
		}

		s = repl.Old.ReplaceAllStringFunc(s, func(match string) string {
			submatches := repl.Old.FindStringSubmatchIndex(match)
			replaced := repl.Old.ExpandString(nil, repl.New, match, submatches)
			return string(replaced) + "[" + strconv.Itoa(seen[match]) + "]"
		})
	}
	return s
}

func (r *ReplacementsContext) append(pattern *regexp.Regexp, replacement string, order int) {
	r.Repls = append(r.Repls, Replacement{
		Old:   pattern,
		New:   replacement,
		Order: order,
	})
}

// replaceWholeWord replaces the matches of the literal pattern re in s, except for matches that
// cut into a longer alphanumeric run. A match whose first byte is an ASCII letter or digit must not
// be preceded by one, and a match whose last byte is one must not be followed by one. Other bytes,
// including "_", are separators, and a value that starts or ends with anything else has no check
// on that side. The neighbours are read from the original s, so matches separated by a single
// character are all replaced.
func replaceWholeWord(re *regexp.Regexp, repl, s string) string {
	var b strings.Builder
	copied := 0 // s[:copied] is already in b
	for pos := 0; pos < len(s); {
		loc := re.FindStringIndex(s[pos:])
		if loc == nil {
			break
		}
		start, end := pos+loc[0], pos+loc[1]
		leftOK := !isASCIIAlnum(s[start]) || start == 0 || !isASCIIAlnum(s[start-1])
		rightOK := !isASCIIAlnum(s[end-1]) || end == len(s) || !isASCIIAlnum(s[end])
		if !leftOK || !rightOK {
			// Retry one byte later: a valid match may start inside the rejected one.
			pos = start + 1
			continue
		}
		b.WriteString(s[copied:start])
		b.Write(re.ExpandString(nil, repl, s[pos:], loc))
		copied = end
		pos = end
	}
	b.WriteString(s[copied:])
	return b.String()
}

func isASCIIAlnum(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

func (r *ReplacementsContext) appendLiteral(old, new string, order int, wholeWord bool) {
	r.Repls = append(r.Repls, Replacement{
		// Transform the input strings such that they can be used as literal strings in regular expressions.
		Old: regexp.MustCompile(regexp.QuoteMeta(old)),
		// Transform the replacement string such that `$` is interpreted as a literal dollar sign.
		// For more information about how the replacement string is used, see [regexp.Regexp.Expand].
		New:       strings.ReplaceAll(new, `$`, `$$`),
		Order:     order,
		WholeWord: wholeWord,
	})
}

func (r *ReplacementsContext) Set(old, new string) {
	r.SetWithOrder(old, new, 0)
}

func (r *ReplacementsContext) SetWithOrder(old, new string, order int) {
	r.setLiteral(old, new, order, false)
}

// SetWholeWordWithOrder is SetWithOrder for a value that only matches as a whole alphanumeric
// run, see replaceWholeWord. Used for the literals that scripts add with add_repl.
func (r *ReplacementsContext) SetWholeWordWithOrder(old, new string, order int) {
	r.setLiteral(old, new, order, true)
}

func (r *ReplacementsContext) setLiteral(old, new string, order int, wholeWord bool) {
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
				r.appendLiteral(encodedStrOld, encodedStrNew, order, wholeWord)
			}
		}
	}

	r.appendLiteral(old, new, order, wholeWord)
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

// PrepareReplacementsUUID registers the UUID replacement. Distinct UUIDs in a single
// output are numbered ([UUID][0], [UUID][1], ...) so the golden pins which value
// appears where; a single distinct UUID stays [UUID], even if repeated.
func PrepareReplacementsUUID(t testutil.TestingT, r *ReplacementsContext) {
	t.Helper()
	r.Repls = append(r.Repls, Replacement{Old: uuidRegex, New: "[UUID]", Distinct: true})
}

func PrepareReplacementSdkVersion(t testutil.TestingT, r *ReplacementsContext) {
	t.Helper()
	r.append(sdkVersionRegex, "databricks-sdk-go/[SDK_VERSION]", 0)
}

func PrepareReplacementTfProviderVersion(t testutil.TestingT, r *ReplacementsContext) {
	t.Helper()
	r.append(tfProviderVersionRegex, "databricks-tf-provider/[TF_PROVIDER_VERSION]", 0)
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
