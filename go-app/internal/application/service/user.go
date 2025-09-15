package service

import (
	"context"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"go-app/internal/application/dto"
	"go-app/internal/domain/entity"
	"go-app/internal/domain/errors"
	"go-app/internal/domain/repository"
	"go-app/internal/infrastructure/telemetry"
)

// UserService handles user-related business operations
type UserService struct {
	repo      repository.UserRepository
	telemetry *telemetry.Telemetry
	tracer    trace.Tracer
}

// NewUserService creates a new UserService
func NewUserService(repo repository.UserRepository, tel *telemetry.Telemetry) *UserService {
	return &UserService{
		repo:      repo,
		telemetry: tel,
		tracer:    tel.Tracer,
	}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, req dto.CreateUserRequest) (*dto.UserResponse, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.CreateUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("operation", "create_user"),
		attribute.String("user.name", req.Name),
		attribute.String("user.email", req.Email),
	)

	telemetry.Log(ctx, telemetry.LevelInfo, "Creating user",
		nil,
		semconv.HTTPRoute("/users"),
		attribute.String("handler", "create_user"),
		attribute.String("operation", "create"),
		attribute.String("name", req.Name),
		attribute.String("email", req.Email),
	)

	// Validate request
	if err := req.Validate(); err != nil {
		span.SetAttributes(attribute.String("error", "validation_failed"))
		s.recordMetric(ctx, "create", "validation_error")
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeValidationFailed, "request validation failed", err)
	}

	// Create domain entity
	user, err := entity.NewUser(req.Name, req.Email)
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_user_data"))
		s.recordMetric(ctx, "create", "validation_error")
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeInvalidUserData, "failed to create user entity", err)
	}

	// Check if user already exists
	email := user.Email()
	exists, err := s.repo.ExistsByEmail(ctx, email)
	if err != nil {
		span.SetAttributes(attribute.String("error", "repository_error"))
		s.recordMetric(ctx, "create", "error")
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to check user existence", err)
	}
	if exists {
		span.SetAttributes(attribute.String("error", "user_already_exists"))
		s.recordMetric(ctx, "create", "conflict")
		return nil, errors.ErrUserAlreadyExists.WithContext("email", email.String())
	}

	// Save user
	if err := s.repo.Create(ctx, user); err != nil {
		span.SetAttributes(attribute.String("error", "repository_error"))
		s.recordMetric(ctx, "create", "error")
		telemetry.Log(ctx, telemetry.LevelError, "Failed to create user", err,
			attribute.String("name", req.Name),
			attribute.String("email", req.Email))
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to save user", err)
	}

	telemetry.Log(ctx, telemetry.LevelInfo, "User created successfully",
		nil,
		semconv.HTTPRoute("/users"),
		attribute.String("handler", "create_user"),
		attribute.String("operation", "create"),
		attribute.String("user.id", user.ID().String()),
	)

	s.recordMetric(ctx, "create", "success")
	return dto.NewUserResponse(user), nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, idStr string) (*dto.UserResponse, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.GetUserByID")
	defer span.End()

	span.SetAttributes(
		attribute.String("operation", "get_user_by_id"),
		attribute.String("user.id", idStr),
	)

	telemetry.Log(ctx, telemetry.LevelInfo, "Fetching user by ID",
		nil,
		semconv.HTTPRoute("/users/{id}"),
		attribute.String("handler", "get_user"),
		attribute.String("operation", "read"),
		attribute.String("user.id", idStr),
	)

	// Parse and validate ID
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		span.SetAttributes(attribute.String("error", "invalid_id"))
		s.recordMetric(ctx, "get_by_id", "validation_error")
		return nil, errors.ErrInvalidID.WithContext("id", idStr)
	}

	userID := entity.UserID(id)

	// Get user from repository
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.IsUserNotFound(err) {
			span.SetAttributes(attribute.String("error", "user_not_found"))
			s.recordMetric(ctx, "get_by_id", "not_found")
			return nil, err
		}
		span.SetAttributes(attribute.String("error", "repository_error"))
		s.recordMetric(ctx, "get_by_id", "error")
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to get user", err)
	}

	telemetry.Log(ctx, telemetry.LevelInfo, "User fetched successfully",
		nil,
		semconv.HTTPRoute("/users/{id}"),
		attribute.String("handler", "get_user"),
		attribute.String("operation", "read"),
		attribute.String("user.id", user.ID().String()),
	)

	s.recordMetric(ctx, "get_by_id", "success")
	return dto.NewUserResponse(user), nil
}

