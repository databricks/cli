package config

type Lock struct {
	// Enabled toggles deployment lock. True by default.
	// Use a pointer value so that only explicitly configured values are set
	// and we don't merge configuration with zero-initialized values.
	Enabled *bool `json:"enabled"`

	// Force acquisition of deployment lock even if it is currently held.
	// This may be necessary if a prior deployment failed to release the lock.
	Force bool `json:"force"`
}

func (lock Lock) IsEnabled() bool {
	if lock.Enabled != nil {
		return *lock.Enabled
	}
	return true
}
