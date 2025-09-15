package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"go-app/internal/application/dto"
	"go-app/internal/application/service"
	domainErrors "go-app/internal/domain/errors"
	"go-app/internal/infrastructure/telemetry"
)

// UsersHandler handles requests to the users endpoint
type UsersHandler struct {
	userService *service.UserService
}

// NewUsersHandler creates a new users handler
func NewUsersHandler(userService *service.UserService) *UsersHandler {
	return &UsersHandler{
		userService: userService,
	}
}

// Handle handles requests to the users endpoint
func (h *UsersHandler) Handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// If there's an ID parameter, get a specific user
		// Otherwise, list all users
		if id := r.PathValue("id"); id != "" {
			h.getUserByID(w, r, id)
		} else {
			h.listUsers(w, r)
		}
	case http.MethodPost:
		h.createUser(w, r)
	case http.MethodPut:
		h.updateUser(w, r)
	case http.MethodDelete:
		h.deleteUser(w, r)
	default:
		h.writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
	}
}

// listUsers handles GET requests to list all users
func (h *UsersHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Add attributes to the current span
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("http.route", "/users"),
		attribute.String("handler", "users"),
		attribute.String("operation", "list"),
	)

	// Parse query parameters for pagination
	limit := 10 // default
	offset := 0 // default

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Create request DTO
	req := dto.ListUsersRequest{
		Limit:  limit,
		Offset: offset,
	}

	// Get users from user service
	response, err := h.userService.ListUsers(ctx, req)
	if err != nil {
		telemetry.Log(ctx, telemetry.LevelError, "Failed to get users", err,
			attribute.String("handler", "users"),
			attribute.String("path", "/users"),
		)
		h.writeErrorResponseFromDomainError(w, err)
		return
	}

	h.writeJSONResponse(w, response, http.StatusOK)
}

// getUserByID handles GET requests to get a specific user by ID
func (h *UsersHandler) getUserByID(w http.ResponseWriter, r *http.Request, idStr string) {
	ctx := r.Context()

	// Add attributes to the current span
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("http.route", "/users/{id}"),
		attribute.String("handler", "users"),
		attribute.String("operation", "get"),
		attribute.String("user.id", idStr),
	)

	// Get user from user service
	user, err := h.userService.GetUserByID(ctx, idStr)
	if err != nil {
		telemetry.Log(ctx, telemetry.LevelError, "Failed to get user", err,
			attribute.String("handler", "users"),
			attribute.String("path", "/users/"+idStr),
			attribute.String("user.id", idStr),
		)
		h.writeErrorResponseFromDomainError(w, err)
		return
	}

	h.writeJSONResponse(w, user, http.StatusOK)
}

// createUser handles POST requests to create a new user
func (h *UsersHandler) createUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req dto.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, "Invalid JSON", http.StatusBadRequest, "INVALID_JSON")
		return
	}

	// Add attributes to the current span
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("http.route", "/users"),
		attribute.String("handler", "users"),
		attribute.String("operation", "create"),
		attribute.String("user.email", req.Email),
		attribute.String("user.name", req.Name),
	)

	// Create user through user service
	user, err := h.userService.CreateUser(ctx, req)
	if err != nil {
		telemetry.Log(ctx, telemetry.LevelError, "Failed to create user", err,
			attribute.String("handler", "users"),
			attribute.String("path", "/users"),
			attribute.String("user.email", req.Email),
			attribute.String("user.name", req.Name),
		)
		h.writeErrorResponseFromDomainError(w, err)
		return
	}

	// Create success response
	response := dto.SuccessResponse{
		Message: "User created successfully",
		Data:    user,
	}

	h.writeJSONResponse(w, response, http.StatusCreated)
}

