package calendar

import (
	"context"
	"fmt"
	"hospital-booking/internal/database"
	"time"

	"github.com/google/uuid"
)

const (
	findDoctorByUUIDQuery    = "SELECT id, uuid, user_id, name, email, mobile_phone, specialty FROM tb_doctor WHERE uuid = $1"
	findDoctorByUserIDQuery  = "SELECT id, uuid, user_id, name, email, mobile_phone, specialty FROM tb_doctor WHERE user_id = $1"
	findPatientByIDQuery     = "SELECT id, uuid, user_id, name, email, mobile_phone FROM tb_patient WHERE id = $1"
	findPatientByUUIDQuery   = "SELECT id, uuid, user_id, name, email, mobile_phone FROM tb_patient WHERE uuid = $1"
	findPatientByUserIDQuery = "SELECT id, uuid, user_id, name, email, mobile_phone FROM tb_patient WHERE user_id = $1"
	insertBlockerQuery       = "INSERT INTO tb_block_period (uuid, doctor_id, start_date, end_date, description) VALUES ($1, $2, $3, $4, $5)"
	listBlockersQuery        = "SELECT id, uuid, doctor_id, start_date, end_date, description FROM tb_block_period WHERE doctor_id = $1 AND $2 BETWEEN date_trunc('day', start_date) AND date_trunc('day', end_date)"
	insertAppointmentQuery   = "INSERT INTO tb_appointment (uuid, doctor_id, patient_id, date) VALUES ($1, $2, $3, $4)"
	listAppointmentsQuery    = "SELECT id, uuid, doctor_id, patient_id, date FROM tb_appointment WHERE doctor_id = $1 AND $2 = date_trunc('day', date)"
)

// Repository provides access to booking data.
type Repository interface {

	// FindDoctorByUUID finds a doctor by its UUID.
	FindDoctorByUUID(ctx context.Context, uuid uuid.UUID) (*Doctor, error)

	// FindDoctorByUserID finds a doctor by its user ID.
	FindDoctorByUserID(ctx context.Context, userID int64) (*Doctor, error)

	// FindPatientByID finds a doctor by its ID.
	FindPatientByID(ctx context.Context, ID int64) (*Patient, error)

	// FindPatientByUUID finds a doctor by its UUID.
	FindPatientByUUID(ctx context.Context, uuid uuid.UUID) (*Patient, error)

	// FindPatientByUserID finds a patient by its user ID.
	FindPatientByUserID(ctx context.Context, userID int64) (*Patient, error)

	// InsertBlocker inserts a new block period.
	InsertBlocker(ctx context.Context, blockPeriod BlockPeriod) error

	// ListBlockers lists the doctor's blockers accordingly the given date.
	ListBlockers(ctx context.Context, doctorID int64, date time.Time) ([]*BlockPeriod, error)

	// InsertAppointment inserts a new appointment.
	InsertAppointment(ctx context.Context, appointment Appointment) error

	// ListAppointments lists the doctor's appointments.
	ListAppointments(ctx context.Context, doctorID int64, date time.Time) ([]*Appointment, error)
}

type defaultRepository struct {
	dbConn database.Connection
}

// NewRepository creates a new Repository.
func newRepository(dbConn database.Connection) Repository {
	return &defaultRepository{dbConn: dbConn}
}

func (d defaultRepository) FindDoctorByUserID(ctx context.Context, userID int64) (*Doctor, error) {
	ctx, cancel := d.dbConn.CreateContext(ctx)
	defer cancel()
	params := make([]interface{}, 1)
	params[0] = userID
	rows, err := d.dbConn.DB().QueryContext(ctx, findDoctorByUserIDQuery, params...)
	if err != nil {
		return nil, err
	}
	defer database.CloseRows(rows)
	doctor := new(Doctor)
	for rows.Next() {
		if err = database.TransformRow(rows, doctor); err != nil {
			return nil, err
		}
		if doctor.ID > 0 {
			return doctor, nil
		}
	}
	return nil, nil
}

func (d defaultRepository) FindPatientByUserID(ctx context.Context, userID int64) (*Patient, error) {
	ctx, cancel := d.dbConn.CreateContext(ctx)
	defer cancel()
	params := make([]interface{}, 1)
	params[0] = userID
	rows, err := d.dbConn.DB().QueryContext(ctx, findPatientByUserIDQuery, params...)
	if err != nil {
		return nil, err
	}
	defer database.CloseRows(rows)
	patient := new(Patient)
	for rows.Next() {
		if err = database.TransformRow(rows, patient); err != nil {
			return nil, err
		}
		if patient.ID > 0 {
			return patient, nil
		}
	}
	return nil, nil
}

