package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"hospital-booking/internal/configs"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/jwt"
)

type emptyWriter struct{}

func (e emptyWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}

var logger = log.New(&emptyWriter{}, "", log.LstdFlags)

type mockConnection struct {
	db   *sql.DB
	mock sqlmock.Sqlmock
}

func (m mockConnection) DB() *sql.DB {
	return m.db
}

func (m mockConnection) Close() {
	_ = m.DB().Close()
}

type mockAuthorizer struct {
	mockValidateToken        func(ctx context.Context, token string) (*User, error)
	mockRefreshTokens        func(ctx context.Context, tokens Tokens) (*Tokens, error)
	mockGetAuthenticatedUser func(ctx context.Context) (User, error)
}

func (m mockAuthorizer) ValidateToken(ctx context.Context, token string) (*User, error) {
	return m.mockValidateToken(ctx, token)
}

func (m mockAuthorizer) RefreshTokens(ctx context.Context, tokens Tokens) (*Tokens, error) {
	return m.mockRefreshTokens(ctx, tokens)
}

func (m mockAuthorizer) GetAuthenticatedUser(ctx context.Context) (User, error) {
	return m.mockGetAuthenticatedUser(ctx)
}

func mustLoadConfig() configs.Config {
	config, err := configs.Load("./../../test/testdata/config_valid.json")
	if err != nil {
		panic(err)
	}
	return config
}

func mustCreateSQLMock() mockConnection {
	db, mock, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	return mockConnection{
		db:   db,
		mock: mock,
	}
}

func mustGenerateAccessToken(config configs.Config, user *User) *Tokens {
	accessToken, err := NewJwtToken(GetDefaultAccessTokenOptions(WithSubject(user.UUID.String()), WithRole(user.Role))...)
	if err != nil {
		panic(err)
	}
	signedAccessToken, err := SignToken(accessToken, config.PrivateKey())
	if err != nil {
		panic(err)
	}
	refreshToken, err := NewJwtToken(GetDefaultRefreshTokenOptions(WithSubject(user.UUID.String()), WithRole(user.Role))...)
	if err != nil {
		panic(err)
	}
	signedRefreshToken, err := SignToken(refreshToken, config.PrivateKey())
	if err != nil {
		panic(err)
	}
	return &Tokens{
		AccessToken:  signedAccessToken,
		RefreshToken: signedRefreshToken,
	}
}

func mustGenerateExpiratedAccessToken(config configs.Config, user *User) *Tokens {
	accessToken, err := NewJwtToken(GetDefaultAccessTokenOptions(WithSubject(user.UUID.String()), WithRole(user.Role), func(token jwt.Token) error {
		return token.Set(jwt.ExpirationKey, time.Now().Add(-10*time.Hour))
	})...)
	if err != nil {
		panic(err)
	}
	signedAccessToken, err := SignToken(accessToken, config.PrivateKey())
	if err != nil {
		panic(err)
	}
	refreshToken, err := NewJwtToken(GetDefaultRefreshTokenOptions(WithSubject(user.UUID.String()), WithRole(user.Role), func(token jwt.Token) error {
		return token.Set(jwt.ExpirationKey, time.Now().Add(-10*time.Hour))
	})...)
	if err != nil {
		panic(err)
	}
	signedRefreshToken, err := SignToken(refreshToken, config.PrivateKey())
	if err != nil {
		panic(err)
	}
	return &Tokens{
		AccessToken:  signedAccessToken,
		RefreshToken: signedRefreshToken,
	}
}

