package repository

import (
	"context"

	"go-app/internal/domain/entity"
	"go-app/internal/domain/errors"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *entity.User) error

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id entity.UserID) (*entity.User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email entity.Email) (*entity.User, error)

	// List retrieves all users with optional pagination
	List(ctx context.Context, limit, offset int) ([]*entity.User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *entity.User) error

	// Delete removes a user by ID
	Delete(ctx context.Context, id entity.UserID) error

	// ExistsByEmail checks if a user with the given email exists
	ExistsByEmail(ctx context.Context, email entity.Email) (bool, error)

	// Count returns the total number of users
	Count(ctx context.Context) (int, error)
}

// Repository errors - these wrap the domain errors for repository-specific context
var (
	ErrUserNotFound      = errors.ErrUserNotFound
	ErrUserAlreadyExists = errors.ErrUserAlreadyExists
	ErrRepositoryError   = errors.ErrRepositoryError
)
