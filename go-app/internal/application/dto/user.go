package dto

import (
	"errors"
	"strings"

	"go-app/internal/domain/entity"
)

// CreateUserRequest represents the request to create a user
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Validate validates the CreateUserRequest
func (r *CreateUserRequest) Validate() error {
	return validateUserPayload(r.Name, r.Email)
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Validate validates the UpdateUserRequest
func (r *UpdateUserRequest) Validate() error {
	return validateUserPayload(r.Name, r.Email)
}

// validateUserPayload provides shared validation for user requests
func validateUserPayload(name, email string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(email) == "" {
		return errors.New("email is required")
	}
	return nil
}

// ListUsersRequest represents the request to list users with pagination
type ListUsersRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// Validate validates the ListUsersRequest
func (r *ListUsersRequest) Validate() error {
	if r.Limit <= 0 {
		r.Limit = 10 // Default limit
	}
	if r.Limit > 100 {
		return errors.New("limit cannot exceed 100")
	}
	if r.Offset < 0 {
		return errors.New("offset cannot be negative")
	}
	return nil
}

// UserResponse represents the response when returning user data
type UserResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// NewUserResponse creates a UserResponse from a domain entity
func NewUserResponse(user *entity.User) *UserResponse {
	return &UserResponse{
		ID:    int(user.ID()),
		Name:  user.Name().String(),
		Email: user.Email().String(),
	}
}

// ListUsersResponse represents the response when listing users
type ListUsersResponse struct {
	Users      []*UserResponse `json:"users"`
	Total      int             `json:"total"`
	Limit      int             `json:"limit"`
	Offset     int             `json:"offset"`
	HasMore    bool            `json:"has_more"`
	NextOffset *int            `json:"next_offset,omitempty"`
}

// NewListUsersResponse creates a ListUsersResponse from domain entities
func NewListUsersResponse(users []*entity.User, total, limit, offset int) *ListUsersResponse {
	userResponses := make([]*UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = NewUserResponse(user)
	}

	hasMore := offset+len(users) < total
	var nextOffset *int
	if hasMore {
		next := offset + limit
		nextOffset = &next
	}

	return &ListUsersResponse{
		Users:      userResponses,
		Total:      total,
		Limit:      limit,
		Offset:     offset,
		HasMore:    hasMore,
		NextOffset: nextOffset,
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    string                 `json:"code,omitempty"`
	Message string                 `json:"message"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
