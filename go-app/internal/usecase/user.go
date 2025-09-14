package usecase

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	models "go-app/internal/domain/model"
	"go-app/internal/domain/repository"
	"go-app/internal/infrastructure/telemetry"
)

type UserUseCase struct {
	repo      repository.UserRepository
	telemetry *telemetry.Telemetry
}

func NewUserUseCase(repo repository.UserRepository, tel *telemetry.Telemetry) *UserUseCase {
	return &UserUseCase{
		repo:      repo,
		telemetry: tel,
	}
}

// Create creates a new user
func (uc *UserUseCase) Create(ctx context.Context, user *models.User) error {
	ctx, span := uc.telemetry.Tracer.Start(ctx, "UserUseCase.Create")
	defer span.End()

	telemetry.Log(ctx, telemetry.LevelInfo, "Creating user",
		nil,
		semconv.HTTPRoute("/users"),
		attribute.String("handler", "create_user"),
		attribute.String("operation", "create"),
		attribute.String("name", user.Name),
		attribute.String("email", user.Email),
	)

	// Create user in memory
	err := uc.repo.Create(ctx, user)
	if err != nil {
		telemetry.Log(ctx, telemetry.LevelError, "Failed to create user", err, attribute.String("name", user.Name), attribute.String("email", user.Email))
		// Record metric for failed operation
		uc.telemetry.UserCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "create"),
			attribute.String("status", "error"),
		))
		return err
	}

	telemetry.Log(ctx, telemetry.LevelInfo, "User created successfully",
		nil,
		semconv.HTTPRoute("/users"),
		attribute.String("handler", "create_user"),
		attribute.String("operation", "create"),
		attribute.Int("user.id", user.ID),
	)

	// Record metric for successful operation
	uc.telemetry.UserCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("operation", "create"),
		attribute.String("status", "success"),
	))

	return nil
}

// GetByID retrieves a user by ID
func (uc *UserUseCase) GetByID(ctx context.Context, id int) (*models.User, error) {
	ctx, span := uc.telemetry.Tracer.Start(ctx, "UserUseCase.GetByID")
	defer span.End()
	telemetry.Log(ctx, telemetry.LevelInfo, "Fetching user by ID",
		nil,
		semconv.HTTPRoute("/users/{id}"),
		attribute.String("handler", "get_user"),
		attribute.String("operation", "read"),
		attribute.Int("user.id", id),
	)

	// Get user from memory
	user, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		// Record metric for failed operation
		uc.telemetry.UserCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "get_by_id"),
			attribute.String("status", "error"),
		))
		return nil, err
	}

	telemetry.Log(ctx, telemetry.LevelInfo, "User fetched successfully",
		nil,
		semconv.HTTPRoute("/users/{id}"),
		attribute.String("handler", "get_user"),
		attribute.String("operation", "read"),
		attribute.Int("user.id", id),
	)

	// Record metric for successful operation
	uc.telemetry.UserCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("operation", "get_by_id"),
		attribute.String("status", "success"),
	))

	return user, nil
}

// GetByEmail retrieves a user by email
func (uc *UserUseCase) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	ctx, span := uc.telemetry.Tracer.Start(ctx, "UserUseCase.GetByEmail")
	defer span.End()
	telemetry.Log(ctx, telemetry.LevelInfo, "Fetching user by email",
		nil,
		semconv.HTTPRoute("/users/email/{email}"),
		attribute.String("handler", "get_user_by_email"),
		attribute.String("operation", "read"),
		attribute.String("user.email", email),
	)

	// Get user from memory
	user, err := uc.repo.GetByEmail(ctx, email)
	if err != nil {
		// Record metric for failed operation
		uc.telemetry.UserCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "get_by_email"),
			attribute.String("status", "error"),
		))
		return nil, err
	}

	telemetry.Log(ctx, telemetry.LevelInfo, "User fetched successfully",
		nil,
		semconv.HTTPRoute("/users/email/{email}"),
		attribute.String("handler", "get_user_by_email"),
		attribute.String("operation", "read"),
		attribute.String("user.email", email),
	)

	// Record metric for successful operation
	uc.telemetry.UserCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("operation", "get_by_email"),
		attribute.String("status", "success"),
	))

	return user, nil
}

