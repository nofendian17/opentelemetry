package entity

import (
	"fmt"
	"regexp"
	"strings"

	"go-app/internal/domain/errors"
)

// UserID represents a unique identifier for a user
type UserID int

// IsValid checks if the UserID is valid
func (id UserID) IsValid() bool {
	return id > 0
}

// String returns string representation of UserID
func (id UserID) String() string {
	return fmt.Sprintf("%d", int(id))
}

// Email represents a validated email address
type Email string

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// NewEmail creates a new Email after validation
func NewEmail(email string) (Email, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return "", errors.NewDomainError(errors.ErrCodeInvalidEmail, "email cannot be empty")
	}
	if !emailRegex.MatchString(email) {
		return "", errors.ErrInvalidEmail
	}
	return Email(email), nil
}

// String returns the string representation of the email
func (e Email) String() string {
	return string(e)
}

// IsValid checks if the email is valid
func (e Email) IsValid() bool {
	return emailRegex.MatchString(string(e))
}

// Name represents a user's name with validation
type Name string

// NewName creates a new Name after validation
func NewName(name string) (Name, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.NewDomainError(errors.ErrCodeInvalidName, "name cannot be empty")
	}
	if len(name) < 2 {
		return "", errors.NewDomainError(errors.ErrCodeInvalidName, "name must be at least 2 characters long")
	}
	if len(name) > 100 {
		return "", errors.NewDomainError(errors.ErrCodeInvalidName, "name cannot exceed 100 characters")
	}
	return Name(name), nil
}

// String returns the string representation of the name
func (n Name) String() string {
	return string(n)
}

// User represents a user entity in the domain
type User struct {
	id    UserID
	name  Name
	email Email
}

// NewUser creates a new User with validation
func NewUser(name, email string) (*User, error) {
	userName, err := NewName(name)
	if err != nil {
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeInvalidUserData, "invalid name for new user", err)
	}

	userEmail, err := NewEmail(email)
	if err != nil {
		return nil, errors.NewDomainErrorWithCause(errors.ErrCodeInvalidUserData, "invalid email for new user", err)
	}

	return &User{
		name:  userName,
		email: userEmail,
	}, nil
}

// ID returns the user's ID
func (u *User) ID() UserID {
	return u.id
}

// Name returns the user's name
func (u *User) Name() Name {
	return u.name
}

// Email returns the user's email
func (u *User) Email() Email {
	return u.email
}

// SetID sets the user's ID (used by repository layer)
func (u *User) SetID(id UserID) {
	u.id = id
}

// UpdateName updates the user's name with validation
func (u *User) UpdateName(name string) error {
	userName, err := NewName(name)
	if err != nil {
		return errors.NewDomainErrorWithCause(errors.ErrCodeInvalidUserData, "invalid name for update", err)
	}
	u.name = userName
	return nil
}

// UpdateEmail updates the user's email with validation
func (u *User) UpdateEmail(email string) error {
	userEmail, err := NewEmail(email)
	if err != nil {
		return errors.NewDomainErrorWithCause(errors.ErrCodeInvalidUserData, "invalid email for update", err)
	}
	u.email = userEmail
	return nil
}

// Equals checks if two users are equal based on their ID
func (u *User) Equals(other *User) bool {
	if other == nil {
		return false
	}
	return u.id == other.id && u.id.IsValid()
}
