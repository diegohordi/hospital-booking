package calendar

type Error string

const (
	ErrDoctorNotFound                    = "doctor not found"
	ErrInvalidIdentifier                 = "invalid identifier"
	ErrInvalidDateReference              = "invalid date reference"
	ErrInvalidYearReference              = "invalid year reference - e.g. 2021"
	ErrInvalidMonthReference             = "invalid month reference - e.g. 08"
	ErrInvalidDayReference               = "invalid day reference - e.g. 10"
	ErrOnlyDoctorCanCreateBlocker        = "only a doctor can create a blocker"
	ErrOnlyPatientCanCreateAppointment   = "only a patient can create an appointment"
	ErrSlotNotAvailable                  = "chosen slot is not available"
	ErrOnlyDoctorCanCheckItsAppointments = "only a doctor can check its appointments"
)

func (e Error) Error() string {
	return string(e)
}
