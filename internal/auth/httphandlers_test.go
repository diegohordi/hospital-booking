package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"hospital-booking/internal/configs"
	"hospital-booking/internal/mock"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/jwt"
)

const (
	hashedTestPassword = "$2a$10$1Q/8dWTn4AsoKm0SIVl8LeBf8x0jNPf7Wj92Ywmk07XI.9s95b/eK"
	plainTestPassword  = "test"
)

var logger = log.New(os.Stdout, "", log.LstdFlags)

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

func withFindUserByEmailResult(rows *sqlmock.Rows) mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(findUserByEmailQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(rows)
	}
}

func withCheckUserPasswordResult(rows *sqlmock.Rows) mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(checkUserPasswordQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(rows)
	}
}

func withFindUserByUUIDResult(rows *sqlmock.Rows) mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(findUserByUUIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(rows)
	}
}

func withFindUserByEmailError() mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(findUserByEmailQuery)).WithArgs(sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
	}
}

func withCheckUserPasswordError() mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(checkUserPasswordQuery)).WithArgs(sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
	}
}

func withFindUserByUUIDError() mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(findUserByUUIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
	}
}

func TestAuthenticate(t *testing.T) {
	config := configs.MustLoad("./../../test/testdata/config_valid.json")
	type args struct {
		config        configs.Config
		dbConn        mock.Connection
		dbMockOptions []mock.DBResultOption
		credentials   Credentials
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should authenticate the user",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				dbMockOptions: []mock.DBResultOption{
					withFindUserByEmailResult(sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(1, uuid.New(), "patient@hospital.com", PatientRole)),
					withCheckUserPasswordResult(sqlmock.NewRows([]string{"id", "password"}).AddRow(1, hashedTestPassword)),
				},
				credentials: Credentials{
					Email:    "patient@hospital.com",
					Password: plainTestPassword,
				},
			},
			want: http.StatusOK,
		},
		{
			name: "should not authenticate the user because the user was not found",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				dbMockOptions: []mock.DBResultOption{
					withFindUserByEmailResult(sqlmock.NewRows([]string{"id", "uuid", "email", "role"})),
				},
				credentials: Credentials{
					Email:    "patient@hospital.com",
					Password: "test",
				},
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "should not authenticate the user because the given password is invalid",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				dbMockOptions: []mock.DBResultOption{
					withFindUserByEmailResult(sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(1, uuid.New(), "patient@hospital.com", PatientRole)),
					withCheckUserPasswordResult(sqlmock.NewRows([]string{"id", "password"}).AddRow(1, "testing")),
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
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				dbMockOptions: []mock.DBResultOption{
					withFindUserByEmailError(),
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
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				dbMockOptions: []mock.DBResultOption{
					withFindUserByEmailResult(sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(-1, false, "patient@hospital.com", PatientRole)),
				},
				credentials: Credentials{
					Email:    "patient@hospital.com",
					Password: "test",
				},
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not authenticate the user due to a database error while searching for the user password",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				dbMockOptions: []mock.DBResultOption{
					withCheckUserPasswordError(),
				},
				credentials: Credentials{
					Email:    "patient@hospital.com",
					Password: "test",
				},
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not authenticate the user due to a database error while parsing the user password",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				dbMockOptions: []mock.DBResultOption{
					withCheckUserPasswordResult(sqlmock.NewRows([]string{"id", "password"}).AddRow(false, -1)),
				},
				credentials: Credentials{
					Email:    "patient@hospital.com",
					Password: "test",
				},
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not authenticate the user because the email was empty",
			args: args{
				config:      config,
				dbConn:      mock.MustCreateConnectionMock(),
				credentials: Credentials{},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not authenticate the user because the password was empty",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
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

			mock.MockDBResults(tt.args.dbConn, tt.args.dbMockOptions...)

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
	config := configs.MustLoad("./../../test/testdata/config_valid.json")
	type args struct {
		config        configs.Config
		dbConn        mock.Connection
		dbMockOptions []mock.DBResultOption
		user          *User
		tokens        *Tokens
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
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				dbMockOptions: []mock.DBResultOption{
					withFindUserByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(1, uuid.UUID{}, "patient@hospital.com", PatientRole)),
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: MustGenerateTokens(context.TODO(), config.PrivateKey(), User{
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
			name: "should not get the authenticated because the user was not found",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				dbMockOptions: []mock.DBResultOption{
					withFindUserByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "email", "role"})),
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: MustGenerateTokens(context.TODO(), config.PrivateKey(), User{
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
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				dbMockOptions: []mock.DBResultOption{
					withFindUserByUUIDError(),
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: MustGenerateTokens(context.TODO(), config.PrivateKey(), User{
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
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				dbMockOptions: []mock.DBResultOption{
					withFindUserByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(-1, false, "patient@hospital.com", PatientRole)),
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: MustGenerateTokens(context.TODO(), config.PrivateKey(), User{
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
			name: "should not get the authenticated user because the authorization header is missing",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
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
			name: "should not get the authenticated user because the given token is expired",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: MustGenerateTokens(context.TODO(), config.PrivateKey(), User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}, []TokenOption{func(token jwt.Token) error {
					return token.Set(jwt.ExpirationKey, time.Now().Add(-10*time.Hour))
				}}...),
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

			mock.MockDBResults(tt.args.dbConn, tt.args.dbMockOptions...)

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
	config := configs.MustLoad("./../../test/testdata/config_valid.json")
	type args struct {
		config        configs.Config
		dbConn        mock.Connection
		dbMockOptions []mock.DBResultOption
		changeToken   func(tokens *Tokens)
		user          *User
		tokens        *Tokens
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should refresh tokens",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				dbMockOptions: []mock.DBResultOption{
					withFindUserByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(1, uuid.UUID{}, "patient@hospital.com", PatientRole)),
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: MustGenerateTokens(context.TODO(), config.PrivateKey(), User{
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
			name: "should not refresh tokens because the grant_type is missing",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: MustGenerateTokens(context.TODO(), config.PrivateKey(), User{
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
			name: "should not refresh tokens because the grant_type is different from expected",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: MustGenerateTokens(context.TODO(), config.PrivateKey(), User{
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
			name: "should not refresh tokens because the given tokens contains no refresh token",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: MustGenerateTokens(context.TODO(), config.PrivateKey(), User{
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
			name: "should not refresh tokens because the given tokens contains no access token",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: MustGenerateTokens(context.TODO(), config.PrivateKey(), User{
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
			name: "should not refresh tokens because the given refresh_token is invalid",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: MustGenerateTokens(context.TODO(), config.PrivateKey(), User{
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
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				dbMockOptions: []mock.DBResultOption{
					withFindUserByUUIDError(),
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: MustGenerateTokens(context.TODO(), config.PrivateKey(), User{
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
			name: "should not refresh token due to a database error while parsing the user found",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				dbMockOptions: []mock.DBResultOption{
					withFindUserByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "email", "role"}).AddRow(-1, false, "patient@hospital.com", PatientRole)),
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: MustGenerateTokens(context.TODO(), config.PrivateKey(), User{
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
			name: "should not refresh token because the user associated to it no longer exists",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				dbMockOptions: []mock.DBResultOption{
					withFindUserByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "email", "role"})),
				},
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: MustGenerateTokens(context.TODO(), config.PrivateKey(), User{
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
			name: "should not refresh token because the given token is expired",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				user: &User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				},
				tokens: MustGenerateTokens(context.TODO(), config.PrivateKey(), User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  PatientRole,
				}, []TokenOption{func(token jwt.Token) error {
					return token.Set(jwt.ExpirationKey, time.Now().Add(-10*time.Hour))
				}}...),
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

			mock.MockDBResults(tt.args.dbConn, tt.args.dbMockOptions...)

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
