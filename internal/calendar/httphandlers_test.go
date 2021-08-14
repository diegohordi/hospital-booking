package calendar

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"hospital-booking/internal/auth"
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
	mockValidateToken        func(ctx context.Context, token string) (*auth.User, error)
	mockRefreshTokens        func(ctx context.Context, tokens auth.Tokens) (*auth.Tokens, error)
	mockGetAuthenticatedUser func(ctx context.Context) (auth.User, error)
}

func (m mockAuthorizer) ValidateToken(ctx context.Context, token string) (*auth.User, error) {
	return m.mockValidateToken(ctx, token)
}

func (m mockAuthorizer) RefreshTokens(ctx context.Context, tokens auth.Tokens) (*auth.Tokens, error) {
	return m.mockRefreshTokens(ctx, tokens)
}

func (m mockAuthorizer) GetAuthenticatedUser(ctx context.Context) (auth.User, error) {
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
	mock.MatchExpectationsInOrder(false)
	return mockConnection{
		db:   db,
		mock: mock,
	}
}

func mustGenerateAccessToken(config configs.Config, user *auth.User) *auth.Tokens {
	accessToken, err := auth.NewJwtToken(auth.GetDefaultAccessTokenOptions(auth.WithSubject(user.UUID.String()), auth.WithRole(user.Role))...)
	if err != nil {
		panic(err)
	}
	signedAccessToken, err := auth.SignToken(accessToken, config.PrivateKey())
	if err != nil {
		panic(err)
	}
	refreshToken, err := auth.NewJwtToken(auth.GetDefaultRefreshTokenOptions(auth.WithSubject(user.UUID.String()), auth.WithRole(user.Role))...)
	if err != nil {
		panic(err)
	}
	signedRefreshToken, err := auth.SignToken(refreshToken, config.PrivateKey())
	if err != nil {
		panic(err)
	}
	return &auth.Tokens{
		AccessToken:  signedAccessToken,
		RefreshToken: signedRefreshToken,
	}
}

func TestGetDoctorCalendar(t *testing.T) {
	type args struct {
		config     configs.Config
		mockAuth   mockAuthorizer
		dbConn     mockConnection
		mockResult func(dbConn mockConnection)
		tokens     *auth.Tokens
		doctorUUID *uuid.UUID
		year       string
		month      string
		day        string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should get the doctor calendar with appointments and blockers",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(uuid.UUID{}).WillReturnRows(findDoctorByUUIDResult)

					listAppointmentsResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listAppointmentsResult)

					listBlockersResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listBlockersQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listBlockersResult)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusOK,
		},
		{
			name: "should get the doctor calendar with no appointments and no blockers",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					findDoctorByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(uuid.UUID{}).WillReturnRows(findDoctorByUUIDResult)

					listAppointmentsResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"})
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listAppointmentsResult)

					listBlockersResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"})
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listBlockersQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listBlockersResult)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusOK,
		},
		{
			name: "should not get the doctor calendar due to a wrong UUID",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {
				},
				doctorUUID: nil,
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not get the doctor calendar due to a wrong date parameter",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {
				},
				doctorUUID: &uuid.UUID{},
				year:       "AAAA",
				month:      "08",
				day:        "10",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not get the doctor calendar due to a unknown doctor",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					findDoctorByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"})
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(uuid.UUID{}).WillReturnRows(findDoctorByUUIDResult)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusNotFound,
		},
		{
			name: "should not get the doctor calendar due to an error while getting the doctor",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(uuid.UUID{}).WillReturnError(sql.ErrConnDone)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the doctor calendar due to an error while parsing the doctor",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					findDoctorByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(uuid.UUID{}).WillReturnRows(findDoctorByUUIDResult)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the doctor calendar due to an error while getting appointments",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					findDoctorByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(uuid.UUID{}).WillReturnRows(findDoctorByUUIDResult)

					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},

		{
			name: "should not get the doctor calendar due to an error while parsing appointments",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					findDoctorByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(uuid.UUID{}).WillReturnRows(findDoctorByUUIDResult)

					listAppointmentsResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, false, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listAppointmentsResult)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the doctor calendar due to an error while getting blockers",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					findDoctorByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(uuid.UUID{}).WillReturnRows(findDoctorByUUIDResult)

					listAppointmentsResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listAppointmentsResult)

					// listBlockersResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listBlockersQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the doctor calendar due to an error while parsing blockers",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					findDoctorByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(uuid.UUID{}).WillReturnRows(findDoctorByUUIDResult)

					listAppointmentsResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listAppointmentsResult)

					listBlockersResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, false, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listBlockersQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listBlockersResult)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			Setup(router, logger, tt.args.mockAuth, tt.args.config, tt.args.dbConn)

			tt.args.mockResult(tt.args.dbConn)

			req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/bookings/calendar/%s/%s/%s/%s", tt.args.doctorUUID, tt.args.year, tt.args.month, tt.args.day), nil)

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
		})
	}
}

