package diag

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiagnosticContainsError(t *testing.T) {
	diags := Diagnostics{
		{
			Severity: Error,
			Summary:  "error 1",
		},
		{
			Severity: Error,
			Summary:  "error 2",
		},
		{
			Severity: Warning,
			Summary:  "warning 1",
		},
	}

	assert.True(t, diags.ContainsError(errors.New("error 1")))
	assert.True(t, diags.ContainsError(errors.New("error 2")))
	assert.False(t, diags.ContainsError(errors.New("error 3")))
}

func TestDiagnosticRemoveError(t *testing.T) {
	diags := Diagnostics{
		{
			Severity: Error,
			Summary:  "error 1",
		},
		{
			Severity: Error,
			Summary:  "error 2",
		},
		{
			Severity: Warning,
			Summary:  "warning 1",
		},
	}

	filtered := diags.RemoveError(errors.New("error 1"))
	assert.Len(t, filtered, 2)
	assert.Equal(t, Diagnostics{
		{
			Severity: Error,
			Summary:  "error 2",
		},
		{
			Severity: Warning,
			Summary:  "warning 1",
		},
	}, filtered)

	filtered = diags.RemoveError(errors.New("error 2"))
	assert.Len(t, filtered, 2)
	assert.Equal(t, Diagnostics{
		{
			Severity: Error,
			Summary:  "error 1",
		},
		{
			Severity: Warning,
			Summary:  "warning 1",
		},
	}, filtered)

	filtered = diags.RemoveError(errors.New("warning 1"))
	assert.Len(t, filtered, 3)
	assert.Equal(t, Diagnostics{
		{
			Severity: Error,
			Summary:  "error 1",
		},
		{
			Severity: Error,
			Summary:  "error 2",
		},
		{
			Severity: Warning,
			Summary:  "warning 1",
		},
	}, filtered)
}
