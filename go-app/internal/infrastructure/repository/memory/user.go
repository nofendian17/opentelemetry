package memory

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"go-app/internal/domain/entity"
	"go-app/internal/domain/errors"
	"go-app/internal/infrastructure/telemetry"
)

// UserRepository implements UserRepository using in-memory storage
type UserRepository struct {
	mu     sync.RWMutex
	users  map[entity.UserID]*entity.User
	nextID entity.UserID
	tracer trace.Tracer
}

// NewUserRepository creates a new in-memory user repository
func NewUserRepository() *UserRepository {
	return &UserRepository{
		users:  make(map[entity.UserID]*entity.User),
		nextID: 1,
		tracer: trace.NewNoopTracerProvider().Tracer("memory-repository"),
	}
}

// WithTracer sets the tracer for the repository
func (r *UserRepository) WithTracer(tracer trace.Tracer) *UserRepository {
	r.tracer = tracer
	return r
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *entity.User) error {
	ctx, span := r.tracer.Start(ctx, "UserRepository.Create")
	span.SetAttributes(
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.collection", "users"),
	)
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if user with same email already exists
	for _, u := range r.users {
		if u.Email() == user.Email() {
			err := errors.ErrUserAlreadyExists.WithContext("email", user.Email().String())
			telemetry.Log(ctx, telemetry.LevelError, "User already exists", err,
				attribute.String("db.operation", "INSERT"),
				attribute.String("db.collection", "users"),
				attribute.String("error", "user already exists"),
			)
			return err
		}
	}

	// Assign ID and store user
	user.SetID(r.nextID)
	r.users[r.nextID] = user
	r.nextID++

	telemetry.Log(ctx, telemetry.LevelInfo, "User created in memory", nil,
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.collection", "users"),
		attribute.String("user.id", user.ID().String()),
	)
	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id entity.UserID) (*entity.User, error) {
	ctx, span := r.tracer.Start(ctx, "UserRepository.GetByID")
	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.collection", "users"),
		attribute.String("user.id", id.String()),
	)
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		err := errors.ErrUserNotFound.WithContext("id", id.String())
		telemetry.Log(ctx, telemetry.LevelError, "User not found", err,
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.collection", "users"),
			attribute.String("user.id", id.String()),
			attribute.String("error", "user not found"),
		)
		return nil, err
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email entity.Email) (*entity.User, error) {
	ctx, span := r.tracer.Start(ctx, "UserRepository.GetByEmail")
	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.collection", "users"),
		attribute.String("user.email", email.String()),
	)
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.Email() == email {
			span.SetAttributes(attribute.String("user.id", user.ID().String()))
			return user, nil
		}
	}

	err := errors.ErrUserNotFound.WithContext("email", email.String())
	telemetry.Log(ctx, telemetry.LevelError, "User not found", err,
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.collection", "users"),
		attribute.String("user.email", email.String()),
		attribute.String("error", "user not found"),
	)
	return nil, err
}

// List retrieves all users with optional pagination
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*entity.User, error) {
	ctx, span := r.tracer.Start(ctx, "UserRepository.List")
	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.collection", "users"),
		attribute.Int("limit", limit),
		attribute.Int("offset", offset),
	)
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Convert map to slice for consistent ordering
	allUsers := make([]*entity.User, 0, len(r.users))
	for _, user := range r.users {
		allUsers = append(allUsers, user)
	}

	// Apply pagination
	start := offset
	if start < 0 {
		start = 0
	}
	if start >= len(allUsers) {
		return []*entity.User{}, nil
	}

	end := start + limit
	if limit <= 0 || end > len(allUsers) {
		end = len(allUsers)
	}

	users := allUsers[start:end]
	span.SetAttributes(attribute.Int("users.count", len(users)))

	return users, nil
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, user *entity.User) error {
	ctx, span := r.tracer.Start(ctx, "UserRepository.Update")
	span.SetAttributes(
		attribute.String("db.operation", "UPDATE"),
		attribute.String("db.collection", "users"),
		attribute.String("user.id", user.ID().String()),
	)
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if user exists
	_, exists := r.users[user.ID()]
	if !exists {
		err := errors.ErrUserNotFound.WithContext("id", user.ID().String())
		telemetry.Log(ctx, telemetry.LevelError, "User not found", err,
			attribute.String("db.operation", "UPDATE"),
			attribute.String("db.collection", "users"),
			attribute.String("user.id", user.ID().String()),
			attribute.String("error", "user not found"),
		)
		return err
	}

	// Check if another user already has this email
	for id, u := range r.users {
		if id != user.ID() && u.Email() == user.Email() {
			err := errors.ErrUserAlreadyExists.WithContext("email", user.Email().String())
			telemetry.Log(ctx, telemetry.LevelError, "User with this email already exists", err,
				attribute.String("db.operation", "UPDATE"),
				attribute.String("db.collection", "users"),
				attribute.String("user.id", user.ID().String()),
				attribute.String("user.email", user.Email().String()),
				attribute.String("error", "user with this email already exists"),
			)
			return err
		}
	}

	// Update user
	r.users[user.ID()] = user

	telemetry.Log(ctx, telemetry.LevelInfo, "User updated in memory", nil,
		attribute.String("db.operation", "UPDATE"),
		attribute.String("db.collection", "users"),
		attribute.String("user.id", user.ID().String()),
	)

	return nil
}

// Delete removes a user by ID
func (r *UserRepository) Delete(ctx context.Context, id entity.UserID) error {
	ctx, span := r.tracer.Start(ctx, "UserRepository.Delete")
	span.SetAttributes(
		attribute.String("db.operation", "DELETE"),
		attribute.String("db.collection", "users"),
		attribute.String("user.id", id.String()),
	)
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if user exists
	_, exists := r.users[id]
	if !exists {
		err := errors.ErrUserNotFound.WithContext("id", id.String())
		telemetry.Log(ctx, telemetry.LevelError, "User not found", err,
			attribute.String("db.operation", "DELETE"),
			attribute.String("db.collection", "users"),
			attribute.String("user.id", id.String()),
			attribute.String("error", "user not found"),
		)
		return err
	}

	// Delete user
	delete(r.users, id)

	telemetry.Log(ctx, telemetry.LevelInfo, "User deleted from memory", nil,
		attribute.String("db.operation", "DELETE"),
		attribute.String("db.collection", "users"),
		attribute.String("user.id", id.String()),
	)

	return nil
}

// ExistsByEmail checks if a user with the given email exists
func (r *UserRepository) ExistsByEmail(ctx context.Context, email entity.Email) (bool, error) {
	ctx, span := r.tracer.Start(ctx, "UserRepository.ExistsByEmail")
	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.collection", "users"),
		attribute.String("user.email", email.String()),
	)
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.Email() == email {
			return true, nil
		}
	}

	return false, nil
}

// Count returns the total number of users
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	ctx, span := r.tracer.Start(ctx, "UserRepository.Count")
	span.SetAttributes(
		attribute.String("db.operation", "COUNT"),
		attribute.String("db.collection", "users"),
	)
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	count := len(r.users)
	span.SetAttributes(attribute.Int("users.count", count))

	return count, nil
}
