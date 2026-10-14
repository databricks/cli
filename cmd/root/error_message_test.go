package root

import (
	"errors"
	"fmt"
	"testing"

	"github.com/databricks/sdk-go/core/apierr"
	"github.com/stretchr/testify/assert"
)

func TestErrorMessageStripsAPIErrorPrefix(t *testing.T) {
	apiErr := apierr.FromHTTPError(404, nil, []byte(`{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"Job 123 does not exist."}`))

	assert.Equal(t, "Job 123 does not exist.", errorMessage(apiErr))
	assert.Equal(t, "cannot read job: Job 123 does not exist.", errorMessage(fmt.Errorf("cannot read job: %w", apiErr)))
	assert.Equal(t, "a\nJob 123 does not exist.", errorMessage(errors.Join(errors.New("a"), apiErr)))
	assert.Equal(t, "plain", errorMessage(errors.New("plain")))
}
