package calendar

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"hospital-booking/internal/auth"
	"hospital-booking/internal/configs"
	"hospital-booking/internal/mock"
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

func withFindDoctorByUUIDResult(rows *sqlmock.Rows) mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(rows)
	}
}

func withFindDoctorByUUIDError() mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(findDoctorByUUIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
	}
}

func withFindDoctorByUserIDResult(rows *sqlmock.Rows) mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(rows)
	}
}

func withFindDoctorByUserIDError() mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(findDoctorByUserIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
	}
}

func withFindPatientByIDResult(rows *sqlmock.Rows) mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(findPatientByIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(rows)
	}
}

func withFindPatientByIDError() mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(findPatientByIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
	}
}

func withFindPatientByUserIDResult(rows *sqlmock.Rows) mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(findPatientByUserIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(rows)
	}
}

func withFindPatientByUserIDError() mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(findPatientByUserIDQuery)).WithArgs(sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
	}
}

func withInsertBlockerResult(result driver.Result) mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectExec(regexp.QuoteMeta(insertBlockerQuery)).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(result)
	}
}

func withInsertBlockerError() mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectExec(regexp.QuoteMeta(insertBlockerQuery)).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
	}
}

func withListBlockersResult(rows *sqlmock.Rows) mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(listBlockersQuery)).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(rows)
	}
}

func withListBlockersError() mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(listBlockersQuery)).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
	}
}

func withInsertAppointmentResult(result driver.Result) mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectExec(regexp.QuoteMeta(insertAppointmentQuery)).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(result)
	}
}

func withInsertAppointmentError() mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(insertAppointmentQuery)).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
	}
}

func withListAppointmentsResult(rows *sqlmock.Rows) mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(rows)
	}
}

func withListAppointmentsError() mock.DBResultOption {
	return func(dbConn mock.Connection) {
		dbConn.SQLMock.ExpectQuery(regexp.QuoteMeta(listAppointmentsQuery)).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
	}
}

func mockPatientUser() *auth.User {
	return &auth.User{
		ID:    1,
		UUID:  uuid.New(),
		Email: "patient@hospital.com",
		Role:  auth.PatientRole,
	}
}

func mockDoctorUser() *auth.User {
	return &auth.User{
		ID:    1,
		UUID:  uuid.UUID{},
		Email: "doctor@hospital.com",
		Role:  auth.DoctorRole,
	}
}

