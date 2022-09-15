package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadProjectConf(t *testing.T) {
	prj, err := loadProjectConf("./testdata")
	assert.NoError(t, err)
	assert.Equal(t, "dev", prj.Name)
	assert.True(t, prj.IsDevClusterJustReference())
}
