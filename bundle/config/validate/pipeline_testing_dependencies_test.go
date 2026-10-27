package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPipelineTestingFileDependency(t *testing.T) {
	for _, dependency := range []string{
		"--editable /Workspace/Users/someone@example.com/orders[test]",
		"-e /Workspace/orders",
		"-r /Workspace/orders/tests/requirements.txt",
		"--requirement ./tests/requirements.txt",
		"/Volumes/main/default/wheels/helpers-0.1-py3-none-any.whl",
		"./dist/*.whl",
		"dist/helpers-0.1-py3-none-any.whl",
		"${resources.volumes.wheels.volume_path}/helpers-0.1-py3-none-any.whl",
	} {
		assert.True(t, isPipelineTestingFileDependency(dependency), dependency)
	}
	for _, dependency := range []string{
		"faker",
		"faker==33.0",
		"requests>=2",
		"git+https://github.com/example/helpers",
		"https://example.com/helpers-0.1-py3-none-any.whl",
		"--editable git+https://github.com/example/helpers",
	} {
		assert.False(t, isPipelineTestingFileDependency(dependency), dependency)
	}
}
