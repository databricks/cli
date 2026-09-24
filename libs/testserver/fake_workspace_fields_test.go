package testserver

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type optionedDashFields struct {
	Skipped string `json:"-"`
	//nolint:govet,staticcheck // fixture intentionally exercises optioned dash tags
	Dash string `json:"-,omitempty"`
}

func TestApplyUpdatedFieldsUsesExactJSONSkip(t *testing.T) {
	existing := &optionedDashFields{Skipped: "old", Dash: "old"}
	update := optionedDashFields{Skipped: "new", Dash: "new"}
	fields := map[string]json.RawMessage{
		"-": json.RawMessage(`"new"`),
	}

	applyUpdatedFields(existing, update, fields)

	require.Equal(t, "old", existing.Skipped)
	require.Equal(t, "new", existing.Dash)
}
