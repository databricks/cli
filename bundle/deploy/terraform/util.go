package terraform

import (
	"encoding/json"
	"io"
)

type state struct {
	Serial int `json:"serial"`
}

func IsLocalStateStale(local io.Reader, remote io.Reader) bool {
	localState, err := loadState(local)
	if err != nil {
		return true
	}

	remoteState, err := loadState(remote)
	if err != nil {
		return false
	}

	return localState.Serial < remoteState.Serial
}

func loadState(input io.Reader) (*state, error) {
	content, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	var s state
	err = json.Unmarshal(content, &s)
	if err != nil {
		return nil, err
	}

	return &s, nil
}
