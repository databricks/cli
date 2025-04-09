package template

import (
	"context"
	"errors"
	"strings"

	"github.com/databricks/cli/libs/git"
)

var gitUrlPrefixes = []string{
	"https://",
	"git@",
}

func isRepoUrl(url string) bool {
	result := false
	for _, prefix := range gitUrlPrefixes {
		if strings.HasPrefix(url, prefix) {
			result = true
			break
		}
	}
	return result
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
		tmpl = &Template{
			name: Custom,
			// We use a writer that does not log verbose telemetry for custom templates.
			// This is important because template definitions can contain PII that we
			// do not want to centralize.
			Writer: &defaultWriter{name: Custom},
		}

		if isRepoUrl(r.TemplatePathOrUrl) {
			tmpl.Reader = &gitReader{
				gitUrl:      r.TemplatePathOrUrl,
				ref:         ref,
				templateDir: r.TemplateDir,
				cloneFunc:   git.Clone,
			}
		} else {
			tmpl.Reader = &localReader{
				path: r.TemplatePathOrUrl,
			}
		}
	}
	err = tmpl.Writer.Configure(ctx, r.ConfigFile, r.OutputDir)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}
