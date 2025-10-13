package direct

import (
	"errors"

	"github.com/databricks/databricks-sdk-go/apierr"
)

func isResourceGone(err error) bool {
	return errors.Is(err, apierr.ErrResourceDoesNotExist) || errors.Is(err, apierr.ErrNotFound)
}
