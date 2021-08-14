package auth

import (
	"hospital-booking/internal/apierrors"

	"github.com/google/uuid"
)

type Role string

const (
	PatientRole = "PATIENT"
	DoctorRole  = "DOCTOR"
)

type Credentials struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
}

// Validate validates if the credentials given are valid.
func (c Credentials) Validate() error {
	if c.Email == "" {
		return apierrors.NewValidationError("email", "required")
	}
	if c.Password == "" {
		return apierrors.NewValidationError("password", "required")
	}
	return nil
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	GrantType    string `json:"grant_type,omitempty"`
}

// Validate validates if the tokens given are valid.
func (c Tokens) Validate() error {
	if c.AccessToken == "" {
		return apierrors.NewValidationError("access_token", "required")
	}
	if c.RefreshToken == "" {
		return apierrors.NewValidationError("refresh_token", "required")
	}
	if c.GrantType == "" {
		return apierrors.NewValidationError("grant_type", "required")
	}
	if c.GrantType != "refresh_token" {
		return apierrors.NewValidationError("grant_type", "invalid")
	}
	return nil
}

type User struct {
	ID       int64     `json:"-" dbfield:"id"`
	UUID     uuid.UUID `json:"uuid" dbfield:"uuid"`
	Email    string    `json:"email" dbfield:"email"`
	Password string    `json:"password,omitempty" dbfield:"password"`
	Role     Role      `json:"role" dbfield:"role"`
}