func (d defaultRepository) FindDoctorByUUID(ctx context.Context, uuid uuid.UUID) (*Doctor, error) {
	ctx, cancel := d.dbConn.CreateContext(ctx)
	defer cancel()
	params := make([]interface{}, 1)
	params[0] = uuid
	rows, err := d.dbConn.DB().QueryContext(ctx, findDoctorByUUIDQuery, params...)
	if err != nil {
		return nil, err
	}
	defer database.CloseRows(rows)
	doctor := new(Doctor)
	for rows.Next() {
		if err = database.TransformRow(rows, doctor); err != nil {
			return nil, err
		}
		if doctor.ID > 0 {
			return doctor, nil
		}
	}
	return nil, nil
}

func (d defaultRepository) FindPatientByID(ctx context.Context, ID int64) (*Patient, error) {
	ctx, cancel := d.dbConn.CreateContext(ctx)
	defer cancel()
	params := make([]interface{}, 1)
	params[0] = ID
	rows, err := d.dbConn.DB().QueryContext(ctx, findPatientByIDQuery, params...)
	if err != nil {
		return nil, err
	}
	defer database.CloseRows(rows)
	patient := new(Patient)
	for rows.Next() {
		if err = database.TransformRow(rows, patient); err != nil {
			return nil, err
		}
		if patient.ID > 0 {
			return patient, nil
		}
	}
	return nil, nil
}

func (d defaultRepository) FindPatientByUUID(ctx context.Context, uuid uuid.UUID) (*Patient, error) {
	ctx, cancel := d.dbConn.CreateContext(ctx)
	defer cancel()
	params := make([]interface{}, 1)
	params[0] = uuid
	rows, err := d.dbConn.DB().QueryContext(ctx, findPatientByUUIDQuery, params...)
	if err != nil {
		return nil, err
	}
	defer database.CloseRows(rows)
	patient := new(Patient)
	for rows.Next() {
		if err = database.TransformRow(rows, patient); err != nil {
			return nil, err
		}
		if patient.ID > 0 {
			return patient, nil
		}
	}
	return nil, nil
}

func (d defaultRepository) InsertBlocker(ctx context.Context, blockPeriod BlockPeriod) error {
	ctx, cancel := d.dbConn.CreateContext(ctx)
	defer cancel()
	params := make([]interface{}, 5)
	params[0] = blockPeriod.UUID
	params[1] = blockPeriod.Doctor.ID
	params[2] = blockPeriod.StartDate
	params[3] = blockPeriod.EndDate
	params[4] = blockPeriod.Description
	result, err := d.dbConn.DB().ExecContext(ctx, insertBlockerQuery, params...)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("blocker not inserted")
	}
	return nil
}

func (d defaultRepository) InsertAppointment(ctx context.Context, appointment Appointment) error {
	ctx, cancel := d.dbConn.CreateContext(ctx)
	defer cancel()
	params := make([]interface{}, 4)
	params[0] = appointment.UUID
	params[1] = appointment.Doctor.ID
	params[2] = appointment.Patient.ID
	params[3] = appointment.Date
	result, err := d.dbConn.DB().ExecContext(ctx, insertAppointmentQuery, params...)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("appointment not inserted")
	}
	return nil
}

func (d defaultRepository) ListBlockers(ctx context.Context, doctorID int64, date time.Time) ([]*BlockPeriod, error) {
	ctx, cancel := d.dbConn.CreateContext(ctx)
	defer cancel()
	params := make([]interface{}, 2)
	params[0] = doctorID
	params[1] = date.Truncate(24 * time.Hour)
	rows, err := d.dbConn.DB().QueryContext(ctx, listBlockersQuery, params...)
	if err != nil {
		return nil, err
	}
	defer database.CloseRows(rows)
	blockers := make([]*BlockPeriod, 0)
	for rows.Next() {
		blocker := new(BlockPeriod)
		if err = database.TransformRow(rows, blocker); err != nil {
			return nil, err
		}
		blockers = append(blockers, blocker)
	}
	return blockers, nil
}

func (d defaultRepository) ListAppointments(ctx context.Context, doctorID int64, date time.Time) ([]*Appointment, error) {
	ctx, cancel := d.dbConn.CreateContext(ctx)
	defer cancel()
	params := make([]interface{}, 2)
	params[0] = doctorID
	params[1] = date.Truncate(24 * time.Hour)
	rows, err := d.dbConn.DB().QueryContext(ctx, listAppointmentsQuery, params...)
	if err != nil {
		return nil, err
	}
	defer database.CloseRows(rows)
	appointments := make([]*Appointment, 0)
	for rows.Next() {
		appointment := new(Appointment)
		if err = database.TransformRow(rows, appointment); err != nil {
			return nil, err
		}
		appointments = append(appointments, appointment)
	}
	return appointments, nil
}
