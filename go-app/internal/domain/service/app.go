package service

import "context"

// AppService defines the interface for general application operations
type AppService interface {
	// GetWelcomeMessage returns a welcome message
	GetWelcomeMessage(ctx context.Context) (map[string]interface{}, error)
}
