package phases

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestSliceConfig(t *testing.T) {
	job1 := dyn.V(map[string]dyn.Value{"name": dyn.V("job1")})
	job2 := dyn.V(map[string]dyn.Value{"name": dyn.V("job2")})
	jobs := dyn.V(map[string]dyn.Value{"job1": job1, "job2": job2})
	resources := dyn.V(map[string]dyn.Value{"jobs": jobs})
	config := dyn.V(map[string]dyn.Value{"resources": resources})

	newConfig, err := sliceConfigResources(config, []resourcePath{{"jobs", "job1"}})
	assert.NoError(t, err)

	actualJob1, err := dyn.GetByPath(newConfig, dyn.MustPathFromString("resources.jobs.job1"))
	assert.NoError(t, err)
	assert.Equal(t, job1, actualJob1)

	actualJob2, err := dyn.GetByPath(newConfig, dyn.MustPathFromString("resources.jobs.job2"))
	assert.Error(t, err)
	assert.Equal(t, dyn.InvalidValue, actualJob2)
}
