package project

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadProjectConf(t *testing.T) {
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	os.Chdir("testdata/a/b/c")

	prj, err := loadProjectConf()
	assert.NoError(t, err)
	assert.Equal(t, "dev", prj.Name)
	assert.True(t, prj.IsDevClusterJustReference())
}
