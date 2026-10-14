package libraries

import (
	"errors"

	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/utils"
)

func libraryPath(library *jobs.Library) (string, error) {
	if whl := utils.Deref(library.Whl); whl != "" {
		return whl, nil
	}
	if jar := utils.Deref(library.Jar); jar != "" {
		return jar, nil
	}
	if egg := utils.Deref(library.Egg); egg != "" {
		return egg, nil
	}
	if requirements := utils.Deref(library.Requirements); requirements != "" {
		return requirements, nil
	}

	return "", errors.New("not supported library type")
}
