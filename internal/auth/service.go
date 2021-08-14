package auth

import (
	"context"
	"fmt"
	"hospital-booking/internal/configs"
	"hospital-booking/internal/database"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/jwt"
)

// Authenticator determines the methods available to users get authenticated.
type Authenticator interface {

	// Authenticate authenticates a user by its credentials and returns a JWT tokens, otherwise an error.
	Authenticate(ctx context.Context, credentials Credentials) (*Tokens, error)
}

// Authorizer determines the methods used to authorize a user to perform some action.
type Authorizer interface {

	// ValidateToken validates the given token, returning the user associated to it.
	ValidateToken(ctx context.Context, token string) (*User, error)

	// RefreshTokens generates new tokens based on the given one.
	RefreshTokens(ctx context.Context, tokens Tokens) (*Tokens, error)

	// GetAuthenticatedUser gets the authenticated user associated to context.
	GetAuthenticatedUser(ctx context.Context) (User, error)
}

type Service interface {
	Authenticator
	Authorizer
}

type defaultService struct {
	repository Repository
	config     configs.Config
}

// NewService creates a new auth service.
func NewService(config configs.Config, dbConn database.Connection) Service {
	return &defaultService{
		config:     config,
		repository: newRepository(dbConn),
	}
}

func (d defaultService) Authenticate(ctx context.Context, credentials Credentials) (*Tokens, error) {
	if err := credentials.Validate(); err != nil {
		return nil, err
	}
	user, err := d.repository.FindUserByEmail(ctx, credentials.Email)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred: %w", err)
	}
	if user == nil {
		return nil, NewUnauthorizedError()
	}
	isValidCredentials, err := d.repository.CheckUserPassword(ctx, credentials.Email, credentials.Password)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred: %w", err)
	}
	if !isValidCredentials {
		return nil, NewUnauthorizedError()
	}
	return GenerateTokens(ctx, d.config.PrivateKey(), *user)
}

func (d defaultService) ValidateToken(ctx context.Context, token string) (*User, error) {
	bearer := strings.TrimPrefix(token, "Bearer ")
	parsedToken, err := ParseToken(bearer, d.config.PrivateKey().PublicKey)
	if err != nil {
		return nil, NewUnauthorizedError()
	}
	if !time.Now().Before(parsedToken.Expiration()) {
		return nil, NewUnauthorizedError()
	}
	user, err := d.repository.FindUserByUUID(ctx, uuid.MustParse(parsedToken.Subject()))
	if err != nil {
		return nil, NewUnauthorizedError()
	}
	if user == nil {
		return nil, NewUnauthorizedError()
	}
	return user, nil
}

func (d defaultService) RefreshTokens(ctx context.Context, tokens Tokens) (*Tokens, error) {
	if err := tokens.Validate(); err != nil {
		return nil, err
	}
	refreshToken, err := jwt.ParseString(tokens.RefreshToken)
	if err != nil {
		return nil, NewUnauthorizedError()
	}
	if !time.Now().Before(refreshToken.Expiration()) {
		return nil, NewUnauthorizedError()
	}
	user, err := d.repository.FindUserByUUID(ctx, uuid.MustParse(refreshToken.Subject()))
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred: %w", err)
	}
	if user == nil {
		return nil, NewUnauthorizedError()
	}
	return GenerateTokens(ctx, d.config.PrivateKey(), *user)
}

func (d defaultService) GetAuthenticatedUser(ctx context.Context) (User, error) {
	user, isUser := ctx.Value(UserContextKey).(User)
	if !isUser {
		return User{}, NewUnauthorizedError()
	}
	return user, nil
}
