// Package calendar contains handlers, services and structures used to manage the hospital calendar.
package calendar

import (
	"context"
	"fmt"
	"hospital-booking/internal/apierrors"
	"hospital-booking/internal/auth"
	"hospital-booking/internal/configs"
	"hospital-booking/internal/database"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	startWorkHour int32 = 9
	endWorkHour   int32 = 17
)

// Reader determines the methods available to reading the calendars.
type Reader interface {

	// GetDoctorCalendar returns the doctor's daily calendar based on the given parameters.
	GetDoctorCalendar(ctx context.Context, user auth.User, doctorUUID uuid.UUID, date time.Time) ([]Entry, error)

	// GetAppointments returns the doctor's appointments based on the given date.
	GetAppointments(ctx context.Context, user auth.User, date time.Time) ([]Entry, error)
}

// Writer determines the methods available to write on calendars.
type Writer interface {

	// InsertAppointment inserts an appointment to the doctor's calendar.
	InsertAppointment(ctx context.Context, user auth.User, appointmentRequest AppointmentRequest) error
}

// Blocker determines the methods available to manage calendar's blockers.
type Blocker interface {

	// InsertBlocker creates a new calendar blocker.
	InsertBlocker(ctx context.Context, user auth.User, blockPeriod BlockPeriod) error
}

// Service determines the methods used to manage the hospital calendar.
type Service interface {
	Reader
	Writer
	Blocker
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

// hourIsBlocked checks if the given hour is blocked or not.
func (d defaultService) hourIsBlocked(blockers []*BlockPeriod, date time.Time, hour int) bool {
	reference := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, date.Location())
	for _, v := range blockers {
		if (reference.After(v.StartDate) || reference.Equal(v.StartDate)) && (reference.Before(v.EndDate) || reference.Equal(v.EndDate)) {
			return true
		}
	}
	return false
}

func (d defaultService) GetDoctorCalendar(ctx context.Context, user auth.User, doctorUUID uuid.UUID, date time.Time) ([]Entry, error) {
	doctor, err := d.repository.FindDoctorByUUID(ctx, doctorUUID)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred: %w", err)
	}
	if doctor == nil {
		return nil, apierrors.NewAPIError(apierrors.WithDetail(ErrDoctorNotFound), apierrors.WithHTTPStatusCode(http.StatusNotFound))
	}
	appointments, err := d.repository.ListAppointments(ctx, doctor.ID, date)
	if err != nil {
		return nil, err
	}
	blockers, err := d.repository.ListBlockers(ctx, doctor.ID, date)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, endWorkHour-startWorkHour)
	for hour := startWorkHour; hour <= endWorkHour; hour++ {
		available := !d.hourIsBlocked(blockers, date, int(hour))
		if !available {
			continue
		}
		available = !d.hasAppointment(appointments, date, int(hour))
		if !available {
			continue
		}
		entry := Entry{
			Hour:      hour,
			Available: available,
			Patient:   nil,
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// hasAppointment checks if there is some appointment in the given date.
func (d defaultService) hasAppointment(appointments []*Appointment, date time.Time, hour int) bool {
	reference := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, time.Local)
	for _, v := range appointments {
		if reference.Equal(v.Date) {
			return true
		}
	}
	return false
}

// getAppointmentPatient gets the appointment patient, if there is one.
func (d defaultService) getAppointmentPatient(ctx context.Context, appointments []*Appointment, date time.Time, hour int) (*Patient, error) {
	reference := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, time.Local)
	for _, v := range appointments {
		if reference.Equal(v.Date) {
			return d.repository.FindPatientByID(ctx, v.PatientID)
		}
	}
	return nil, nil
}