func TestGetAppointments(t *testing.T) {
	type args struct {
		config     configs.Config
		mockAuth   mockAuthorizer
		dbConn     mockConnection
		mockResult func(dbConn mockConnection)
		tokens     *auth.Tokens
		doctorUUID *uuid.UUID
		year       string
		month      string
		day        string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should get the calendar with appointments and blockers",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)

					listAppointmentsResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listAppointmentsResult)

					listBlockersResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listBlockersQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listBlockersResult)

					findPatientByIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findPatientByIDQuery)).WithArgs(1).WillReturnRows(findPatientByIDResult)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusOK,
		},
		{
			name: "should get the calendar with no appointments and no blockers",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)

					listAppointmentsResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"})
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listAppointmentsResult)

					listBlockersResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"})
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listBlockersQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listBlockersResult)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusOK,
		},
		{
			name: "should not get the calendar due to a wrong date parameter",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
				},
				doctorUUID: &uuid.UUID{},
				year:       "AAAA",
				month:      "08",
				day:        "10",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not get the calendar due to a unknown doctor",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"})
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusForbidden,
		},
		{
			name: "should not get the calendar due to an error while getting the doctor",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(uuid.UUID{}).WillReturnError(sql.ErrConnDone)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the calendar due to an error while parsing the doctor",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, false, "John Doe", "doctor@hospital.com")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the calendar due to an error while getting appointments",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)

					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},

		{
			name: "should not get the calendar due to an error while parsing appointments",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)

					listAppointmentsResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, false, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listAppointmentsResult)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the calendar due to an error while getting blockers",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)

					listAppointmentsResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"})
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listAppointmentsResult)

					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listBlockersQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the calendar due to an error while parsing blockers",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)

					listAppointmentsResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listAppointmentsResult)

					listBlockersResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, false, 1, true, false, "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listBlockersQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listBlockersResult)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the calendar due to an error while getting appointment's patient",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)

					listAppointmentsResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listAppointmentsResult)

					listBlockersResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listBlockersQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listBlockersResult)

					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findPatientByIDQuery)).WithArgs(1).WillReturnError(sql.ErrConnDone)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the calendar due to an error while parsing appointment's patient",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)

					listAppointmentsResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listAppointmentsResult)

					listBlockersResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listBlockersQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listBlockersResult)

					findPatientByIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, false, 1, "John Doe", "doctor@hospital.com", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findPatientByIDQuery)).WithArgs(1).WillReturnRows(findPatientByIDResult)
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			Setup(router, logger, tt.args.mockAuth, tt.args.config, tt.args.dbConn)

			tt.args.mockResult(tt.args.dbConn)

			req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/bookings/calendar/%s/%s/%s", tt.args.year, tt.args.month, tt.args.day), nil)

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
		})
	}
}

