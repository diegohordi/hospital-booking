package calendar

import (
	"encoding/json"
	"fmt"
	"hospital-booking/internal/apierrors"
	"hospital-booking/internal/auth"
	"hospital-booking/internal/configs"
	"hospital-booking/internal/database"
	"hospital-booking/internal/logging"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/google/uuid"

	"github.com/go-chi/chi/v5"
)

type httpHandler struct {
	authorizer auth.Authorizer
	service    Service
	logger     *log.Logger
}

// Setup setups the routes handled by auth context.
func Setup(router *chi.Mux, logger *log.Logger, authorizer auth.Authorizer, config configs.Config, dbConn database.Connection) {
	handler := &httpHandler{logger: logger, authorizer: authorizer, service: NewService(config, dbConn)}

	// protected routes, only for patients
	router.Group(func(group chi.Router) {
		group.Use(auth.JwtValidator(authorizer))
		group.Use(auth.AllowedRole(authorizer, auth.PatientRole))
		group.Get("/api/v1/calendar/{doctorUUID}/{year}/{month}/{day}", handler.GetDoctorCalendar)
		group.Post("/api/v1/calendar/{doctorUUID}/{year}/{month}/{day}", handler.InsertAppointment)
	})

	// protected routes, only for doctors
	router.Group(func(group chi.Router) {
		group.Use(auth.JwtValidator(authorizer))
		group.Use(auth.AllowedRole(authorizer, auth.DoctorRole))
		group.Get("/api/v1/calendar/{year}/{month}/{day}", handler.GetAppointments)
		group.Post("/api/v1/calendar/blockers", handler.InsertBlockPeriod)
	})
}

func (h httpHandler) writeResponseError(w http.ResponseWriter, r *http.Request, err error) {
	logging.PrintlnError(h.logger, fmt.Sprint(r.Context().Value(middleware.RequestIDKey), " ", err))
	switch errType := err.(type) {
	case *auth.UnauthorizedError:
		w.WriteHeader(http.StatusUnauthorized)
		return
	case *apierrors.ValidationError:
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(err)
		return
	case *apierrors.APIError:
		w.WriteHeader(errType.HTTPStatusCode())
		_ = json.NewEncoder(w).Encode(err)
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
}

// parseDate parses the given parameters into a valid time.
func (h httpHandler) parseDateParameters(r *http.Request) (time.Time, error) {
	var zeroTime time.Time
	year := chi.URLParam(r, "year")
	month := chi.URLParam(r, "month")
	day := chi.URLParam(r, "day")
	if year == "" || month == "" || day == "" {
		return zeroTime, apierrors.NewAPIError(apierrors.WithDetail(ErrInvalidDateReference), apierrors.WithHTTPStatusCode(http.StatusNotFound))
	}
	yearInt, err := strconv.Atoi(year)
	if len(year) != 4 || err != nil {
		return zeroTime, apierrors.NewAPIError(apierrors.WithDetail(ErrInvalidYearReference), apierrors.WithHTTPStatusCode(http.StatusBadRequest))
	}
	monthInt, err := strconv.Atoi(month)
	if len(month) > 2 || err != nil {
		return zeroTime, apierrors.NewAPIError(apierrors.WithDetail(ErrInvalidMonthReference), apierrors.WithHTTPStatusCode(http.StatusBadRequest))
	}
	dayInt, err := strconv.Atoi(day)
	if len(day) > 2 || err != nil {
		return zeroTime, apierrors.NewAPIError(apierrors.WithDetail(ErrInvalidDayReference), apierrors.WithHTTPStatusCode(http.StatusBadRequest))
	}
	concatDate := fmt.Sprintf("%4d-%02d-%02d", yearInt, monthInt, dayInt)
	date, err := time.Parse("2006-01-02", concatDate)
	if err != nil {
		return zeroTime, apierrors.NewAPIError(apierrors.WithDetail(ErrInvalidDateReference), apierrors.WithHTTPStatusCode(http.StatusBadRequest))
	}
	return date, nil
}

// parseUUIDParameter parses a UUID parameter into a valid UUID.
func (h httpHandler) parseUUIDParameter(parName string, r *http.Request) (uuid.UUID, error) {
	zeroUUID := uuid.UUID{}
	uuidPar := chi.URLParam(r, parName)
	if uuidPar == "" {
		return zeroUUID, apierrors.NewAPIError(apierrors.WithDetail(ErrInvalidIdentifier), apierrors.WithHTTPStatusCode(http.StatusNotFound))
	}
	parsedUUID, err := uuid.Parse(uuidPar)
	if err != nil {
		return zeroUUID, apierrors.NewAPIError(apierrors.WithDetail(ErrInvalidIdentifier), apierrors.WithHTTPStatusCode(http.StatusBadRequest))
	}
	return parsedUUID, nil
}

func (h httpHandler) GetDoctorCalendar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	date, err := h.parseDateParameters(r)
	if err != nil {
		h.writeResponseError(w, r, err)
		return
	}
	doctorUUID, err := h.parseUUIDParameter("doctorUUID", r)
	if err != nil {
		h.writeResponseError(w, r, err)
	}
	user, err := h.authorizer.GetAuthenticatedUser(ctx)
	if err != nil {
		h.writeResponseError(w, r, err)
		return
	}
	entries, err := h.service.GetDoctorCalendar(ctx, user, doctorUUID, date)
	if err != nil {
		h.writeResponseError(w, r, err)
		return
	}
	_ = json.NewEncoder(w).Encode(entries)
}

func (h httpHandler) InsertAppointment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	date, err := h.parseDateParameters(r)
	if err != nil {
		h.writeResponseError(w, r, err)
		return
	}
	doctorUUID, err := h.parseUUIDParameter("doctorUUID", r)
	if err != nil {
		h.writeResponseError(w, r, err)
		return
	}
	user, err := h.authorizer.GetAuthenticatedUser(ctx)
	if err != nil {
		h.writeResponseError(w, r, err)
		return
	}
	appointmentRequest := &AppointmentRequest{}
	if err = json.NewDecoder(r.Body).Decode(appointmentRequest); err != nil {
		h.writeResponseError(w, r, err)
		return
	}
	appointmentRequest.DoctorUUID = doctorUUID
	appointmentRequest.Date = date
	err = h.service.InsertAppointment(ctx, user, *appointmentRequest)
	if err != nil {
		h.writeResponseError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h httpHandler) GetAppointments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	date, err := h.parseDateParameters(r)
	if err != nil {
		h.writeResponseError(w, r, err)
		return
	}
	user, err := h.authorizer.GetAuthenticatedUser(ctx)
	if err != nil {
		h.writeResponseError(w, r, err)
		return
	}
	entries, err := h.service.GetAppointments(ctx, user, date)
	if err != nil {
		h.writeResponseError(w, r, err)
		return
	}
	_ = json.NewEncoder(w).Encode(entries)
}

func (h httpHandler) InsertBlockPeriod(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, err := h.authorizer.GetAuthenticatedUser(ctx)
	if err != nil {
		h.writeResponseError(w, r, err)
		return
	}
	blockPeriod := &BlockPeriod{}
	if err = json.NewDecoder(r.Body).Decode(blockPeriod); err != nil {
		h.writeResponseError(w, r, err)
		return
	}
	err = h.service.InsertBlocker(ctx, user, *blockPeriod)
	if err != nil {
		h.writeResponseError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