// GetUserByEmail retrieves a user by email
func (s *UserService) GetUserByEmail(ctx context.Context, emailStr string) (*dto.UserResponse, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.GetUserByEmail")
	defer span.End()

	span.SetAttributes(
		attribute.String("operation", "get_user_by_email"),
		attribute.String("user.email", emailStr),
	)

	telemetry.Log(ctx, telemetry.LevelInfo, "Fetching user by email",
		nil,
		semconv.HTTPRoute("/users/email/{email}"),
		attribute.String("handler", "get_user_by_email"),
		attribute.String("operation", "read"),
		attribute.String("user.email", emailStr),
	)

	// Validate email
	email, err := entity.NewEmail(emailStr)
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_email"))
		s.recordMetric(ctx, "get_by_email", "validation_error")
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeInvalidEmail, "invalid email format", err)
	}

	// Get user from repository
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.IsUserNotFound(err) {
			span.SetAttributes(attribute.String("error", "user_not_found"))
			s.recordMetric(ctx, "get_by_email", "not_found")
			return nil, err
		}
		span.SetAttributes(attribute.String("error", "repository_error"))
		s.recordMetric(ctx, "get_by_email", "error")
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to get user", err)
	}

	telemetry.Log(ctx, telemetry.LevelInfo, "User fetched successfully",
		nil,
		semconv.HTTPRoute("/users/email/{email}"),
		attribute.String("handler", "get_user_by_email"),
		attribute.String("operation", "read"),
		attribute.String("user.email", emailStr),
	)

	s.recordMetric(ctx, "get_by_email", "success")
	return dto.NewUserResponse(user), nil
}

// ListUsers returns a list of all users with pagination
func (s *UserService) ListUsers(ctx context.Context, req dto.ListUsersRequest) (*dto.ListUsersResponse, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.ListUsers")
	defer span.End()

	span.SetAttributes(
		attribute.String("operation", "list_users"),
		attribute.Int("limit", req.Limit),
		attribute.Int("offset", req.Offset),
	)

	telemetry.Log(ctx, telemetry.LevelInfo, "Fetching all users",
		nil,
		semconv.HTTPRoute("/users"),
		attribute.String("handler", "list_users"),
		attribute.String("operation", "read"),
		attribute.Int("limit", req.Limit),
		attribute.Int("offset", req.Offset),
	)

	// Validate request
	if err := req.Validate(); err != nil {
		span.SetAttributes(attribute.String("error", "validation_failed"))
		s.recordMetric(ctx, "list", "validation_error")
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeValidationFailed, "request validation failed", err)
	}

	// Simulate some work with a child span
	_, childSpan := s.tracer.Start(ctx, "fetch-users")
	childSpan.SetAttributes(attribute.String("db.operation", "SELECT"))
	defer childSpan.End()

	// Simulate database operation
	time.Sleep(50 * time.Millisecond)

	// Get users from repository
	users, err := s.repo.List(ctx, req.Limit, req.Offset)
	if err != nil {
		span.SetAttributes(attribute.String("error", "repository_error"))
		s.recordMetric(ctx, "list", "error")
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to list users", err)
	}

	// Get total count
	total, err := s.repo.Count(ctx)
	if err != nil {
		span.SetAttributes(attribute.String("error", "repository_error"))
		s.recordMetric(ctx, "list", "error")
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to count users", err)
	}

	telemetry.Log(ctx, telemetry.LevelInfo, "Users fetched successfully",
		nil,
		semconv.HTTPRoute("/users"),
		attribute.String("handler", "list_users"),
		attribute.String("operation", "read"),
		attribute.Int("users.count", len(users)),
		attribute.Int("total.count", total),
	)

	s.recordMetric(ctx, "list", "success")
	return dto.NewListUsersResponse(users, total, req.Limit, req.Offset), nil
}