func TestInsertBlockPeriod(t *testing.T) {
	type args struct {
		config      configs.Config
		mockAuth    mockAuthorizer
		dbConn      mockConnection
		mockResult  func(dbConn mockConnection)
		tokens      *auth.Tokens
		blockPeriod *BlockPeriod
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should insert a block period",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)

					insertBlockerResult := sqlmock.NewResult(1, 1)
					dbConn.mock.ExpectExec(regexp.QuoteMeta(insertBlockerQuery)).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(insertBlockerResult)
				},
				blockPeriod: &BlockPeriod{
					StartDate:   time.Now(),
					EndDate:     time.Now().Add(24 * time.Hour),
					Description: nil,
				},
			},
			want: http.StatusCreated,
		},
		{
			name: "should not insert a block period due to a unknown doctor",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"})
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)
				},
				blockPeriod: &BlockPeriod{
					StartDate:   time.Now(),
					EndDate:     time.Now().Add(24 * time.Hour),
					Description: nil,
				},
			},
			want: http.StatusForbidden,
		},
		{
			name: "should not insert a block period due to an error while getting the doctor",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnError(sql.ErrConnDone)
				},
				blockPeriod: &BlockPeriod{
					StartDate:   time.Now(),
					EndDate:     time.Now().Add(24 * time.Hour),
					Description: nil,
				},
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not insert a block period due to a empty start date",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)
				},
				blockPeriod: &BlockPeriod{
					EndDate:     time.Now().Add(24 * time.Hour),
					Description: nil,
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not insert a block period due to a empty end date",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)
				},
				blockPeriod: &BlockPeriod{
					StartDate:   time.Now(),
					Description: nil,
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not insert a block period due to a invalid end date",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)
				},
				blockPeriod: &BlockPeriod{
					StartDate:   time.Now(),
					EndDate:     time.Now().Add(-24 * time.Hour),
					Description: nil,
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not insert a block period due to an error while inserting",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)

					dbConn.mock.ExpectExec(regexp.QuoteMeta(insertBlockerQuery)).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
				},
				blockPeriod: &BlockPeriod{
					StartDate:   time.Now(),
					EndDate:     time.Now().Add(24 * time.Hour),
					Description: nil,
				},
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not insert a block period due to no rows affected after inserting into database",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.New(),
							Email: "doctor@hospital.com",
							Role:  auth.DoctorRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "doctor@hospital.com",
					Role:  auth.DoctorRole,
				}),
				mockResult: func(dbConn mockConnection) {
					findDoctorByUserIDDResult := sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(1).WillReturnRows(findDoctorByUserIDDResult)

					insertBlockerResult := sqlmock.NewResult(0, 0)
					dbConn.mock.ExpectExec(regexp.QuoteMeta(insertBlockerQuery)).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(insertBlockerResult)
				},
				blockPeriod: &BlockPeriod{
					StartDate:   time.Now(),
					EndDate:     time.Now().Add(24 * time.Hour),
					Description: nil,
				},
			},
			want: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			Setup(router, logger, tt.args.mockAuth, tt.args.config, tt.args.dbConn)

			tt.args.mockResult(tt.args.dbConn)

			body, _ := json.Marshal(tt.args.blockPeriod)
			req, _ := http.NewRequest("POST", "/api/v1/bookings/calendar/blockers", bytes.NewBuffer(body))

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
		})
	}
}

