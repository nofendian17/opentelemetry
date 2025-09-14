package service

import (
	"context"

	models "go-app/internal/domain/model"
)

// UserService defines the interface for user-related operations
type UserService interface {
	// Create creates a new user
	Create(ctx context.Context, user *models.User) error

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id int) (*models.User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*models.User, error)

	// List returns a list of all users
	List(ctx context.Context) ([]models.User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *models.User) error

	// Delete removes a user by ID
	Delete(ctx context.Context, id int) error
}
