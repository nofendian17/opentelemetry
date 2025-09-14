package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	models "go-app/internal/domain/model"
	"go-app/internal/domain/repository"
	"go-app/internal/infrastructure/telemetry"
	"go-app/internal/usecase"
)

// UsersHandler handles requests to the users endpoint
type UsersHandler struct {
	userService *usecase.UserUseCase
}

// NewUsersHandler creates a new users handler
func NewUsersHandler(userService *usecase.UserUseCase) *UsersHandler {
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listUsers handles GET requests to list all users
func (h *UsersHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	// Create a context with the current request
	ctx := r.Context()

	// Add attributes to the current span
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("http.route", "/users"),
		attribute.String("handler", "users"),
		attribute.String("operation", "list"),
	)

	// Get users from user service
	users, err := h.userService.List(ctx)
	if err != nil {
		telemetry.Log(ctx, telemetry.LevelError, "Failed to get users", err,
			attribute.String("handler", "users"),
			attribute.String("path", "/users"),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to map for JSON response
	usersMap := make([]map[string]interface{}, len(users))
	for i, user := range users {
		usersMap[i] = map[string]interface{}{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		}
	}

	// Respond with JSON
	response := map[string]interface{}{
		"users":  usersMap,
		"count":  len(users),
		"path":   "/users",
		"method": r.Method,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		telemetry.Log(ctx, telemetry.LevelError, "Failed to encode response", err,
			attribute.String("handler", "users"),
			attribute.String("path", "/users"),
		)
	}
}

// getUserByID handles GET requests to get a specific user by ID
func (h *UsersHandler) getUserByID(w http.ResponseWriter, r *http.Request, idStr string) {
	// Create a context with the current request
	ctx := r.Context()

	// Parse user ID
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Add attributes to the current span
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("http.route", "/users/{id}"),
		attribute.String("handler", "users"),
		attribute.String("operation", "get"),
		attribute.Int("user.id", id),
	)

	// Get user from user service
	user, err := h.userService.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrUserNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		telemetry.Log(ctx, telemetry.LevelError, "Failed to get user", err,
			attribute.String("handler", "users"),
			attribute.String("path", "/users/"+idStr),
			attribute.Int("user.id", id),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with JSON
	response := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
		"path":   "/users/" + idStr,
		"method": r.Method,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		telemetry.Log(ctx, telemetry.LevelError, "Failed to encode response", err,
			attribute.String("handler", "users"),
			attribute.String("path", "/users/"+idStr),
			attribute.Int("user.id", id),
		)
	}
}

// createUser handles POST requests to create a new user
func (h *UsersHandler) createUser(w http.ResponseWriter, r *http.Request) {
	// Create a context with the current request
	ctx := r.Context()

	// Parse request body
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Add attributes to the current span
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("http.route", "/users"),
		attribute.String("handler", "users"),
		attribute.String("operation", "create"),
		attribute.String("user.email", user.Email),
		attribute.String("user.name", user.Name),
	)

	// Create user through user service
	err := h.userService.Create(ctx, &user)
	if err != nil {
		if err == repository.ErrUserAlreadyExists {
			http.Error(w, "User already exists", http.StatusConflict)
			return
		}
		telemetry.Log(ctx, telemetry.LevelError, "Failed to create user", err,
			attribute.String("handler", "users"),
			attribute.String("path", "/users"),
			attribute.String("user.email", user.Email),
			attribute.String("user.name", user.Name),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with JSON
	response := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
		"message": "User created successfully",
		"path":    "/users",
		"method":  r.Method,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		telemetry.Log(ctx, telemetry.LevelError, "Failed to encode response", err,
			attribute.String("handler", "users"),
			attribute.String("path", "/users"),
			attribute.String("user.email", user.Email),
			attribute.String("user.name", user.Name),
		)
	}
}

// updateUser handles PUT requests to update an existing user
func (h *UsersHandler) updateUser(w http.ResponseWriter, r *http.Request) {
	// Create a context with the current request
	ctx := r.Context()

	// Parse user ID from path
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set the user ID from the path
	user.ID = id

	// Add attributes to the current span
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("http.route", "/users/{id}"),
		attribute.String("handler", "users"),
		attribute.String("operation", "update"),
		attribute.Int("user.id", id),
		attribute.String("user.email", user.Email),
		attribute.String("user.name", user.Name),
	)

	// Update user through user service
	err = h.userService.Update(ctx, &user)
	if err != nil {
		if err == repository.ErrUserNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		if err == repository.ErrUserAlreadyExists {
			http.Error(w, "User with this email already exists", http.StatusConflict)
			return
		}
		telemetry.Log(ctx, telemetry.LevelError, "Failed to update user", err,
			attribute.String("handler", "users"),
			attribute.String("path", "/users/"+idStr),
			attribute.Int("user.id", id),
			attribute.String("user.email", user.Email),
			attribute.String("user.name", user.Name),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with JSON
	response := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
		"message": "User updated successfully",
		"path":    "/users/" + idStr,
		"method":  r.Method,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		telemetry.Log(ctx, telemetry.LevelError, "Failed to encode response", err,
			attribute.String("handler", "users"),
			attribute.String("path", "/users/"+idStr),
			attribute.Int("user.id", id),
			attribute.String("user.email", user.Email),
			attribute.String("user.name", user.Name),
		)
	}
}

// deleteUser handles DELETE requests to remove a user
func (h *UsersHandler) deleteUser(w http.ResponseWriter, r *http.Request) {
	// Create a context with the current request
	ctx := r.Context()

	// Parse user ID from path
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Add attributes to the current span
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("http.route", "/users/{id}"),
		attribute.String("handler", "users"),
		attribute.String("operation", "delete"),
		attribute.Int("user.id", id),
	)

	// Delete user through user service
	err = h.userService.Delete(ctx, id)
	if err != nil {
		if err == repository.ErrUserNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		telemetry.Log(ctx, telemetry.LevelError, "Failed to delete user", err,
			attribute.String("handler", "users"),
			attribute.String("path", "/users/"+idStr),
			attribute.Int("user.id", id),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with JSON
	response := map[string]interface{}{
		"message": "User deleted successfully",
		"path":    "/users/" + idStr,
		"method":  r.Method,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		telemetry.Log(ctx, telemetry.LevelError, "Failed to encode response", err,
			attribute.String("handler", "users"),
			attribute.String("path", "/users/"+idStr),
			attribute.Int("user.id", id),
		)
	}
}