// UpdateUser updates an existing user
func (s *UserService) UpdateUser(ctx context.Context, idStr string, req dto.UpdateUserRequest) (*dto.UserResponse, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.UpdateUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("operation", "update_user"),
		attribute.String("user.id", idStr),
		attribute.String("user.name", req.Name),
		attribute.String("user.email", req.Email),
	)

	telemetry.Log(ctx, telemetry.LevelInfo, "Updating user",
		nil,
		semconv.HTTPRoute("/users/{id}"),
		attribute.String("handler", "update_user"),
		attribute.String("operation", "update"),
		attribute.String("user.id", idStr),
	)

	// Parse and validate ID
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		span.SetAttributes(attribute.String("error", "invalid_id"))
		s.recordMetric(ctx, "update", "validation_error")
		return nil, errors.ErrInvalidID.WithContext("id", idStr)
	}

	userID := entity.UserID(id)

	// Validate request
	if err := req.Validate(); err != nil {
		span.SetAttributes(attribute.String("error", "validation_failed"))
		s.recordMetric(ctx, "update", "validation_error")
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeValidationFailed, "request validation failed", err)
	}

	// Get existing user
	existingUser, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.IsUserNotFound(err) {
			span.SetAttributes(attribute.String("error", "user_not_found"))
			s.recordMetric(ctx, "update", "not_found")
			return nil, err
		}
		span.SetAttributes(attribute.String("error", "repository_error"))
		s.recordMetric(ctx, "update", "error")
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to get user", err)
	}

	// Update user fields
	if err := existingUser.UpdateName(req.Name); err != nil {
		span.SetAttributes(attribute.String("error", "invalid_name"))
		s.recordMetric(ctx, "update", "validation_error")
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeInvalidName, "failed to update name", err)
	}

	if err := existingUser.UpdateEmail(req.Email); err != nil {
		span.SetAttributes(attribute.String("error", "invalid_email"))
		s.recordMetric(ctx, "update", "validation_error")
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeInvalidEmail, "failed to update email", err)
	}

	// Save updated user
	if err := s.repo.Update(ctx, existingUser); err != nil {
		if errors.IsUserAlreadyExists(err) {
			span.SetAttributes(attribute.String("error", "email_conflict"))
			s.recordMetric(ctx, "update", "conflict")
			return nil, err
		}
		span.SetAttributes(attribute.String("error", "repository_error"))
		s.recordMetric(ctx, "update", "error")
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to update user", err)
	}

	telemetry.Log(ctx, telemetry.LevelInfo, "User updated successfully",
		nil,
		semconv.HTTPRoute("/users/{id}"),
		attribute.String("handler", "update_user"),
		attribute.String("operation", "update"),
		attribute.String("user.id", existingUser.ID().String()),
	)

	s.recordMetric(ctx, "update", "success")
	return dto.NewUserResponse(existingUser), nil
}

// DeleteUser removes a user by ID
func (s *UserService) DeleteUser(ctx context.Context, idStr string) error {
	ctx, span := s.tracer.Start(ctx, "UserService.DeleteUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("operation", "delete_user"),
		attribute.String("user.id", idStr),
	)

	telemetry.Log(ctx, telemetry.LevelInfo, "Deleting user",
		nil,
		semconv.HTTPRoute("/users/{id}"),
		attribute.String("handler", "delete_user"),
		attribute.String("operation", "delete"),
		attribute.String("user.id", idStr),
	)

	// Parse and validate ID
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		span.SetAttributes(attribute.String("error", "invalid_id"))
		s.recordMetric(ctx, "delete", "validation_error")
		return errors.ErrInvalidID.WithContext("id", idStr)
	}

	userID := entity.UserID(id)

	// Delete user from repository
	if err := s.repo.Delete(ctx, userID); err != nil {
		if errors.IsUserNotFound(err) {
			span.SetAttributes(attribute.String("error", "user_not_found"))
			s.recordMetric(ctx, "delete", "not_found")
			return err
		}
		span.SetAttributes(attribute.String("error", "repository_error"))
		s.recordMetric(ctx, "delete", "error")
		return errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to delete user", err)
	}

	telemetry.Log(ctx, telemetry.LevelInfo, "User deleted successfully",
		nil,
		semconv.HTTPRoute("/users/{id}"),
		attribute.String("handler", "delete_user"),
		attribute.String("operation", "delete"),
		attribute.String("user.id", idStr),
	)

	s.recordMetric(ctx, "delete", "success")
	return nil
}

// recordMetric records a metric for user operations
func (s *UserService) recordMetric(ctx context.Context, operation, status string) {
	if s.telemetry != nil && s.telemetry.UserCounter != nil {
		s.telemetry.UserCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("status", status),
		))
	}
}
