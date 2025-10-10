package tfpermissions

import (
	"errors"
	"strings"

	"github.com/databricks/databricks-sdk-go/apierr"
)

// Suppress the error if it is 404
func IgnoreNotFoundError(err error) error {
	var apiErr *apierr.APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	if apiErr.StatusCode == 404 {
		return nil
	}
	if strings.Contains(apiErr.Message, "does not exist") {
		return nil
	}
	return err
}
