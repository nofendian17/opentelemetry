# Clean Architecture Documentation

This Go application follows Clean Architecture principles with clear separation of concerns and proper dependency
inversion.

## Architecture Overview

```
go-app/
├── internal/
│   ├── domain/           # Business logic layer (innermost)
│   │   ├── entity/       # Core business entities
│   │   ├── repository/   # Repository interfaces
│   │   ├── service/      # Domain service interfaces
│   │   └── errors/       # Domain-specific errors
│   ├── application/      # Application layer
│   │   ├── service/      # Application services (use cases)
│   │   └── dto/          # Data Transfer Objects
│   ├── infrastructure/   # Infrastructure layer (outermost)
│   │   ├── config/       # Configuration management
│   │   ├── repository/   # Repository implementations
│   │   │   └── memory/   # In-memory implementations
│   │   └── telemetry/    # Observability (logging, tracing, metrics)
│   └── interface/        # Interface adapters
│       └── http/         # HTTP handlers, routes, middleware
│           ├── handler/  # HTTP request handlers
│           ├── middleware/ # HTTP middleware
│           └── routes/   # Route definitions
└── main.go              # Application entry point
```

## Layer Responsibilities

### Domain Layer (Core Business Logic)

- **Entities**: Core business objects with behavior and business rules
- **Value Objects**: Immutable objects representing domain concepts
- **Repository Interfaces**: Define data access contracts
- **Domain Services**: Business logic that doesn't naturally fit in entities
- **Domain Errors**: Structured error handling for business rules

**Key Principles:**

- No dependencies on external layers
- Contains pure business logic
- Framework and technology agnostic

### Application Layer (Use Cases)

- **Application Services**: Orchestrate business workflows
- **DTOs**: Data transfer between layers
- **Input Validation**: Request validation and sanitization
- **Transaction Management**: Coordinate repository operations

**Key Principles:**

- Depends only on domain layer
- Implements specific use cases
- Handles application-specific logic

### Infrastructure Layer (External Dependencies)

- **Repository Implementations**: Data persistence logic
- **Configuration**: Environment and application settings
- **Telemetry**: Logging, tracing, and metrics
- **External Services**: Third-party integrations

**Key Principles:**

- Implements interfaces from inner layers
- Contains framework-specific code
- Handles external dependencies

### Interface Layer (Adapters)

- **HTTP Handlers**: Convert HTTP requests to application calls
- **Middleware**: Cross-cutting concerns (logging, auth, CORS)
- **Routes**: URL routing and handler mapping
- **Response Formatting**: Convert domain objects to API responses

**Key Principles:**

- Adapts external interfaces to internal use
- Handles protocol-specific concerns
- Maps between external and internal models

## Key Design Patterns

### Dependency Inversion

All dependencies point inward toward the domain layer:

```go
// Domain defines interface
type UserRepository interface {
Save(ctx context.Context, user *User) error
}

// Infrastructure implements interface
type MemoryUserRepository struct {
// implementation
}

func (r *MemoryUserRepository) Save(ctx context.Context, user *User) error {
// concrete implementation
}
```

### Repository Pattern

Abstract data access through interfaces:

```go
// Domain layer
type UserRepository interface {
GetByID(ctx context.Context, id UserID) (*User, error)
Save(ctx context.Context, user *User) error
}

// Infrastructure layer
type MemoryUserRepository struct {
users map[string]*User
tracer trace.Tracer
}
```

### Service Pattern

Encapsulate business logic in services:

```go
// Application layer
type UserService struct {
repo UserRepository
telemetry *telemetry.Telemetry
}

func (s *UserService) CreateUser(ctx context.Context, req dto.CreateUserRequest) (*dto.UserResponse, error) {
// Application logic
}
```

### DTO Pattern

Transfer data between layers safely:

```go
type CreateUserRequest struct {
Name  string `json:"name"`
Email string `json:"email"`
}

func (r CreateUserRequest) Validate() error {
// Validation logic
}
```

## Error Handling

### Domain Errors

Structured error handling with specific error codes:

```go
type DomainError struct {
Code    ErrorCode
Message string
Context map[string]interface{}
Cause   error
}

const (
ErrCodeUserNotFound ErrorCode = "USER_NOT_FOUND"
ErrCodeInvalidEmail ErrorCode = "INVALID_EMAIL"
)
```

### HTTP Error Mapping

Domain errors are mapped to appropriate HTTP status codes:

```go
switch domainErr.Code {
case errors.ErrCodeUserNotFound:
statusCode = http.StatusNotFound
case errors.ErrCodeUserAlreadyExists:
statusCode = http.StatusConflict
case errors.ErrCodeValidationFailed:
statusCode = http.StatusBadRequest
}
```

## Value Objects

### Strong Typing

Domain concepts are represented as typed values:

```go
type UserID struct {
value int64
}

type Email struct {
value string
}

type Name struct {
value string
}
```

### Validation

Business rules are enforced at the value object level:

```go
func NewEmail(email string) (Email, error) {
if !isValidEmail(email) {
return Email{}, errors.New("invalid email format")
}
return Email{value: email}, nil
}
```

## Observability

### Structured Logging

Context-aware logging with attributes:

```go
telemetry.Log(ctx, telemetry.LevelInfo, "User created", nil,
attribute.String("user.id", user.ID().String()),
attribute.String("user.email", user.Email().String()),
)
```

### Distributed Tracing

OpenTelemetry integration for request tracing:

```go
ctx, span := s.tracer.Start(ctx, "UserService.CreateUser")
defer span.End()

span.SetAttributes(
attribute.String("operation", "create_user"),
attribute.String("user.email", req.Email),
)
```

## Testing Strategy

### Unit Tests

- Test business logic in isolation
- Mock external dependencies
- Focus on domain entities and services

### Integration Tests

- Test layer interactions
- Use real implementations where possible
- Test complete workflows

### Repository Tests

- Test data access logic
- Use test databases or in-memory stores
- Verify data integrity

## Benefits

### Maintainability

- Clear separation of concerns
- Testable architecture
- Easy to understand and modify

### Scalability

- Loose coupling between layers
- Easy to swap implementations
- Supports different deployment strategies

### Testing

- Business logic is isolated and testable
- Dependencies can be easily mocked
- Clear test boundaries

### Technology Independence

- Core business logic is framework agnostic
- Easy to migrate between technologies
- Reduced vendor lock-in

## Best Practices

1. **Keep the Domain Pure**: No external dependencies in domain layer
2. **Use Interfaces**: Define contracts for external dependencies
3. **Validate Early**: Input validation at application layer boundaries
4. **Handle Errors Properly**: Use structured error handling
5. **Test Business Logic**: Focus testing on domain entities and services
6. **Monitor Everything**: Use observability for debugging and monitoring
7. **Document Decisions**: Keep architecture decisions documented
8. **Review Dependencies**: Regularly review and minimize dependencies

## Migration Guide

When adding new features:

1. **Start with Domain**: Define entities and business rules
2. **Add Repository Interface**: Define data access needs
3. **Implement Application Service**: Orchestrate the use case
4. **Add DTOs**: Define input/output contracts
5. **Implement Repository**: Add data persistence
6. **Add HTTP Handler**: Expose via API
7. **Add Tests**: Ensure quality and coverage

This architecture provides a solid foundation for building maintainable, testable, and scalable applications.
