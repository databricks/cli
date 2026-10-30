package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPipelineTestingDependencyProblem(t *testing.T) {
	accepted := []string{
		"--editable /Workspace/Users/someone@example.com/orders[test]",
		"-e /Workspace/orders",
		"-e/Workspace/orders",
		"--editable=/Workspace/orders",
		"-r /Workspace/orders/tests/requirements.txt",
		"--requirement=/Workspace/orders/requirements.txt",
		"--requirement /Volumes/main/default/reqs/requirements.TXT",
		"/Volumes/main/default/wheels/helpers-0.1-py3-none-any.whl",
		"/Workspace/Users/someone@example.com/my orders/project",
		"./dist/*.whl",
		"dist/helpers-0.1-py3-none-any.whl",
		"${resources.volumes.wheels.volume_path}/helpers-0.1-py3-none-any.whl",
		// Not a common mistake, so the API decides.
		"/Workspace/orders\n/Workspace/shared",
	}
	for _, dependency := range accepted {
		assert.Empty(t, pipelineTestingDependencyProblem(dependency), dependency)
	}

	notFiles := []string{
		// Package names.
		"faker",
		"faker==33.0",
		"requests>=2",
		// URLs.
		"git+https://github.com/example/helpers",
		"https://example.com/helpers-0.1-py3-none-any.whl",
		"--editable git+https://github.com/example/helpers",
		// Other pip options.
		"-i https://example.com/simple",
		"/Workspace/orders -i example",
		"--no-deps /Workspace/orders",
		// Paths outside /Workspace and /Volumes.
		"/dbfs/helpers-0.1-py3-none-any.whl",
		"/Users/someone@example.com/helpers-0.1-py3-none-any.whl",
		// "..".
		"/Workspace/orders/../shared",
		// A path in the bundle that path translation did not rewrite.
		"--editable ./project",
		"--requirement ./tests/requirements.txt",
	}
	for _, dependency := range notFiles {
		assert.Equal(t, pipelineTestingDependencyNotFile, pipelineTestingDependencyProblem(dependency), dependency)
	}

	needsRequirementOption := []string{
		"/Workspace/orders/requirements.txt",
		"--editable /Workspace/orders/requirements.txt",
		"./tests/requirements.txt",
	}
	for _, dependency := range needsRequirementOption {
		assert.Equal(t, pipelineTestingDependencyNeedsRequirementOption, pipelineTestingDependencyProblem(dependency), dependency)
	}
}
