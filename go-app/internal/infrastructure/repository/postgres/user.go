package postgres

import (
	"context"
	"database/sql"

	"go-app/internal/domain/entity"
	"go-app/internal/domain/errors"
	"go-app/internal/domain/repository"
)

// PostgresUserRepository implements the UserRepository interface for PostgreSQL.
// It requires a 'users' table with the following schema:
// CREATE TABLE users (
//
//	id SERIAL PRIMARY KEY,
//	name VARCHAR(100) NOT NULL,
//	email VARCHAR(100) NOT NULL UNIQUE
//
// );
type PostgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository creates a new PostgresUserRepository.
func NewPostgresUserRepository(db *sql.DB) repository.UserRepository {
	return &PostgresUserRepository{db: db}
}

// Create creates a new user in the database.
func (r *PostgresUserRepository) Create(ctx context.Context, user *entity.User) error {
	query := "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id"
	var id entity.UserID
	err := r.db.QueryRowContext(ctx, query, user.Name().String(), user.Email().String()).Scan(&id)
	if err != nil {
		return errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to create user", err)
	}
	user.SetID(id)
	return nil
}

// GetByID retrieves a user by ID from the database.
func (r *PostgresUserRepository) GetByID(ctx context.Context, id entity.UserID) (*entity.User, error) {
	query := "SELECT id, name, email FROM users WHERE id = $1"
	row := r.db.QueryRowContext(ctx, query, int(id))

	var userID int
	var name, email string
	if err := row.Scan(&userID, &name, &email); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to get user by id", err)
	}

	user, err := entity.NewUser(name, email)
	if err != nil {
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeInvalidUserData, "failed to create user entity from db data", err)
	}
	user.SetID(entity.UserID(userID))

	return user, nil
}

// GetByEmail retrieves a user by email from the database.
func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email entity.Email) (*entity.User, error) {
	query := "SELECT id, name, email FROM users WHERE email = $1"
	row := r.db.QueryRowContext(ctx, query, email.String())

	var userID int
	var name, dbEmail string
	if err := row.Scan(&userID, &name, &dbEmail); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to get user by email", err)
	}

	user, err := entity.NewUser(name, dbEmail)
	if err != nil {
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeInvalidUserData, "failed to create user entity from db data", err)
	}
	user.SetID(entity.UserID(userID))

	return user, nil
}

// List retrieves all users with optional pagination.
func (r *PostgresUserRepository) List(ctx context.Context, limit, offset int) ([]*entity.User, error) {
	query := "SELECT id, name, email FROM users ORDER BY id LIMIT $1 OFFSET $2"
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to list users", err)
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		var userID int
		var name, email string
		if err := rows.Scan(&userID, &name, &email); err != nil {
			return nil, errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to scan user row", err)
		}

		user, err := entity.NewUser(name, email)
		if err != nil {
			return nil, errors.NewDomainErrorWithCause(errors.ErrCodeInvalidUserData, "failed to create user entity from db data", err)
		}
		user.SetID(entity.UserID(userID))
		users = append(users, user)
	}

	return users, nil
}

// Update updates an existing user in the database.
func (r *PostgresUserRepository) Update(ctx context.Context, user *entity.User) error {
	query := "UPDATE users SET name = $1, email = $2 WHERE id = $3"
	_, err := r.db.ExecContext(ctx, query, user.Name().String(), user.Email().String(), int(user.ID()))
	if err != nil {
		return errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to update user", err)
	}
	return nil
}

// Delete removes a user by ID from the database.
func (r *PostgresUserRepository) Delete(ctx context.Context, id entity.UserID) error {
	query := "DELETE FROM users WHERE id = $1"
	_, err := r.db.ExecContext(ctx, query, int(id))
	if err != nil {
		return errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to delete user", err)
	}
	return nil
}

// ExistsByEmail checks if a user with the given email exists.
func (r *PostgresUserRepository) ExistsByEmail(ctx context.Context, email entity.Email) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)"
	var exists bool
	if err := r.db.QueryRowContext(ctx, query, email.String()).Scan(&exists); err != nil {
		return false, errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to check if user exists by email", err)
	}
	return exists, nil
}

// Count returns the total number of users.
func (r *PostgresUserRepository) Count(ctx context.Context) (int, error) {
	query := "SELECT COUNT(*) FROM users"
	var count int
	if err := r.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, errors.NewDomainErrorWithCause(errors.ErrCodeRepositoryError, "failed to count users", err)
	}
	return count, nil
}
