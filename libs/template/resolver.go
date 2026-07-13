package template

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/git"
)

type gitUrlPrefix struct {
	prefix string

	// invalid marks a prefix that git recognizes as a URL but that we refuse to
	// clone from, so we can report an actionable error instead of falling through
	// to the local-path reader.
	invalid bool
}

// See https://git-scm.com/docs/git-clone#_git_urls for the set of Git URL forms.
// We deliberately reject the deprecated/insecure transports (git, http, ftp[s]).
var gitUrlPrefixes = []gitUrlPrefix{
	{prefix: "https://"},
	{prefix: "ssh://"},
	// recognize git@ without ssh:// protocol because this is very common
	{prefix: "git@"},
	{prefix: "http://", invalid: true},
	{prefix: "git://", invalid: true},
	{prefix: "ftp://", invalid: true},
	{prefix: "ftps://", invalid: true},
}

// matchGitUrlPrefix returns the matching prefix entry, or nil if the input does
// not look like a Git URL.
func matchGitUrlPrefix(url string) *gitUrlPrefix {
	for i := range gitUrlPrefixes {
		if strings.HasPrefix(url, gitUrlPrefixes[i].prefix) {
			return &gitUrlPrefixes[i]
		}
	}
	return nil
}

func IsGitRepoUrl(url string) bool {
	p := matchGitUrlPrefix(url)
	return p != nil && !p.invalid
}

// ResolveReader resolves a template path/URL to a Reader (built-in, git or local)
func ResolveReader(templatePathOrUrl, templateDir, ref string) (Reader, bool, error) {
	if tmpl := GetDatabricksTemplate(TemplateName(templatePathOrUrl)); tmpl != nil {
		return tmpl.Reader, false, nil
	}

	if p := matchGitUrlPrefix(templatePathOrUrl); p != nil {
		if p.invalid {
			return nil, false, fmt.Errorf("unsupported protocol in Git URL %q: only %s URLs are supported", templatePathOrUrl, strings.Join(supportedGitUrlPrefixes(), ", "))
		}
		return NewGitReader(templatePathOrUrl, ref, templateDir, git.Clone), true, nil
	}

	return NewLocalReader(templatePathOrUrl), false, nil
}

// supportedGitUrlPrefixes returns the valid (non-rejected) Git URL prefixes.
func supportedGitUrlPrefixes() []string {
	var prefixes []string
	for i := range gitUrlPrefixes {
		if !gitUrlPrefixes[i].invalid {
			prefixes = append(prefixes, gitUrlPrefixes[i].prefix)
		}
	}
	return prefixes
}

type Resolver struct {
	// One of the following three:
	// 1. Path to a local template directory.
	// 2. URL to a Git repository containing a template.
	// 3. Name of a built-in template.
	TemplatePathOrUrl string

	// Path to a JSON file containing the configuration values to be used for
	// template initialization.
	ConfigFile string

	// Directory to write the initialized template to.
	OutputDir string

	// Directory path within a Git repository containing the template.
	TemplateDir string

	// Git tag or branch to download the template from. Only one of these can be
	// specified.
	Tag    string
	Branch string
}

// ErrCustomSelected is returned when the user selects the "custom..." option
// in the prompt UI when they run `databricks bundle init`. This error signals
// the upstream callsite to show documentation to the user on how to use a custom
// template.
var ErrCustomSelected = errors.New("custom template selected")

// Configures the reader and the writer for template and returns
// a handle to the template.
// Prompts the user if needed.
func (r Resolver) Resolve(ctx context.Context) (*Template, error) {
	if r.Tag != "" && r.Branch != "" {
		return nil, errors.New("only one of tag or branch can be specified")
	}

	// Git ref to use for template initialization
	ref := r.Branch
	if r.Tag != "" {
		ref = r.Tag
	}

	var err error
	var templateName TemplateName

	if r.TemplatePathOrUrl == "" {
		// Prompt the user to select a template
		// if a template path or URL is not provided.
		templateName, err = SelectTemplate(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		templateName = TemplateName(r.TemplatePathOrUrl)
	}

	tmpl := GetDatabricksTemplate(templateName)

	// If we could not find a databricks template with the name provided by the user,
	// then we assume that the user provided us with a reference to a custom template.
	//
	// This reference could be one of:
	// 1. Path to a local template directory.
	// 2. URL to a Git repository containing a template.
	//
	// We resolve the appropriate reader according to the reference provided by the user.
	if tmpl == nil {
		reader, _, err := ResolveReader(r.TemplatePathOrUrl, r.TemplateDir, ref)
		if err != nil {
			return nil, err
		}
		tmpl = &Template{
			name:   Custom,
			Reader: reader,
			// We use a writer that does not log verbose telemetry for custom templates.
			// This is important because template definitions can contain PII that we
			// do not want to centralize.
			Writer: &defaultWriter{name: Custom},
		}
	}
	err = tmpl.Writer.Configure(ctx, r.ConfigFile, r.OutputDir)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}