func (d defaultService) GetAppointments(ctx context.Context, user auth.User, date time.Time) ([]Entry, error) {
	doctor, err := d.repository.FindDoctorByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred: %w", err)
	}
	if doctor == nil {
		return nil, apierrors.NewAPIError(apierrors.WithDetail(ErrOnlyDoctorCanCheckItsAppointments), apierrors.WithHTTPStatusCode(http.StatusForbidden))
	}
	appointments, err := d.repository.ListAppointments(ctx, doctor.ID, date)
	if err != nil {
		return nil, err
	}
	blockers, err := d.repository.ListBlockers(ctx, doctor.ID, date)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, endWorkHour-startWorkHour)
	for hour := startWorkHour; hour <= endWorkHour; hour++ {
		available := !d.hourIsBlocked(blockers, date, int(hour))
		var patient *Patient
		if available {
			available = !d.hasAppointment(appointments, date, int(hour))
			if !available {
				patient, err = d.getAppointmentPatient(ctx, appointments, date, int(hour))
				if err != nil {
					return nil, err
				}
			}
		}
		entry := Entry{
			Hour:      hour,
			Available: available,
			Patient:   patient,
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (d defaultService) InsertBlocker(ctx context.Context, user auth.User, blockPeriod BlockPeriod) error {
	doctor, err := d.repository.FindDoctorByUserID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("an unexpected error occurred: %w", err)
	}
	if doctor == nil {
		return apierrors.NewAPIError(apierrors.WithDetail(ErrOnlyDoctorCanCreateBlocker), apierrors.WithHTTPStatusCode(http.StatusForbidden))
	}
	if err = blockPeriod.Validate(); err != nil {
		return err
	}
	blocker := BlockPeriod{
		Doctor:      doctor,
		UUID:        uuid.New(),
		StartDate:   blockPeriod.StartDate.Truncate(time.Hour),
		EndDate:     blockPeriod.EndDate.Truncate(time.Hour),
		Description: blockPeriod.Description,
	}
	err = d.repository.InsertBlocker(ctx, blocker)
	if err != nil {
		return fmt.Errorf("an unexpected error occurred: %w", err)
	}
	return nil
}

// slotAvailable checks if the given slot is available or not.
func (d defaultService) slotIsAvailable(entries []Entry, hour int32) bool {
	for _, v := range entries {
		if v.Hour == hour {
			return v.Available
		}
	}
	return false
}

func (d defaultService) InsertAppointment(ctx context.Context, user auth.User, appointmentRequest AppointmentRequest) error {
	if err := appointmentRequest.Validate(); err != nil {
		return err
	}
	patient, err := d.repository.FindPatientByUserID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("an unexpected error occurred: %w", err)
	}
	if patient == nil {
		return apierrors.NewAPIError(apierrors.WithDetail(ErrOnlyPatientCanCreateAppointment), apierrors.WithHTTPStatusCode(http.StatusForbidden))
	}
	doctor, err := d.repository.FindDoctorByUUID(ctx, appointmentRequest.DoctorUUID)
	if err != nil {
		return fmt.Errorf("an unexpected error occurred: %w", err)
	}
	if doctor == nil {
		return apierrors.NewAPIError(apierrors.WithDetail(ErrDoctorNotFound), apierrors.WithHTTPStatusCode(http.StatusNotFound))
	}
	entries, err := d.GetDoctorCalendar(ctx, user, appointmentRequest.DoctorUUID, appointmentRequest.Date)
	if err != nil {
		return err
	}
	slotAvailable := d.slotIsAvailable(entries, appointmentRequest.Hour)
	if !slotAvailable {
		return apierrors.NewAPIError(apierrors.WithDetail(ErrSlotNotAvailable), apierrors.WithHTTPStatusCode(http.StatusBadRequest))
	}
	date := appointmentRequest.Date
	appointment := Appointment{
		UUID:    uuid.New(),
		Doctor:  doctor,
		Patient: patient,
		Date:    time.Date(date.Year(), date.Month(), date.Day(), int(appointmentRequest.Hour), 0, 0, 0, date.Location()),
	}
	err = d.repository.InsertAppointment(ctx, appointment)
	if err != nil {
		return fmt.Errorf("an unexpected error occurred: %w", err)
	}
	return nil
}
