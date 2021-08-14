package auth

import (
	"context"
	"net/http"
	"strings"
)

type ctxKeyUser string

const UserContextKey ctxKeyUser = "user"

// JwtValidator middleware validates the Authorization header if there is one in the given request and
// associate the user in the request's context with the key UserContextKey.
//
// If no Authorization header was found or if the token is not valid, abort the request with a 403 status.
func JwtValidator(service Authorizer) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			authHeader := request.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				writer.WriteHeader(http.StatusUnauthorized)
				return
			}
			user, err := service.ValidateToken(ctx, authHeader)
			if err != nil {
				writer.WriteHeader(http.StatusUnauthorized)
				return
			}
			ctx = context.WithValue(ctx, UserContextKey, *user)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// AllowedRole middleware checks if the authenticated user has the given role.
//
// If there is no user authenticated or if the user doesn't have the given role, abort the request
// with a 403 status.
func AllowedRole(service Authorizer, role Role) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			user, err := service.GetAuthenticatedUser(request.Context())
			if err != nil {
				writer.WriteHeader(http.StatusUnauthorized)
				return
			}
			if user.Role != role {
				writer.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}
