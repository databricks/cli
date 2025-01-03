package template

import (
	"context"
	"errors"
)

type Resolver struct {
	TemplatePathOrUrl string
	ConfigFile        string
	OutputDir         string
	TemplateDir       string
	Tag               string
	Branch            string
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

	var tmpl *Template
	if r.TemplatePathOrUrl == "" {
		// Prompt the user to select a template
		// if a template path or URL is not provided.
		tmplId, err := SelectTemplate(ctx)
		if err != nil {
			return nil, err
		}

		if tmplId == Custom {
			return nil, ErrCustomSelected
		}

		tmpl = Get(tmplId)
	} else {
		// Based on the provided template path or URL,
		// configure a reader for the template.
		tmpl = Get(Custom)
		if IsGitRepoUrl(r.TemplatePathOrUrl) {
			tmpl.Reader = &gitReader{
				gitUrl:      r.TemplatePathOrUrl,
				ref:         ref,
				templateDir: r.TemplateDir,
			}
		} else {
			tmpl.Reader = &localReader{
				path: r.TemplatePathOrUrl,
			}
		}
	}

	err := tmpl.Writer.Configure(ctx, r.ConfigFile, r.OutputDir)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}
