package selection

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

// commitClouds are the prefixes a commit message line can name tests with. "cloud" names a
// test for every cloud; the others name it for that cloud only. tools/testmask/commit.go
// repeats this list (the tools module cannot import this package); keep the two in sync.
var commitClouds = []string{"aws", "azure", "gcp", "cloud"}

// commitLineRe matches a "<cloud>: <test names>" line once surrounding whitespace is trimmed.
var commitLineRe = regexp.MustCompile(`^(` + strings.Join(commitClouds, "|") + `):(.*)$`)

// githubMergeRe matches the message of the merge commit GitHub creates for refs/pull/N/merge,
// which is what pull_request CI checks out: "Merge <head sha> into <base sha>".
var githubMergeRe = regexp.MustCompile(`^Merge ([0-9a-f]{40}) into [0-9a-f]{40}$`)

// testAcceptName is the go test name of the acceptance suite. Commit message names are full go
// test names, so they start with it.
const testAcceptName = "TestAccept"

// CommitName is one test name from a commit message line, with the cloud it was named for.
type CommitName struct {
	// Cloud is one of commitClouds.
	Cloud string

	// Name is the test name as written, e.g. TestAccept/bundle/resources/jobs.
	Name string
}

// ParseCommitMessage returns the test names on the "<cloud>: <names>" lines of msg. A line may
// list several whitespace-separated names. Lines with any other prefix are ignored, and so are
// lines with a word that is not a go test name: "aws: fix flaky TestAccept retries" is prose,
// and taking its bare TestAccept as a name would run the whole suite.
func ParseCommitMessage(msg string) []CommitName {
	var names []CommitName
	for line := range strings.SplitSeq(msg, "\n") {
		m := commitLineRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		words := strings.Fields(m[2])
		if len(words) == 0 || slices.ContainsFunc(words, func(word string) bool { return !strings.HasPrefix(word, "Test") }) {
			continue
		}
		for _, name := range words {
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
//     have those values, i.e. a variant subtest name as go test prints it. root is the
//     acceptance directory, read for the variants each dir runs, so a misspelled key or value
//     is reported rather than selecting every variant or none.
func ResolveNames(root string, names []string, testDirs map[string]bool) (Named, []string) {
	named := Named{}
	var unmatched []string
	for _, name := range names {
		if !resolveName(root, name, testDirs, named) {
			unmatched = append(unmatched, name)
		}
	}
	return named, unmatched
}

func resolveName(root, name string, testDirs map[string]bool, named Named) bool {
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
		if !slices.ContainsFunc(Variants(root, dir), func(envset []string) bool { return hasAll(envset, filters) }) {
			return false
		}
		named.add(dir, filters)
		return true
	}
	return false
}

// hasAll reports whether every KEY=value filter is literally in envset. Unlike MatchesFilters,
// a key the envset lacks does not match, so a misspelled key selects nothing.
func hasAll(envset, filters []string) bool {
	for _, filter := range filters {
		if !slices.Contains(envset, filter) {
			return false
		}
	}
	return true
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
	out, err := exec.Command("git", "log", "-1", "--format=%B", rev).Output()
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return "", fmt.Errorf("git log -1 %s failed: %w: %s", rev, err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git log -1 %s failed: %w", rev, err)
	}
	return string(out), nil
}

// FromCommitMessage selects among testDirs the tests the last commit message names for cloud
// (empty for a local run, see NamesForCloud). root is the acceptance directory. It returns the
// selection, a one-line summary of the names that matched for the test log, and the names that
// matched no test.
func FromCommitMessage(root string, testDirs map[string]bool, cloud string) (Named, string, []string, error) {
	msg, err := LastCommitMessage()
	if err != nil {
		return nil, "", nil, err
	}
	names := NamesForCloud(ParseCommitMessage(msg), cloud)
	named, unmatched := ResolveNames(root, names, testDirs)
	matched := slices.DeleteFunc(names, func(name string) bool { return slices.Contains(unmatched, name) })
	where := "local"
	if cloud != "" {
		where = cloud
	}
	summary := fmt.Sprintf("Selected %d test dirs named in the last commit message (%s): %s", len(named), where, strings.Join(matched, " "))
	return named, summary, unmatched, nil
}