func TestAuthenticate(t *testing.T) {
	type args struct {
		config      configs.Config
		dbConn      mockConnection
		mockResult  func(dbConn mockConnection)
		credentials Credentials
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should authenticate the user",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection) {
					hashedPass, _ := EncryptPassword("test")

					findUserByEmailResult := sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(1, uuid.New(), "patient@hospital.com", PatientRole)
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByEmailQuery)).WithArgs("patient@hospital.com").WillReturnRows(findUserByEmailResult)

					checkUserPasswordResult := sqlmock.NewRows([]string{"id", "password"}).AddRow(1, hashedPass)
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(checkUserPasswordQuery)).WithArgs("patient@hospital.com").WillReturnRows(checkUserPasswordResult)
				},
				credentials: Credentials{
					Email:    "patient@hospital.com",
					Password: "test",
				},
			},
			want: http.StatusOK,
		},
		{
			name: "should not authenticate the user due to a unknown user",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection) {
					findUserByEmailResult := sqlmock.NewRows([]string{"id", "uuid", "email", "role"})
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByEmailQuery)).WithArgs("patient@hospital.com").WillReturnRows(findUserByEmailResult)
				},
				credentials: Credentials{
					Email:    "patient@hospital.com",
					Password: "test",
				},
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "should not authenticate the user due to a invalid password",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection) {
					hashedPass, _ := EncryptPassword("testing")

					findUserByEmailResult := sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(1, uuid.New(), "patient@hospital.com", PatientRole)
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByEmailQuery)).WithArgs("patient@hospital.com").WillReturnRows(findUserByEmailResult)

					checkUserPasswordResult := sqlmock.NewRows([]string{"id", "password"}).AddRow(1, hashedPass)
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(checkUserPasswordQuery)).WithArgs("patient@hospital.com").WillReturnRows(checkUserPasswordResult)
				},
				credentials: Credentials{
					Email:    "patient@hospital.com",
					Password: "test",
				},
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "should not authenticate the user due to a database error while searching for the user",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection) {
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByEmailQuery)).WithArgs("patient@hospital.com").WillReturnError(sql.ErrConnDone)
				},
				credentials: Credentials{
					Email:    "patient@hospital.com",
					Password: "test",
				},
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not authenticate the user due to a database error while parsing the user found",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection) {
					findUserByEmailResult := sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(-1, false, "patient@hospital.com", PatientRole)
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByEmailQuery)).WithArgs("patient@hospital.com").WillReturnRows(findUserByEmailResult)
				},
				credentials: Credentials{
					Email:    "patient@hospital.com",
					Password: "test",
				},
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not authenticate the user due to a database error while searching for the user to check password",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection) {
					findUserByEmailResult := sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(1, uuid.New(), "patient@hospital.com", PatientRole)
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByEmailQuery)).WithArgs("patient@hospital.com").WillReturnRows(findUserByEmailResult)

					dbConn.mock.ExpectQuery(regexp.QuoteMeta(checkUserPasswordQuery)).WithArgs("patient@hospital.com").WillReturnError(sql.ErrConnDone)
				},
				credentials: Credentials{
					Email:    "patient@hospital.com",
					Password: "test",
				},
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not authenticate the user due to a database error while parsing for the user to check password",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection) {
					findUserByEmailResult := sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(1, uuid.New(), "patient@hospital.com", PatientRole)
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByEmailQuery)).WithArgs("patient@hospital.com").WillReturnRows(findUserByEmailResult)

					checkUserPasswordResult := sqlmock.NewRows([]string{"id", "password"}).AddRow(false, -1)
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(checkUserPasswordQuery)).WithArgs("patient@hospital.com").WillReturnRows(checkUserPasswordResult)
				},
				credentials: Credentials{
					Email:    "patient@hospital.com",
					Password: "test",
				},
			},
			want: http.StatusInternalServerError,
		},

		{
			name: "should not authenticate the user due to empty email",
			args: args{
				config:      mustLoadConfig(),
				dbConn:      mustCreateSQLMock(),
				mockResult:  func(dbConn mockConnection) {},
				credentials: Credentials{},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not authenticate the user due to empty password",
			args: args{
				config:     mustLoadConfig(),
				dbConn:     mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection) {},
				credentials: Credentials{
					Email: "patient@hospital.com",
				},
			},
			want: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			Setup(router, logger, tt.args.config, tt.args.dbConn)

			tt.args.mockResult(tt.args.dbConn)

			body, _ := json.Marshal(tt.args.credentials)
			req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			response := recorder.Result()

			if response.StatusCode != tt.want {
				t.Errorf("response status is incorrect, got %d, want %d", recorder.Code, tt.want)
			}
		})
	}
}

func TestGetAuthenticatedUser(t *testing.T) {
	type args struct {
		config     configs.Config
		dbConn     mockConnection
		mockResult func(dbConn mockConnection, user *User)
		user       *User
		tokens     *Tokens
	}
	tests := []struct {
		name         string
		args         args
		want         int
		wantResponse string
	}{
		{
			name: "should get the authenticated user",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection, user *User) {
					findUserByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(user.ID, user.UUID, user.Email, user.Role)
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByUUIDQuery)).WithArgs(user.UUID).WillReturnRows(findUserByUUIDResult)
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}),
			},
			want:         http.StatusOK,
			wantResponse: "{\"uuid\":\"00000000-0000-0000-0000-000000000000\",\"email\":\"patient@hospital.com\",\"role\":\"PATIENT\"}\n",
		},
		{
			name: "should not get the authenticated user due to a unknown error",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection, user *User) {
					findUserByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "email", "role"})
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByUUIDQuery)).WithArgs(user.UUID).WillReturnRows(findUserByUUIDResult)
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}),
			},
			want:         http.StatusUnauthorized,
			wantResponse: "",
		},
		{
			name: "should not get the authenticated user due to a database error while searching for the user",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection, user *User) {
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByUUIDQuery)).WithArgs(user.UUID).WillReturnError(sql.ErrConnDone)
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}),
			},
			want:         http.StatusUnauthorized,
			wantResponse: "",
		},
		{
			name: "should not get the authenticated user due to a database error while parsing the user",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection, user *User) {
					findUserByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(-1, false, "patient@hospital.com", PatientRole)
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByUUIDQuery)).WithArgs(user.UUID).WillReturnRows(findUserByUUIDResult)
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}),
			},
			want:         http.StatusUnauthorized,
			wantResponse: "",
		},
		{
			name: "should not get the authenticated user due to the missing header",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection, user *User) {
					findUserByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(-1, false, "patient@hospital.com", PatientRole)
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByUUIDQuery)).WithArgs(user.UUID).WillReturnRows(findUserByUUIDResult)
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: nil,
			},
			want:         http.StatusUnauthorized,
			wantResponse: "",
		},
		{
			name: "should not get the authenticated user due to a expired token",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection, user *User) {
					findUserByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(-1, false, "patient@hospital.com", PatientRole)
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByUUIDQuery)).WithArgs(user.UUID).WillReturnRows(findUserByUUIDResult)
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: mustGenerateExpiratedAccessToken(mustLoadConfig(), &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}),
			},
			want:         http.StatusUnauthorized,
			wantResponse: "",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			Setup(router, logger, tt.args.config, tt.args.dbConn)

			tt.args.mockResult(tt.args.dbConn, tt.args.user)

			req, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)

			token := ""
			if tt.args.tokens != nil {
				token = fmt.Sprintf("Bearer %s", tt.args.tokens.AccessToken)
			}

			req.Header.Add("Authorization", token)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			response := recorder.Result()

			if response.StatusCode != tt.want {
				t.Errorf("response status is incorrect, got %d, want %d", recorder.Code, tt.want)
			}

			buf := new(bytes.Buffer)
			_, err := buf.ReadFrom(response.Body)
			if err != nil {
				t.Errorf("an error occurred while reading response body: %v", err)
			}

			responseBody := buf.String()

			if tt.wantResponse != responseBody {
				t.Errorf("response body is incorrect, got %s, want %s", responseBody, tt.wantResponse)
			}
		})
	}
}

