package service

import (
	"context"

	"go-app/internal/domain/entity"
)

// UserService defines the interface for user-related operations
type UserService interface {
	// Create creates a new user
	Create(ctx context.Context, user *entity.User) error

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id int) (*entity.User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*entity.User, error)

	// List returns a list of all users
	List(ctx context.Context) ([]*entity.User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *entity.User) error

	// Delete removes a user by ID
	Delete(ctx context.Context, id int) error
}
