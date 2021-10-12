package calendar

import (
	"hospital-booking/internal/apierrors"
	"time"

	"github.com/google/uuid"
)

type Patient struct {
	ID          int64     `json:"-" dbfield:"id"`
	UserID      int64     `json:"-" dbfield:"user_id"`
	UUID        uuid.UUID `json:"uuid" dbfield:"uuid"`
	Name        string    `json:"name" dbfield:"name"`
	Email       string    `json:"email" dbfield:"email"`
	MobilePhone string    `json:"mobile_phone" dbfield:"mobile_phone"`
}

type Doctor struct {
	ID          int64     `json:"-" dbfield:"id"`
	UserID      int64     `json:"-" dbfield:"user_id"`
	UUID        uuid.UUID `json:"uuid" dbfield:"uuid"`
	Name        string    `json:"name" dbfield:"name"`
	Email       string    `json:"email" dbfield:"email"`
	MobilePhone string    `json:"mobile_phone" dbfield:"mobile_phone"`
	Specialty   string    `json:"specialty" dbfield:"specialty"`
}

type BlockPeriod struct {
	ID          int64     `json:"-" dbfield:"id"`
	UUID        uuid.UUID `json:"uuid,omitempty" dbfield:"uuid"`
	DoctorID    int64     `json:"-" dbfield:"doctor_id"`
	Doctor      *Doctor   `json:"doctor,omitempty"`
	StartDate   time.Time `json:"start_date,omitempty" dbfield:"start_date"`
	EndDate     time.Time `json:"end_date,omitempty" dbfield:"end_date"`
	Description *string   `json:"description" dbfield:"description"`
}

// Validate validates if the block period is valid.
func (b BlockPeriod) Validate() error {
	if b.StartDate.IsZero() {
		return apierrors.NewValidationError("start_date", "required")
	}
	if b.EndDate.IsZero() {
		return apierrors.NewValidationError("end_date", "required")
	}
	if b.EndDate.Before(b.StartDate) {
		return apierrors.NewValidationError("end_date", "invalid period")
	}
	return nil
}

type Appointment struct {
	ID        int64     `json:"-" dbfield:"id"`
	UUID      uuid.UUID `json:"uuid" dbfield:"uuid"`
	Doctor    *Doctor   `json:"doctor"`
	DoctorID  int64     `json:"-" dbfield:"doctor_id"`
	Patient   *Patient  `json:"patient"`
	PatientID int64     `json:"-" dbfield:"patient_id"`
	Date      time.Time `json:"date" dbfield:"date"`
}

type AppointmentRequest struct {
	Hour       int32 `json:"hour"`
	DoctorUUID uuid.UUID
	Date       time.Time
}

// Validate checks if the given request is valid.
func (a AppointmentRequest) Validate() error {
	if !(a.Hour >= startWorkHour && a.Hour <= endWorkHour) {
		return apierrors.NewValidationError("hour", "out of working hours")
	}
	if a.Date.IsZero() {
		return apierrors.NewValidationError("date", "required")
	}
	return nil
}

type Entry struct {
	Hour      int32    `json:"hour"`
	Available bool     `json:"available"`
	Patient   *Patient `json:"patient,omitempty"`
}
