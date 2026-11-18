package phases

import (
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
)

func TestErrorClass(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"server error", &apierr.APIError{StatusCode: 500}, errorClassService},
		{"rate limited", &apierr.APIError{StatusCode: 429}, errorClassService},
		{"wrapped server error", fmt.Errorf("cannot create resources.jobs.foo: %w", &apierr.APIError{StatusCode: 503}), errorClassService},
		{"no response", &url.Error{Op: "Post", URL: "https://host.test", Err: errors.New("connection refused")}, errorClassService},
		{"permission denied", &apierr.APIError{StatusCode: 403}, errorClassUser},
		{"not an API error", errors.New("reference does not exist"), errorClassUser},
		{"no error to inspect", nil, errorClassUser},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, errorClass(c.err))
		})
	}
}
