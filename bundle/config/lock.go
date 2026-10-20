package config

type Lock struct {
	// Enabled toggles deployment lock. True by default except in development mode.
	// Use a pointer value so that only explicitly configured values are set
	// and we don't merge configuration with zero-initialized values.
	Enabled *bool `json:"enabled,omitempty"`

	// Force acquisition of deployment lock even if it is currently held.
	// This may be necessary if a prior deployment failed to release the lock.
	Force bool `json:"force,omitempty"`
}

// IsEnabled checks if the deployment lock is enabled.
func (lock Lock) IsEnabled() bool {
	if lock.Enabled != nil {
		return *lock.Enabled
	}
	return true
}

// IsExplicitlyEnabled checks if the deployment lock is explicitly enabled.
// Only returns true if locking is explicitly set using a command-line
// flag or configuration file.
func (lock Lock) IsExplicitlyEnabled() bool {
	if lock.Enabled != nil {
		return *lock.Enabled
	}
	return false
}
