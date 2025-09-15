package service

import "context"

// AppService defines the interface for general application operations
type AppService interface {
	// GetWelcomeMessage returns a welcome message
	GetWelcomeMessage(ctx context.Context) (map[string]interface{}, error)
	// HealthCheck performs a health check of the application
	HealthCheck(ctx context.Context) map[string]interface{}
	// GetStatus returns the current application status
	GetStatus(ctx context.Context) map[string]interface{}
}
