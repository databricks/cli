package libraries

import (
	"errors"

	"github.com/databricks/databricks-sdk-go/service/compute"
)

func libraryPath(library *compute.Library) (string, error) {
	if library.Whl != "" {
		return library.Whl, nil
	}
	if library.Jar != "" {
		return library.Jar, nil
	}
	if library.Egg != "" {
		return library.Egg, nil
	}
	if library.Requirements != "" {
		return library.Requirements, nil
	}

	return "", errors.New("not supported library type")
}
