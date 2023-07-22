package config

type Lock struct {
	// Enabled toggles deployment lock. True by default.
	// Use a pointer value so that only explicitly configured values are set
	// and we don't merge configuration with zero-initialized values.
	Enabled *bool `json:"enabled"`
}

func (lock Lock) IsEnabled() bool {
	if lock.Enabled != nil {
		return *lock.Enabled
	}
	return true
}
