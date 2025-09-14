package repository

import (
	"context"
	"errors"

	models "go-app/internal/domain/model"
)

var (
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")

	// ErrUserAlreadyExists is returned when trying to create a user that already exists
	ErrUserAlreadyExists = errors.New("user already exists")
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *models.User) error

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id int) (*models.User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*models.User, error)

	// List retrieves all users
	List(ctx context.Context) ([]models.User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *models.User) error

	// Delete removes a user by ID
	Delete(ctx context.Context, id int) error
}