func TestRefreshToken(t *testing.T) {
	type args struct {
		config      configs.Config
		dbConn      mockConnection
		mockResult  func(dbConn mockConnection, user *User)
		changeToken func(tokens *Tokens)
		user        *User
		tokens      *Tokens
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should refresh tokens successfully",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection, user *User) {
					findUserByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(user.ID, user.UUID, user.Email, user.Role)
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByUUIDQuery)).WithArgs(user.UUID).WillReturnRows(findUserByUUIDResult)
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}),
				changeToken: func(tokens *Tokens) {
					tokens.GrantType = "refresh_token"
				},
			},
			want: http.StatusOK,
		},
		{
			name: "should not refresh tokens due to a token without grant type",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection, user *User) {
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}),
				changeToken: func(tokens *Tokens) {
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not refresh tokens due to a token without grant type different than expected",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection, user *User) {
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}),
				changeToken: func(tokens *Tokens) {
					tokens.GrantType = "invalid"
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not refresh tokens due to a token without refresh token",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection, user *User) {
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}),
				changeToken: func(tokens *Tokens) {
					tokens.RefreshToken = ""
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not refresh tokens due to a token without access token",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection, user *User) {
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}),
				changeToken: func(tokens *Tokens) {
					tokens.AccessToken = ""
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not refresh tokens due to a token with a invalid refresh token",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection, user *User) {
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}),
				changeToken: func(tokens *Tokens) {
					tokens.RefreshToken = "invalid"
					tokens.GrantType = "refresh_token"
				},
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "should not refresh token due to a database error while searching for the user",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection, user *User) {
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByUUIDQuery)).WithArgs(user.UUID).WillReturnError(sql.ErrConnDone)
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}),
				changeToken: func(tokens *Tokens) {
					tokens.GrantType = "refresh_token"
				},
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not refresh token due to a database error while parsing the user",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection, user *User) {
					findUserByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(-1, false, "patient@hospital.com", PatientRole)
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByUUIDQuery)).WithArgs(user.UUID).WillReturnRows(findUserByUUIDResult)
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}),
				changeToken: func(tokens *Tokens) {
					tokens.GrantType = "refresh_token"
				},
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not refresh token due to a unknown error",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection, user *User) {
					findUserByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "email", "role"})
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByUUIDQuery)).WithArgs(user.UUID).WillReturnRows(findUserByUUIDResult)
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}),
				changeToken: func(tokens *Tokens) {
					tokens.GrantType = "refresh_token"
				},
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "should not refresh token due to a expired token",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockResult: func(dbConn mockConnection, user *User) {
					findUserByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "email", "role"})
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findUserByUUIDQuery)).WithArgs(user.UUID).WillReturnRows(findUserByUUIDResult)
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: mustGenerateExpiratedAccessToken(mustLoadConfig(), &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}),
				changeToken: func(tokens *Tokens) {
					tokens.GrantType = "refresh_token"
				},
			},
			want: http.StatusUnauthorized,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			Setup(router, logger, tt.args.config, tt.args.dbConn)

			tt.args.mockResult(tt.args.dbConn, tt.args.user)

			tt.args.changeToken(tt.args.tokens)

			body, _ := json.Marshal(tt.args.tokens)
			req, _ := http.NewRequest("PUT", "/api/v1/auth/token", bytes.NewBuffer(body))

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			response := recorder.Result()

			if response.StatusCode != tt.want {
				t.Errorf("response status is incorrect, got %d, want %d", recorder.Code, tt.want)
			}

			buf := new(bytes.Buffer)
			_, err := buf.ReadFrom(response.Body)
			if err != nil {
				t.Errorf("an error occurred while reading response body: %v", err)
			}
		})
	}
}
