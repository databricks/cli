package patchwheel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterLatestWheels(t *testing.T) {
	paths := []string{
		"project_name_bvs7tide6bhhpjy4dmcsb2qg44-0.0.1+20250604.74809-py3-none-any.whl",
		"not-a-wheel.txt",
		"mypkg-0.1.0-py3-none-any.whl",
		"mypkg-0.2.0-py3-none-any.whl",
		"other-1.0.0-py3-none-any.whl",
		"other-0.9.0-py3-none-any.whl",
		"project_name_bvs7tide6bhhpjy4dmcsb2qg44-0.0.1+20250604.74804-py3-none-any.whl",
		"not-a-wheel.whl",
		"hello-1.2.3-py3-none-any.whl",
		"hello-1.2.3+1741091696780123321-py3-none-any.whl",
	}

	filtered := FilterLatestWheels(context.Background(), paths)
	require.ElementsMatch(t, []string{
		"project_name_bvs7tide6bhhpjy4dmcsb2qg44-0.0.1+20250604.74809-py3-none-any.whl",
		"not-a-wheel.txt",
		"mypkg-0.2.0-py3-none-any.whl",
		"other-1.0.0-py3-none-any.whl",
		"not-a-wheel.whl",
		"hello-1.2.3+1741091696780123321-py3-none-any.whl",
	}, filtered)
}