// updateUser handles PUT requests to update an existing user
func (h *UsersHandler) updateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse user ID from path
	idStr := r.PathValue("id")
	if idStr == "" {
		h.writeErrorResponse(w, "User ID is required", http.StatusBadRequest, "MISSING_USER_ID")
		return
	}

	// Parse request body
	var req dto.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, "Invalid JSON", http.StatusBadRequest, "INVALID_JSON")
		return
	}

	// Add attributes to the current span
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("http.route", "/users/{id}"),
		attribute.String("handler", "users"),
		attribute.String("operation", "update"),
		attribute.String("user.id", idStr),
		attribute.String("user.email", req.Email),
		attribute.String("user.name", req.Name),
	)

	// Update user through user service
	user, err := h.userService.UpdateUser(ctx, idStr, req)
	if err != nil {
		telemetry.Log(ctx, telemetry.LevelError, "Failed to update user", err,
			attribute.String("handler", "users"),
			attribute.String("path", "/users/"+idStr),
			attribute.String("user.id", idStr),
			attribute.String("user.email", req.Email),
			attribute.String("user.name", req.Name),
		)
		h.writeErrorResponseFromDomainError(w, err)
		return
	}

	// Create success response
	response := dto.SuccessResponse{
		Message: "User updated successfully",
		Data:    user,
	}

	h.writeJSONResponse(w, response, http.StatusOK)
}

// deleteUser handles DELETE requests to remove a user
func (h *UsersHandler) deleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse user ID from path
	idStr := r.PathValue("id")
	if idStr == "" {
		h.writeErrorResponse(w, "User ID is required", http.StatusBadRequest, "MISSING_USER_ID")
		return
	}

	// Add attributes to the current span
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("http.route", "/users/{id}"),
		attribute.String("handler", "users"),
		attribute.String("operation", "delete"),
		attribute.String("user.id", idStr),
	)

	// Delete user through user service
	err := h.userService.DeleteUser(ctx, idStr)
	if err != nil {
		telemetry.Log(ctx, telemetry.LevelError, "Failed to delete user", err,
			attribute.String("handler", "users"),
			attribute.String("path", "/users/"+idStr),
			attribute.String("user.id", idStr),
		)
		h.writeErrorResponseFromDomainError(w, err)
		return
	}

	// Create success response
	response := dto.SuccessResponse{
		Message: "User deleted successfully",
	}

	h.writeJSONResponse(w, response, http.StatusOK)
}

// writeJSONResponse writes a JSON response
func (h *UsersHandler) writeJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error but don't change response since headers are already written
		telemetry.Log(context.Background(), telemetry.LevelError, "Failed to encode JSON response", err)
	}
}

// writeErrorResponse writes an error response
func (h *UsersHandler) writeErrorResponse(w http.ResponseWriter, message string, statusCode int, code string) {
	errorResp := dto.ErrorResponse{
		Error:   message,
		Code:    code,
		Message: message,
	}
	h.writeJSONResponse(w, errorResp, statusCode)
}

// writeErrorResponseFromDomainError writes an error response from a domain error
func (h *UsersHandler) writeErrorResponseFromDomainError(w http.ResponseWriter, err error) {
	var domainErr *domainErrors.DomainError
	var statusCode int
	var errorResp dto.ErrorResponse

	if errors.As(err, &domainErr) {
		// Map domain error codes to HTTP status codes
		switch domainErr.Code {
		case domainErrors.ErrCodeUserNotFound:
			statusCode = http.StatusNotFound
		case domainErrors.ErrCodeUserAlreadyExists:
			statusCode = http.StatusConflict
		case domainErrors.ErrCodeValidationFailed, domainErrors.ErrCodeInvalidUserData,
			domainErrors.ErrCodeInvalidEmail, domainErrors.ErrCodeInvalidName, domainErrors.ErrCodeInvalidID:
			statusCode = http.StatusBadRequest
		case domainErrors.ErrCodeRepositoryError, domainErrors.ErrCodeDatabaseError:
			statusCode = http.StatusInternalServerError
		default:
			statusCode = http.StatusInternalServerError
		}

		errorResp = dto.ErrorResponse{
			Error:   domainErr.Error(),
			Code:    string(domainErr.Code),
			Message: domainErr.Message,
			Context: domainErr.Context,
		}
	} else {
		// Generic error
		statusCode = http.StatusInternalServerError
		errorResp = dto.ErrorResponse{
			Error:   err.Error(),
			Code:    "INTERNAL_ERROR",
			Message: "An internal error occurred",
		}
	}

	h.writeJSONResponse(w, errorResp, statusCode)
}
