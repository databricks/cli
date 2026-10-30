package validate

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

// pipelineTestingDependencyOptions are the pip options that may come before a test dependency's path.
var pipelineTestingDependencyOptions = []string{"--requirement", "--editable", "-r", "-e"}

const (
	pipelineTestingDependencyNotFile                = "not a file"
	pipelineTestingDependencyNeedsRequirementOption = "requirements file without -r"
)

type pipelineTestingDependencies struct{ bundle.RO }

// PipelineTestingDependencies rejects the entries of testing.environment.dependencies that the API
// rejects on deploy, such as package names and URLs. It runs after paths are translated.
func PipelineTestingDependencies() bundle.ReadOnlyMutator {
	return &pipelineTestingDependencies{}
}

func (v *pipelineTestingDependencies) Name() string {
	return "validate:pipeline_testing_dependencies"
}

func (v *pipelineTestingDependencies) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	pattern := dyn.NewPattern(
		dyn.Key("resources"), dyn.Key("pipelines"), dyn.AnyKey(),
		dyn.Key("testing"), dyn.Key("environment"), dyn.Key("dependencies"), dyn.AnyIndex(),
	)
	_, err := dyn.MapByPattern(b.Config.Value(), pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		dependency, ok := v.AsString()
		if !ok {
			return v, nil
		}
		summary := fmt.Sprintf("testing.environment.dependencies must point at files, not %q", dependency)
		detail := "A test dependency is a project folder, a wheel, or a requirements file after -r, in the bundle, under /Workspace/, or under /Volumes/. Declare test packages in a project file, such as the test extra of a pyproject.toml, and reference its folder, for example -e ./project[test]. For a path in the bundle, use -e or -r followed by a space. URLs, other pip options, and .. are not supported."
		switch pipelineTestingDependencyProblem(dependency) {
		case "":
			return v, nil
		case pipelineTestingDependencyNeedsRequirementOption:
			summary = fmt.Sprintf("testing.environment.dependencies needs -r before the requirements file %q", dependency)
			detail = "Without -r, pip treats the file as a package to install. For example: -r ./tests/requirements.txt."
		}
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   summary,
			Detail:    detail,
			Locations: v.Locations(),
			Paths:     dyn.ToStructPaths(p),
		})
		return v, nil
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}

// pipelineTestingDependencyProblem returns why dependency is a common mistake that the API rejects,
// or "" if it is not: a package name, a URL, another pip option, "..", a path outside /Workspace and
// /Volumes, or a requirements file without -r. Anything else is left to the API. dependency is a
// value after path translation, so a relative path is a local library that deploy uploads.
func pipelineTestingDependencyProblem(dependency string) string {
	// A reference resolved at deploy time, such as a volume path, is checked by the API.
	if dynvar.ContainsVariableReference(dependency) {
		return ""
	}
	value := strings.TrimSpace(dependency)
	option, filePath := cutPipelineTestingDependencyOption(value)
	switch {
	case strings.Contains(value, "://"):
		return pipelineTestingDependencyNotFile
	case strings.HasPrefix(filePath, "-") || strings.Contains(filePath, " -"):
		return pipelineTestingDependencyNotFile
	case strings.HasPrefix(filePath, "/"):
		if !strings.HasPrefix(filePath, "/Workspace/") && !strings.HasPrefix(filePath, "/Volumes/") {
			return pipelineTestingDependencyNotFile
		}
		if slices.Contains(strings.Split(filePath, "/"), "..") {
			return pipelineTestingDependencyNotFile
		}
	// Path translation rewrites a path in the bundle after "-r " or "-e ", so a relative path that
	// remains after an option was not rewritten.
	case option != "" || !libraries.IsLibraryLocal(filePath):
		return pipelineTestingDependencyNotFile
	}
	if strings.HasSuffix(strings.ToLower(filePath), ".txt") && option != "-r" && option != "--requirement" {
		return pipelineTestingDependencyNeedsRequirementOption
	}
	return ""
}

// cutPipelineTestingDependencyOption splits a leading pip option, such as "-r", from the path after it.
func cutPipelineTestingDependencyOption(value string) (string, string) {
	for _, option := range pipelineTestingDependencyOptions {
		rest, ok := strings.CutPrefix(value, option)
		if !ok {
			continue
		}
		switch {
		case strings.HasPrefix(rest, "=") && strings.HasPrefix(option, "--"):
			return option, strings.TrimSpace(rest[1:])
		case strings.HasPrefix(rest, " ") || strings.HasPrefix(rest, "\t"):
			return option, strings.TrimSpace(rest)
		case strings.HasPrefix(rest, "/") && !strings.HasPrefix(option, "--"):
			return option, rest
		}
	}
	return "", value
}
