package selection

import (
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

// CommitClouds are the prefixes a commit message line can name tests with. "cloud" names a
// test for every cloud; the others name it for that cloud only.
var CommitClouds = []string{"aws", "azure", "gcp", "cloud"}

// commitLineRe matches a "<cloud>: <test names>" line once surrounding whitespace is trimmed.
var commitLineRe = regexp.MustCompile(`^(aws|azure|gcp|cloud):(.*)$`)

// githubMergeRe matches the message of the merge commit GitHub creates for refs/pull/N/merge,
// which is what pull_request CI checks out: "Merge <head sha> into <base sha>".
var githubMergeRe = regexp.MustCompile(`^Merge ([0-9a-f]{40}) into [0-9a-f]{40}$`)

// testAcceptName is the go test name of the acceptance suite. Commit message names are full go
// test names, so they start with it.
const testAcceptName = "TestAccept"

// CommitName is one test name from a commit message line, with the cloud it was named for.
type CommitName struct {
	// Cloud is one of CommitClouds.
	Cloud string

	// Name is the test name as written, e.g. TestAccept/bundle/resources/jobs.
	Name string
}

// ParseCommitMessage returns the test names on the "<cloud>: <names>" lines of msg. A line may
// list several whitespace-separated names. Lines with any other prefix are ignored.
func ParseCommitMessage(msg string) []CommitName {
	var names []CommitName
	for line := range strings.SplitSeq(msg, "\n") {
		m := commitLineRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		for name := range strings.FieldsSeq(m[2]) {
			names = append(names, CommitName{Cloud: m[1], Name: name})
		}
	}
	return names
}

// NamesForCloud returns the names that apply to cloud (an aws/azure/gcp base cloud name), or
// every name when cloud is empty, which is a local run: there a test named for any cloud is
// also selected locally.
func NamesForCloud(names []CommitName, cloud string) []string {
	var out []string
	for _, n := range names {
		if cloud == "" || n.Cloud == "cloud" || n.Cloud == cloud {
			if !slices.Contains(out, n.Name) {
				out = append(out, n.Name)
			}
		}
	}
	return out
}

// Named maps a test dir to the variant filters of each name that selected it. A nil entry
// means a name selected every variant of the dir. Names are kept apart because each one's
// filters must all hold together, while the names themselves are alternatives.
type Named map[string][][]string

// Has reports whether any name selected dir.
func (n Named) Has(dir string) bool {
	_, ok := n[dir]
	return ok
}

// Matches reports whether a name selected the variant envset of dir.
func (n Named) Matches(dir string, envset []string) bool {
	for _, filters := range n[dir] {
		if filters == nil || MatchesFilters(envset, filters) {
			return true
		}
	}
	return false
}

// add records that a name selected dir with filters, nil for every variant.
func (n Named) add(dir string, filters []string) {
	n[dir] = append(n[dir], filters)
}

// ResolveNames maps go test names to the test dirs they select, and returns the names that
// selected nothing. A name selects:
//   - "TestAccept": every test;
//   - "TestAccept/<path>": every test dir that is <path> or under <path>/, with all variants;
//   - "TestAccept/<dir>/<KEY=value>[/<KEY=value>...]": the variants of test dir <dir> that
//     have those values, i.e. a variant subtest name as go test prints it.
func ResolveNames(names []string, testDirs map[string]bool) (Named, []string) {
	named := Named{}
	var unmatched []string
	for _, name := range names {
		if !resolveName(name, testDirs, named) {
			unmatched = append(unmatched, name)
		}
	}
	return named, unmatched
}

func resolveName(name string, testDirs map[string]bool, named Named) bool {
	rest, ok := strings.CutPrefix(name, testAcceptName)
	if !ok || (rest != "" && !strings.HasPrefix(rest, "/")) {
		return false
	}
	rest = strings.Trim(rest, "/")

	found := false
	for dir := range testDirs {
		if rest == "" || dir == rest || strings.HasPrefix(dir, rest+"/") {
			named.add(dir, nil)
			found = true
		}
	}
	if found {
		return true
	}

	// The longest test dir the name starts with, followed by the variant.
	for i := strings.LastIndex(rest, "/"); i > 0; i = strings.LastIndex(rest[:i], "/") {
		dir := rest[:i]
		if !testDirs[dir] {
			continue
		}
		filters := strings.Split(rest[i+1:], "/")
		for _, filter := range filters {
			if key, _, ok := strings.Cut(filter, "="); !ok || key == "" {
				return false
			}
		}
		named.add(dir, filters)
		return true
	}
	return false
}

// LastCommitMessage returns the message of the commit at HEAD. On the merge commit GitHub
// creates for a pull request, it returns the message of the pull request's head commit
// instead, which is the last commit its author pushed.
func LastCommitMessage() (string, error) {
	msg, err := commitMessage("HEAD")
	if err != nil {
		return "", err
	}
	if m := githubMergeRe.FindStringSubmatch(strings.TrimSpace(msg)); m != nil {
		return commitMessage(m[1])
	}
	return msg, nil
}

func commitMessage(rev string) (string, error) {
	out, err := exec.Command("git", "log", "-1", "--format=%B", rev).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git log -1 %s failed: %w: %s", rev, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// FromCommitMessage selects among testDirs the tests the last commit message names for cloud
// (empty for a local run, see NamesForCloud). It returns the selection, a one-line summary for
// the test log, and the names that matched no test.
func FromCommitMessage(testDirs map[string]bool, cloud string) (Named, string, []string, error) {
	msg, err := LastCommitMessage()
	if err != nil {
		return nil, "", nil, err
	}
	names := NamesForCloud(ParseCommitMessage(msg), cloud)
	named, unmatched := ResolveNames(names, testDirs)
	where := "local"
	if cloud != "" {
		where = cloud
	}
	summary := fmt.Sprintf("Selected %d test dirs named in the last commit message (%s): %s", len(named), where, strings.Join(names, " "))
	return named, summary, unmatched, nil
}
