package auth

import (
	"encoding/json"
	"fmt"
	"hospital-booking/internal/apierrors"
	"hospital-booking/internal/configs"
	"hospital-booking/internal/database"
	"hospital-booking/internal/logging"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/go-chi/chi/v5"
)

type httpHandler struct {
	service Service
	logger  *log.Logger
}

// Setup setups the routes handled by auth context.
func Setup(router *chi.Mux, logger *log.Logger, config configs.Config, dbConn database.Connection) {
	handler := &httpHandler{logger: logger, service: NewService(config, dbConn)}

	// public routes
	router.Group(func(group chi.Router) {
		group.Post("/api/v1/auth/login", handler.Authenticate)
		group.Put("/api/v1/auth/token", handler.RefreshToken)
	})

	// protected routes
	router.Group(func(group chi.Router) {
		group.Use(JwtValidator(handler.service))
		group.Get("/api/v1/auth/me", handler.GetAuthenticatedUser)
	})
}

// Authenticate handles the request to authenticate a user.
func (h httpHandler) Authenticate(w http.ResponseWriter, r *http.Request) {
	credentials := new(Credentials)
	if err := json.NewDecoder(r.Body).Decode(credentials); err != nil {
		logging.PrintlnError(h.logger, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	tokens, err := h.service.Authenticate(r.Context(), *credentials)
	if err != nil {
		logging.PrintlnError(h.logger, fmt.Sprint(r.Context().Value(middleware.RequestIDKey), " ", err))
		switch err.(type) {
		case *UnauthorizedError:
			w.WriteHeader(http.StatusUnauthorized)
			return
		case *apierrors.ValidationError:
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(err)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(tokens)
}

// RefreshToken handles the request to return a new refresh token to the authenticated user.
func (h httpHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	tokens := new(Tokens)
	if err := json.NewDecoder(r.Body).Decode(tokens); err != nil {
		logging.PrintlnError(h.logger, fmt.Sprint(r.Context().Value(middleware.RequestIDKey), " ", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	tokens, err := h.service.RefreshTokens(r.Context(), *tokens)
	if err != nil {
		switch err.(type) {
		case *UnauthorizedError:
			w.WriteHeader(http.StatusUnauthorized)
			return
		case *apierrors.ValidationError:
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(err)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(tokens)
}

// GetAuthenticatedUser handles the request to return data about the authenticated user.
func (h httpHandler) GetAuthenticatedUser(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.GetAuthenticatedUser(r.Context())
	if err != nil {
		logging.PrintlnError(h.logger, fmt.Sprint(r.Context().Value(middleware.RequestIDKey), " ", err))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	_ = json.NewEncoder(w).Encode(user)
}