func TestGetDoctorCalendar(t *testing.T) {
	config := configs.MustLoad("./../../test/testdata/config_valid.json")
	type args struct {
		config        configs.Config
		mockAuth      mockAuthorizer
		dbConn        mock.Connection
		dbMockOptions []mock.DBResultOption
		tokens        *auth.Tokens
		doctorUUID    *uuid.UUID
		year          string
		month         string
		day           string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should get the doctor calendar with appointments and blockers",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")),
					withListAppointmentsResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))),
					withListBlockersResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")),
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
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")),
					withListAppointmentsResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"})),
					withListBlockersResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"})),
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusOK,
		},
		{
			name: "should not get the doctor calendar because the given doctor UUID is wrong",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens:     auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				doctorUUID: nil,
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not get the doctor calendar because the given date parameters are wrong",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens:     auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				doctorUUID: &uuid.UUID{},
				year:       "AAAA",
				month:      "08",
				day:        "10",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not get the doctor calendar because no doctor with given UUID was found",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"})),
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusNotFound,
		},
		{
			name: "should not get the doctor calendar due to a database error while searching for the doctor",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUUIDError(),
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the doctor calendar due to a database error while parsing found doctor",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")),
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the doctor calendar due to a database error while searching for the appointments",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")),
					withListAppointmentsError(),
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the doctor calendar due to a database error while parsing found appointments",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")),
					withListAppointmentsResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, false, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))),
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the doctor calendar due to a database error while searching for the blockers",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")),
					withListAppointmentsResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))),
					withListBlockersError(),
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the doctor calendar due to a database error while parsing found blockers",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")),
					withListAppointmentsResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))),
					withListBlockersResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, false, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")),
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

			mock.MockDBResults(tt.args.dbConn, tt.args.dbMockOptions...)

			req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/calendar/%s/%s/%s/%s", tt.args.doctorUUID, tt.args.year, tt.args.month, tt.args.day), nil)

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
	config := configs.MustLoad("./../../test/testdata/config_valid.json")
	type args struct {
		config        configs.Config
		mockAuth      mockAuthorizer
		dbConn        mock.Connection
		dbMockOptions []mock.DBResultOption
		tokens        *auth.Tokens
		doctorUUID    *uuid.UUID
		year          string
		month         string
		day           string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should get the calendar with appointments and blockers",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")),
					withListAppointmentsResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))),
					withListBlockersResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")),
					withFindPatientByIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "")),
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
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")),
					withListAppointmentsResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"})),
					withListBlockersResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"})),
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusOK,
		},
		{
			name: "should not get the calendar because the date parameters are wrong",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens:     auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				doctorUUID: &uuid.UUID{},
				year:       "AAAA",
				month:      "08",
				day:        "10",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not get the calendar because no doctor associated with the user was found",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"})),
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusForbidden,
		},
		{
			name: "should not get the calendar due to a database error while searching for the doctor",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDError(),
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the calendar due to a database error while parsing the found doctor",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, false, "John Doe", "doctor@hospital.com")),
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the calendar due to a database error while searching for the appointments",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")),
					withListAppointmentsError(),
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},

		{
			name: "should not get the calendar due to a database error while parsing the found appointments",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")),
					withListAppointmentsResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, false, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))),
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the calendar due to a database error while searching for the blockers",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")),
					withListAppointmentsResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"})),
					withListBlockersError(),
				},
				doctorUUID: &uuid.UUID{},
				year:       "2021",
				month:      "08",
				day:        "10",
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "should not get the calendar due to a database error while parsing found blockers",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")),
					withListAppointmentsResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))),
					withListBlockersResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, false, 1, true, false, "")),
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
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")),
					withListAppointmentsResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))),
					withListBlockersResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")),
					withFindPatientByIDError(),
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
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")),
					withListAppointmentsResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))),
					withListBlockersResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")),
					withFindPatientByIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, false, 1, "John Doe", "doctor@hospital.com", "")),
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

			mock.MockDBResults(tt.args.dbConn, tt.args.dbMockOptions...)

			req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/calendar/%s/%s/%s", tt.args.year, tt.args.month, tt.args.day), nil)

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
	config := configs.MustLoad("./../../test/testdata/config_valid.json")
	type args struct {
		config        configs.Config
		mockAuth      mockAuthorizer
		dbConn        mock.Connection
		dbMockOptions []mock.DBResultOption
		tokens        *auth.Tokens
		blockPeriod   *BlockPeriod
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should insert a block period",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")),
					withInsertBlockerResult(sqlmock.NewResult(1, 1)),
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
			name: "should not insert a block period because no doctor associated to the user was found",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"})),
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				blockPeriod: &BlockPeriod{
					StartDate:   time.Now(),
					EndDate:     time.Now().Add(24 * time.Hour),
					Description: nil,
				},
			},
			want: http.StatusForbidden,
		},
		{
			name: "should not insert a block period due to a database error while searching for the doctor",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUUIDError(),
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
			name: "should not insert a block period because the given start date is empty",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")),
				},
				blockPeriod: &BlockPeriod{
					EndDate:     time.Now().Add(24 * time.Hour),
					Description: nil,
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not insert a block period because the given end date is empty",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")),
				},
				blockPeriod: &BlockPeriod{
					StartDate:   time.Now(),
					Description: nil,
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "should not insert a block period because the given end date is after the start date",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")),
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
			name: "should not insert a block period due to a database error while inserting",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")),
					withInsertBlockerError(),
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
			name: "should not insert a block period because no rows are affected after insertion",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockDoctorUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockDoctorUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockDoctorUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "name", "email"}).AddRow(1, uuid.UUID{}, "John Doe", "doctor@hospital.com")),
					withInsertBlockerResult(sqlmock.NewResult(0, 0)),
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

			mock.MockDBResults(tt.args.dbConn, tt.args.dbMockOptions...)

			body, _ := json.Marshal(tt.args.blockPeriod)
			req, _ := http.NewRequest("POST", "/api/v1/calendar/blockers", bytes.NewBuffer(body))

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
	config := configs.MustLoad("./../../test/testdata/config_valid.json")
	type args struct {
		config             configs.Config
		mockAuth           mockAuthorizer
		dbConn             mock.Connection
		dbMockOptions      []mock.DBResultOption
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
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindPatientByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, uuid.UUID{}, 1, "Patient", "patient@hospital.com", "")),
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")),
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")),
					withListAppointmentsResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))),
					withListBlockersResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")),
					withInsertAppointmentResult(sqlmock.NewResult(1, 1)),
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
			name: "should not insert an appointment because no patient associated with the user was found",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindPatientByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"})),
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
			name: "should not insert an appointment due to a database error while searching for the patient",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindPatientByUserIDError(),
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
			name: "should not insert an appointment due to a database error while parsing the found patient",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindPatientByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, false, 1, "Patient", "patient@hospital.com", "")),
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
			name: "should not insert an appointment because no doctor was found with the given UUID",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindPatientByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, uuid.UUID{}, 1, "Patient", "patient@hospital.com", "")),
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"})),
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
			name: "should not insert an appointment due to a database error while searching for the doctor",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUUIDError(),
					withFindPatientByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, uuid.UUID{}, 1, "Patient", "patient@hospital.com", "")),
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
			name: "should not insert an appointment due to a database error while parsing the found doctor",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, false, 1, "John Doe", "doctor@hospital.com", "", "")),
					withFindPatientByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, uuid.UUID{}, 1, "Patient", "patient@hospital.com", "")),
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
			name: "should not insert an appointment because the given error is out of valid range",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
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
			name: "should not insert an appointment because the chosen slot is unavailable",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindPatientByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, uuid.UUID{}, 1, "Patient", "patient@hospital.com", "")),
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")),
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")),
					withListAppointmentsResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))),
					withListBlockersResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")),
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
			name: "should not insert an appointment due to a database error while inserting the appointment",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")),
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")),
					withFindPatientByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, uuid.UUID{}, 1, "Patient", "patient@hospital.com", "")),
					withListAppointmentsResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))),
					withListBlockersResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")),
					withInsertAppointmentError(),
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
			name: "should not insert an appointment due because no rows are affected after insertion",
			args: args{
				config: config,
				dbConn: mock.MustCreateConnectionMock(),
				mockAuth: mockAuthorizer{
					mockValidateToken: func(ctx context.Context, token string) (*auth.User, error) {
						return mockPatientUser(), nil
					},
					mockGetAuthenticatedUser: func(ctx context.Context) (auth.User, error) {
						return *mockPatientUser(), nil
					},
				},
				tokens: auth.MustGenerateTokens(context.TODO(), config.PrivateKey(), *mockPatientUser()),
				dbMockOptions: []mock.DBResultOption{
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")),
					withFindDoctorByUUIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone", "specialty"}).AddRow(1, uuid.UUID{}, 1, "John Doe", "doctor@hospital.com", "", "")),
					withFindPatientByUserIDResult(sqlmock.NewRows([]string{"id", "uuid", "user_id", "name", "email", "mobile_phone"}).AddRow(1, uuid.UUID{}, 1, "Patient", "patient@hospital.com", "")),
					withListAppointmentsResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "patient_id", "date"}).AddRow(1, uuid.UUID{}, 1, 1, time.Date(2021, 8, 10, 10, 0, 0, 0, time.Local))),
					withListBlockersResult(sqlmock.NewRows([]string{"id", "uuid", "doctor_id", "start_date", "end_date", "description"}).AddRow(1, uuid.UUID{}, 1, time.Date(2021, 8, 10, 15, 0, 0, 0, time.Local), time.Date(2021, 8, 10, 16, 0, 0, 0, time.Local), "")),
					withInsertAppointmentResult(sqlmock.NewResult(0, 0)),
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

			mock.MockDBResults(tt.args.dbConn, tt.args.dbMockOptions...)

			body, _ := json.Marshal(tt.args.appointmentRequest)
			req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/calendar/%s/%s/%s/%s", tt.args.doctorUUID, tt.args.year, tt.args.month, tt.args.day), bytes.NewBuffer(body))

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
