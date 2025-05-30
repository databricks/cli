package tnresources

import (
	"encoding/json"
	"errors"
	"fmt"
)

func copyViaJSON[T1, T2 any](dest *T1, src T2) error {
	if dest == nil {
		return errors.New("internal error: unexpected nil")
	}
	data, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("Failed to serialize %T: %w", src, err)
	}
	err = json.Unmarshal(data, dest)
	if err != nil {
		return fmt.Errorf("Failed JSON roundtrip from %T to %T: %w", src, dest, err)
	}
	return nil
}
