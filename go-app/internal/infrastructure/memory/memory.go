package memory

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	models "go-app/internal/domain/model"
	"go-app/internal/domain/repository"
	"go-app/internal/infrastructure/telemetry"
)

// InMemoryRepository implements UserRepository using in-memory storage
type InMemoryRepository struct {
	mu     sync.RWMutex
	users  map[int]models.User
	nextID int
}

// NewInMemoryRepository creates a new in-memory user memory
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		users:  make(map[int]models.User),
		nextID: 1,
	}
}

// Create creates a new user
func (r *InMemoryRepository) Create(ctx context.Context, user *models.User) error {
	// Add span for database operation
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("memory-repository").Start(ctx, "CreateUser")
	span.SetAttributes(
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.collection", "users"),
	)
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if user with same email already exists
	for _, u := range r.users {
		if u.Email == user.Email {
			telemetry.Log(ctx, telemetry.LevelError, "User already exists", repository.ErrUserAlreadyExists,
				attribute.String("db.operation", "INSERT"),
				attribute.String("db.collection", "users"),
				attribute.String("error", "user already exists"),
			)
			return repository.ErrUserAlreadyExists
		}
	}

	// Assign ID and store user
	user.ID = r.nextID
	r.users[r.nextID] = *user
	r.nextID++

	telemetry.Log(ctx, telemetry.LevelInfo, "User created in memory", nil,
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.collection", "users"),
		attribute.Int("user.id", user.ID),
	)
	return nil
}

// GetByID retrieves a user by ID
func (r *InMemoryRepository) GetByID(ctx context.Context, id int) (*models.User, error) {
	// Add span for database operation
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("memory-repository").Start(ctx, "GetUserByID")
	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.collection", "users"),
		attribute.Int("user.id", id),
	)
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		telemetry.Log(ctx, telemetry.LevelError, "User not found", repository.ErrUserNotFound,
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.collection", "users"),
			attribute.Int("user.id", id),
			attribute.String("error", "user not found"),
		)
		return nil, repository.ErrUserNotFound
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *InMemoryRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	// Add span for database operation
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("memory-repository").Start(ctx, "GetUserByEmail")
	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.collection", "users"),
		attribute.String("user.email", email),
	)
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.Email == email {
			span.SetAttributes(attribute.Int("user.id", user.ID))
			return &user, nil
		}
	}

	telemetry.Log(ctx, telemetry.LevelError, "User not found", repository.ErrUserNotFound,
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.collection", "users"),
		attribute.String("user.email", email),
		attribute.String("error", "user not found"),
	)
	return nil, repository.ErrUserNotFound
}

// List retrieves all users
func (r *InMemoryRepository) List(ctx context.Context) ([]models.User, error) {
	// Add span for database operation
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("memory-repository").Start(ctx, "ListUsers")
	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.collection", "users"),
	)
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]models.User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}

	span.SetAttributes(attribute.Int("users.count", len(users)))
	return users, nil
}

// Update updates an existing user
func (r *InMemoryRepository) Update(ctx context.Context, user *models.User) error {
	// Add span for database operation
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("memory-repository").Start(ctx, "UpdateUser")
	span.SetAttributes(
		attribute.String("db.operation", "UPDATE"),
		attribute.String("db.collection", "users"),
		attribute.Int("user.id", user.ID),
	)
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if user exists
	_, exists := r.users[user.ID]
	if !exists {
		telemetry.Log(ctx, telemetry.LevelError, "User not found", repository.ErrUserNotFound,
			attribute.String("db.operation", "UPDATE"),
			attribute.String("db.collection", "users"),
			attribute.Int("user.id", user.ID),
			attribute.String("error", "user not found"),
		)
		return repository.ErrUserNotFound
	}

	// Check if another user already has this email
	for id, u := range r.users {
		if id != user.ID && u.Email == user.Email {
			telemetry.Log(ctx, telemetry.LevelError, "User with this email already exists", repository.ErrUserAlreadyExists,
				attribute.String("db.operation", "UPDATE"),
				attribute.String("db.collection", "users"),
				attribute.Int("user.id", user.ID),
				attribute.String("user.email", user.Email),
				attribute.String("error", "user with this email already exists"),
			)
			return repository.ErrUserAlreadyExists
		}
	}

	// Update user
	r.users[user.ID] = *user

	return nil
}

// Delete removes a user by ID
func (r *InMemoryRepository) Delete(ctx context.Context, id int) error {
	// Add span for database operation
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("memory-repository").Start(ctx, "DeleteUser")
	span.SetAttributes(
		attribute.String("db.operation", "DELETE"),
		attribute.String("db.collection", "users"),
		attribute.Int("user.id", id),
	)
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if user exists
	_, exists := r.users[id]
	if !exists {
		telemetry.Log(ctx, telemetry.LevelError, "User not found", repository.ErrUserNotFound,
			attribute.String("db.operation", "DELETE"),
			attribute.String("db.collection", "users"),
			attribute.Int("user.id", id),
			attribute.String("error", "user not found"),
		)
		return repository.ErrUserNotFound
	}

	// Delete user
	delete(r.users, id)

	return nil
}