// List returns a list of all users
func (uc *UserUseCase) List(ctx context.Context) ([]models.User, error) {
	ctx, span := uc.telemetry.Tracer.Start(ctx, "UserUseCase.List")
	defer span.End()
	telemetry.Log(ctx, telemetry.LevelInfo, "Fetching all users",
		nil,
		semconv.HTTPRoute("/users"),
		attribute.String("handler", "list_users"),
		attribute.String("operation", "read"),
	)

	// Simulate some work with a child span
	_, childSpan := uc.telemetry.Tracer.Start(ctx, "fetch-users")
	childSpan.SetAttributes(attribute.String("db.operation", "SELECT"))
	defer childSpan.End()

	// Simulate database operation
	time.Sleep(50 * time.Millisecond)

	// Get users from memory
	users, err := uc.repo.List(ctx)
	if err != nil {
		// Record metric for failed operation
		uc.telemetry.UserCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "list"),
			attribute.String("status", "error"),
		))
		return nil, err
	}

	telemetry.Log(ctx, telemetry.LevelInfo, "Users fetched successfully",
		nil,
		semconv.HTTPRoute("/users"),
		attribute.String("handler", "list_users"),
		attribute.String("operation", "read"),
		attribute.Int("users.count", len(users)),
	)

	// Record metric for successful operation
	uc.telemetry.UserCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("operation", "list"),
		attribute.String("status", "success"),
	))

	return users, nil
}

// Update updates an existing user
func (uc *UserUseCase) Update(ctx context.Context, user *models.User) error {
	ctx, span := uc.telemetry.Tracer.Start(ctx, "UserUseCase.Update")
	defer span.End()
	telemetry.Log(ctx, telemetry.LevelInfo, "Updating user",
		nil,
		semconv.HTTPRoute("/users/{id}"),
		attribute.String("handler", "update_user"),
		attribute.String("operation", "update"),
		attribute.Int("user.id", user.ID),
	)

	// Update user in memory
	err := uc.repo.Update(ctx, user)
	if err != nil {
		// Record metric for failed operation
		uc.telemetry.UserCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "update"),
			attribute.String("status", "error"),
		))
		return err
	}

	telemetry.Log(ctx, telemetry.LevelInfo, "User updated successfully",
		nil,
		semconv.HTTPRoute("/users/{id}"),
		attribute.String("handler", "update_user"),
		attribute.String("operation", "update"),
		attribute.Int("user.id", user.ID),
	)

	// Record metric for successful operation
	uc.telemetry.UserCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("operation", "update"),
		attribute.String("status", "success"),
	))

	return nil
}

// Delete removes a user by ID
func (uc *UserUseCase) Delete(ctx context.Context, id int) error {
	ctx, span := uc.telemetry.Tracer.Start(ctx, "UserUseCase.Delete")
	defer span.End()
	telemetry.Log(ctx, telemetry.LevelInfo, "Deleting user",
		nil,
		semconv.HTTPRoute("/users/{id}"),
		attribute.String("handler", "delete_user"),
		attribute.String("operation", "delete"),
		attribute.Int("user.id", id),
	)

	// Delete user from memory
	err := uc.repo.Delete(ctx, id)
	if err != nil {
		// Record metric for failed operation
		uc.telemetry.UserCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "delete"),
			attribute.String("status", "error"),
		))
		return err
	}

	telemetry.Log(ctx, telemetry.LevelInfo, "User deleted successfully",
		nil,
		semconv.HTTPRoute("/users/{id}"),
		attribute.String("handler", "delete_user"),
		attribute.String("operation", "delete"),
		attribute.Int("user.id", id),
	)

	// Record metric for successful operation
	uc.telemetry.UserCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("operation", "delete"),
		attribute.String("status", "success"),
	))

	return nil
}