func TestInsertAppointment(t *testing.T) {
	type args struct {
		config             configs.Config
		mockAuth           mockAuthorizer
		dbConn             mockConnection
		mockResult         func(dbConn mockConnection)
		tokens             *auth.Tokens
		appointmentRequest *AppointmentRequest
		doctorUUID         *uuid.UUID
		year               string
		month              string
		day                string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should insert a appointment",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					findDoctorByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findDoctorByUUIDResult)

					findDoctorByUUIDResult2 := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findDoctorByUUIDResult2)

					findPatientByUserIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, uuid.UUID{}, 1, "Patient", "patient@hospital.com", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findPatientByUserIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findPatientByUserIDResult)

					listAppointmentsResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listAppointmentsResult)

					listBlockersResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listBlockersQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listBlockersResult)

					insertAppointmentResult := sqlmock.NewResult(1, 1)
					dbConn.mock.ExpectExec(regexp.QuoteMeta(insertAppointmentQuery)).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(insertAppointmentResult)

				},
				appointmentRequest: &AppointmentRequest{
					Hour: 9,
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusCreated,
		},
		{
			name: "should not insert an appointment due to unknown patient",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					findPatientByUserIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"})
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findPatientByUserIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findPatientByUserIDResult)

				},
				appointmentRequest: &AppointmentRequest{
					Hour: 9,
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusForbidden,
		},
		{
			name: "should not insert an appointment due to an error while getting the patient from database",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findPatientByUserIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)

				},
				appointmentRequest: &AppointmentRequest{
					Hour: 9,
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not insert an appointment due to an error while parsing the patient from database",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					findPatientByUserIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, false, 1, "Patient", "patient@hospital.com", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findPatientByUserIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findPatientByUserIDResult)

				},
				appointmentRequest: &AppointmentRequest{
					Hour: 9,
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not insert an appointment due to unknown doctor",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					findDoctorByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"})
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findDoctorByUUIDResult)

					findPatientByUserIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, uuid.UUID{}, 1, "Patient", "patient@hospital.com", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findPatientByUserIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findPatientByUserIDResult)

				},
				appointmentRequest: &AppointmentRequest{
					Hour: 9,
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusNotFound,
		},
		{
			name: "should not insert an appointment due to an error while getting the doctor from database",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)

					findPatientByUserIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, uuid.UUID{}, 1, "Patient", "patient@hospital.com", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findPatientByUserIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findPatientByUserIDResult)

				},
				appointmentRequest: &AppointmentRequest{
					Hour: 9,
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not insert an appointment due to an error while parsing the doctor from database",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					findDoctorByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, false, 1, "John Doe", "doctor@hospital.com", "", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findDoctorByUUIDResult)

					findPatientByUserIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, uuid.UUID{}, 1, "Patient", "patient@hospital.com", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findPatientByUserIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findPatientByUserIDResult)

				},
				appointmentRequest: &AppointmentRequest{
					Hour: 9,
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not insert an appointment due to a invalid hour",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {
				},
				appointmentRequest: &AppointmentRequest{
					Hour: 19,
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not insert an appointment due to a unavailable slot",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					findDoctorByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findDoctorByUUIDResult)

					findDoctorByUUIDResult2 := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findDoctorByUUIDResult2)

					findPatientByUserIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, uuid.UUID{}, 1, "Patient", "patient@hospital.com", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findPatientByUserIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findPatientByUserIDResult)

					listAppointmentsResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listAppointmentsResult)

					listBlockersResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listBlockersQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listBlockersResult)

				},
				appointmentRequest: &AppointmentRequest{
					Hour: 10,
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not insert an appointment due to an error while inserting the appointment into database",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					findDoctorByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findDoctorByUUIDResult)

					findDoctorByUUIDResult2 := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findDoctorByUUIDResult2)

					findPatientByUserIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, uuid.UUID{}, 1, "Patient", "patient@hospital.com", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findPatientByUserIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findPatientByUserIDResult)

					listAppointmentsResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listAppointmentsResult)

					listBlockersResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listBlockersQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listBlockersResult)

					dbConn.mock.ExpectExec(regexp.QuoteMeta(insertAppointmentQuery)).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)

				},
				appointmentRequest: &AppointmentRequest{
					Hour: 9,
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not insert an appointment due to an empty result after inserting the appointment into database",
			args: args{
				config: mustLoadConfig(),
				dbConn: mustCreateSQLMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return &auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return auth.User{
							ID:    1,
							UUID:  uuid.UUID{},
							Email: "patient@hospital.com",
							Role:  auth.PatientRole,
						}, nil
					},
				},
				tokens: mustGenerateAccessToken(mustLoadConfig(), &auth.User{
					ID:    1,
					UUID:  uuid.UUID{},
					Email: "patient@hospital.com",
					Role:  auth.PatientRole,
				}),
				mockResult: func(dbConn mockConnection) {

					findDoctorByUUIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findDoctorByUUIDResult)

					findDoctorByUUIDResult2 := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findDoctorByUUIDResult2)

					findPatientByUserIDResult := sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, uuid.UUID{}, 1, "Patient", "patient@hospital.com", "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(findPatientByUserIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(findPatientByUserIDResult)

					listAppointmentsResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listAppointmentsResult)

					listBlockersResult := sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")
					dbConn.mock.ExpectQuery(regexp.QuoteMeta(listBlockersQuery)).WithArgs(1, sqlmock.AnyArg()).WillReturnRows(listBlockersResult)

					insertAppointmentResult := sqlmock.NewResult(0, 0)
					dbConn.mock.ExpectExec(regexp.QuoteMeta(insertAppointmentQuery)).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(insertAppointmentResult)


				},
				appointmentRequest: &AppointmentRequest{
					Hour: 9,
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			Setup(router, logger, tt.args.mockAuth, tt.args.config, tt.args.dbConn)

			tt.args.mockResult(tt.args.dbConn)

			body, _ := json.Marshal(tt.args.appointmentRequest)
			req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/bookings/calendar/%s/%s/%s/%s", tt.args.doctorUUID, tt.args.year, tt.args.month, tt.args.day), bytes.NewBuffer(body))

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
		})
	}
}
