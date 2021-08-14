package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestAllowedRole(t *testing.T) {
	type args struct {
		service Authorizer
		role    Role
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should allow the request",
			args: args{
				service: mockAuthorizer{
					mockGetAuthenticatedUser: func(ctx context.Context) (User, error) {
						return User{Email: "patient@hostpital.com", Role: PatientRole}, nil
					},
				},
				role: PatientRole,
			},
			want: http.StatusOK,
		},
		{
			name: "should not allow the request due to a wrong role",
			args: args{
				service: mockAuthorizer{
					mockGetAuthenticatedUser: func(ctx context.Context) (User, error) {
						return User{Email: "patient@hostpital.com", Role: DoctorRole}, nil
					},
				},
				role: PatientRole,
			},
			want: http.StatusForbidden,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			router.Use(AllowedRole(tt.args.service, tt.args.role))
			router.Get("/", func(w http.ResponseWriter, r *http.Request) {})

			req, _ := http.NewRequest("GET", "/", nil)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			response := recorder.Result()

			if response.StatusCode != tt.want {
				t.Errorf("response status is incorrect, got %d, want %d", recorder.Code, tt.want)
			}
		})
	}
}

func TestJwtValidator(t *testing.T) {
	type args struct {
		service    Authorizer
		authHeader string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should allow the request and return status 200",
			args: args{
				service: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*User, error) {
						return &User{Email: "patient@hostpital.com", Role: PatientRole}, nil
					},
				},
				authHeader: "Bearer testing",
			},
			want: http.StatusOK,
		},
		{
			name: "should not allow the request and return status 401 due to a invalid header",
			args: args{
				service: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*User, error) {
						return &User{Email: "patient@hostpital.com", Role: PatientRole}, nil
					},
				},
				authHeader: "",
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "should not allow the request and return status 401 due to missing user",
			args: args{
				service: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*User, error) {
						return nil, NewUnauthorizedError()
					},
				},
				authHeader: "Bearer testing",
			},
			want: http.StatusUnauthorized,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			router.Use(JwtValidator(tt.args.service))
			router.Get("/", func(w http.ResponseWriter, r *http.Request) {})

			req, _ := http.NewRequest("GET", "/", nil)
			req.Header.Add("Authorization", tt.args.authHeader)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			response := recorder.Result()

			if response.StatusCode != tt.want {
				t.Errorf("response status is incorrect, got %d, want %d", recorder.Code, tt.want)
			}
		})
	}
}
