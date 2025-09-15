package postgres

import (
	"context"
	"errors"

	"go-app/internal/domain/entity"
	domainErrors "go-app/internal/domain/errors"
	"go-app/internal/domain/repository"

	"gorm.io/gorm"
)

// PostgresUserRepository implements the UserRepository interface for PostgreSQL using GORM.
type PostgresUserRepository struct {
	db *gorm.DB
}

// NewPostgresUserRepository creates a new PostgresUserRepository.
func NewPostgresUserRepository(db *gorm.DB) repository.UserRepository {
	return &PostgresUserRepository{db: db}
}

// Create creates a new user in the database.
func (r *PostgresUserRepository) Create(ctx context.Context, user *entity.User) error {
	userModel := NewUserModelFromEntity(user)

	result := r.db.WithContext(ctx).Create(userModel)
	if result.Error != nil {
		return domainErrors.NewDomainErrorWithCause(domainErrors.ErrCodeRepositoryError, "failed to create user", result.Error)
	}

	// Set the generated ID back to the entity
	user.SetID(entity.UserID(userModel.ID))
	return nil
}

// GetByID retrieves a user by ID from the database.
func (r *PostgresUserRepository) GetByID(ctx context.Context, id entity.UserID) (*entity.User, error) {
	var userModel UserModel

	result := r.db.WithContext(ctx).First(&userModel, uint(id))
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrUserNotFound
		}
		return nil, domainErrors.NewDomainErrorWithCause(domainErrors.ErrCodeRepositoryError, "failed to get user by id", result.Error)
	}

	user, err := userModel.ToEntity()
	if err != nil {
		return nil, domainErrors.NewDomainErrorWithCause(domainErrors.ErrCodeInvalidUserData, "failed to convert model to entity", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email from the database.
func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email entity.Email) (*entity.User, error) {
	var userModel UserModel

	result := r.db.WithContext(ctx).Where("email = ?", email.String()).First(&userModel)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrUserNotFound
		}
		return nil, domainErrors.NewDomainErrorWithCause(domainErrors.ErrCodeRepositoryError, "failed to get user by email", result.Error)
	}

	user, err := userModel.ToEntity()
	if err != nil {
		return nil, domainErrors.NewDomainErrorWithCause(domainErrors.ErrCodeInvalidUserData, "failed to convert model to entity", err)
	}

	return user, nil
}

// List retrieves all users with optional pagination.
func (r *PostgresUserRepository) List(ctx context.Context, limit, offset int) ([]*entity.User, error) {
	var userModels []UserModel

	result := r.db.WithContext(ctx).
		Order("id ASC").
		Limit(limit).
		Offset(offset).
		Find(&userModels)

	if result.Error != nil {
		return nil, domainErrors.NewDomainErrorWithCause(domainErrors.ErrCodeRepositoryError, "failed to list users", result.Error)
	}

	users := make([]*entity.User, 0, len(userModels))
	for _, userModel := range userModels {
		user, err := userModel.ToEntity()
		if err != nil {
			return nil, domainErrors.NewDomainErrorWithCause(domainErrors.ErrCodeInvalidUserData, "failed to convert model to entity", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// Update updates an existing user in the database.
func (r *PostgresUserRepository) Update(ctx context.Context, user *entity.User) error {
	userModel := NewUserModelFromEntity(user)

	result := r.db.WithContext(ctx).
		Model(&userModel).
		Where("id = ?", uint(user.ID())).
		Updates(map[string]interface{}{
			"name":  user.Name().String(),
			"email": user.Email().String(),
		})

	if result.Error != nil {
		return domainErrors.NewDomainErrorWithCause(domainErrors.ErrCodeRepositoryError, "failed to update user", result.Error)
	}

	if result.RowsAffected == 0 {
		return domainErrors.ErrUserNotFound
	}

	return nil
}

// Delete removes a user by ID from the database.
func (r *PostgresUserRepository) Delete(ctx context.Context, id entity.UserID) error {
	result := r.db.WithContext(ctx).Delete(&UserModel{}, uint(id))
	if result.Error != nil {
		return domainErrors.NewDomainErrorWithCause(domainErrors.ErrCodeRepositoryError, "failed to delete user", result.Error)
	}

	if result.RowsAffected == 0 {
		return domainErrors.ErrUserNotFound
	}

	return nil
}

// ExistsByEmail checks if a user with the given email exists.
func (r *PostgresUserRepository) ExistsByEmail(ctx context.Context, email entity.Email) (bool, error) {
	var count int64

	result := r.db.WithContext(ctx).
		Model(&UserModel{}).
		Where("email = ?", email.String()).
		Count(&count)

	if result.Error != nil {
		return false, domainErrors.NewDomainErrorWithCause(domainErrors.ErrCodeRepositoryError, "failed to check if user exists by email", result.Error)
	}

	return count > 0, nil
}

// Count returns the total number of users.
func (r *PostgresUserRepository) Count(ctx context.Context) (int, error) {
	var count int64

	result := r.db.WithContext(ctx).Model(&UserModel{}).Count(&count)
	if result.Error != nil {
		return 0, domainErrors.NewDomainErrorWithCause(domainErrors.ErrCodeRepositoryError, "failed to count users", result.Error)
	}

	return int(count), nil
}
