package template

import (
	"context"
	"errors"

	"github.com/databricks/cli/libs/git"
)

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

	Tag    string
	Branch string
}

var ErrCustomSelected = errors.New("custom template selected")

// Configures the reader and the writer for template and returns
// a handle to the template.
// Prompts the user if needed.
func (r Resolver) Resolve(ctx context.Context) (*Template, error) {
	if r.Tag != "" && r.Branch != "" {
		return nil, errors.New("only one of --tag or --branch can be specified")
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
	}

	templateName = TemplateName(r.TemplatePathOrUrl)

	// User should not directly select "custom" and instead should provide the
	// file path or the Git URL for the template directly.
	// Custom is just for internal representation purposes.
	if templateName == Custom {
		return nil, ErrCustomSelected
	}

	tmpl := Get(templateName)

	// If the user directory provided a template path or URL that is not a built-in template,
	// then configure a reader for the template.
	if tmpl == nil {
		tmpl = Get(Custom)
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
