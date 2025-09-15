package routes

import (
	"net/http"

	"go-app/internal/application/service"
	"go-app/internal/interface/http/handler"
)

// Router holds the router dependencies
type Router struct {
	userService *service.UserService
	appService  *service.AppService
}

// NewRouter creates a new router
func NewRouter(userService *service.UserService, appService *service.AppService) *Router {
	return &Router{
		userService: userService,
		appService:  appService,
	}
}

// RegisterRoutes registers all routes
func (r *Router) RegisterRoutes(mux *http.ServeMux) {
	// Create handlers
	rootHandler := handler.NewRootHandler(r.appService)
	usersHandler := handler.NewUsersHandler(r.userService)
	healthHandler := handler.NewHealthHandler()

	// Register routes
	mux.HandleFunc("/", rootHandler.Handle)
	mux.HandleFunc("/health", healthHandler.Handle)
	mux.HandleFunc("/users", usersHandler.Handle)
	mux.HandleFunc("/users/", usersHandler.Handle)
	mux.HandleFunc("/users/{id}", usersHandler.Handle)
}
