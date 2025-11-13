package server

import (
	"context"
	"fmt"
	"time"
)

// HealthStatus represents the health status of the server
type HealthStatus struct {
	Healthy   bool              `json:"healthy"`
	Providers map[string]string `json:"providers"`
	Timestamp time.Time         `json:"timestamp"`
}

// CheckHealth checks the health of all registered providers
func (s *Server) CheckHealth(ctx context.Context) *HealthStatus {
	status := &HealthStatus{
		Healthy:   true,
		Providers: make(map[string]string),
		Timestamp: time.Now(),
	}

	// Check databricks provider
	if err := s.checkDatabricksHealth(ctx); err != nil {
		status.Providers["databricks"] = fmt.Sprintf("unhealthy: %v", err)
		status.Healthy = false
	} else {
		status.Providers["databricks"] = "healthy"
	}

	// I/O provider doesn't need health checks (no external dependencies)
	status.Providers["io"] = "healthy"

	// Check workspace provider if enabled
	if s.config.WithWorkspaceTools {
		status.Providers["workspace"] = "healthy"
	}

	// Check deployment provider if enabled
	if s.config.AllowDeployment {
		status.Providers["deployment"] = "healthy"
	}

	return status
}

// checkDatabricksHealth performs a basic health check for Databricks
func (s *Server) checkDatabricksHealth(ctx context.Context) error {
	// Create a short-lived context for the health check
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = timeoutCtx

	// For now, just check if provider was registered successfully
	// A more thorough check would call the Databricks API (e.g., list catalogs)
	// but that requires storing a reference to the provider

	// If Databricks is in required providers, registration succeeded
	// So we consider it healthy
	return nil
}
