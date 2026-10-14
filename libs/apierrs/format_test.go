package apierrs

import (
	"errors"
	"fmt"
	"testing"

	"github.com/databricks/sdk-go/core/apierr"
	"github.com/stretchr/testify/assert"
)

func TestErrorStringStripsSDKPrefix(t *testing.T) {
	apiErr := apierr.FromHTTPError(404, nil, []byte(`{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"Job 123 does not exist."}`))

	assert.Equal(t, "Job 123 does not exist.", ErrorString(apiErr))
	assert.Equal(t, "cannot read job: Job 123 does not exist.", ErrorString(fmt.Errorf("cannot read job: %w", apiErr)))
	assert.Equal(t, "a\nJob 123 does not exist.", ErrorString(errors.Join(errors.New("a"), apiErr)))
	assert.Equal(t, "plain", ErrorString(errors.New("plain")))
}
