package auth

import (
	"context"
	"database/sql"
	"hospital-booking/internal/database"

	"github.com/google/uuid"
)

const (
	findUserByUUIDQuery    = "SELECT id, uuid, email, role FROM tb_user WHERE uuid = $1"
	findUserByEmailQuery   = "SELECT id, uuid, email, role FROM tb_user WHERE email = $1"
	checkUserPasswordQuery = "SELECT id, password FROM tb_user WHERE email = $1"
)

// Repository provides access to auth data.
type Repository interface {

	// FindUserByUUID finds a user by its UUID.
	FindUserByUUID(ctx context.Context, uuid uuid.UUID) (*User, error)

	// FindUserByEmail finds a user by its email.
	FindUserByEmail(ctx context.Context, email string) (*User, error)

	// CheckUserPassword checks if the stored password is equals to the given password.
	CheckUserPassword(ctx context.Context, email string, password string) (bool, error)
}

type defaultRepository struct {
	dbConn database.Connection
}

// NewRepository creates a new Repository.
func newRepository(dbConn database.Connection) Repository {
	return &defaultRepository{dbConn: dbConn}
}

func (d defaultRepository) FindUserByUUID(ctx context.Context, uuid uuid.UUID) (*User, error) {
	params := make([]interface{}, 1)
	params[0] = uuid.String()
	rows, err := d.dbConn.DB().QueryContext(ctx, findUserByUUIDQuery, params...)
	if err != nil {
		return nil, err
	}
	defer database.CloseRows(rows)
	user := new(User)
	for rows.Next() {
		if err = database.TransformRow(rows, user); err != nil {
			return nil, err
		}
		if user.ID > 0 {
			return user, nil
		}
	}
	return nil, nil
}

func (d defaultRepository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	params := make([]interface{}, 1)
	params[0] = email
	rows, err := d.dbConn.DB().QueryContext(ctx, findUserByEmailQuery, params...)
	if err != nil {
		return nil, err
	}
	defer database.CloseRows(rows)
	user := new(User)
	for rows.Next() {
		if err = database.TransformRow(rows, user); err != nil {
			return nil, err
		}
		if user.ID > 0 {
			return user, nil
		}
	}
	return nil, nil
}

func (d defaultRepository) CheckUserPassword(ctx context.Context, email string, password string) (bool, error) {
	params := make([]interface{}, 1)
	params[0] = email
	row := d.dbConn.DB().QueryRowContext(ctx, checkUserPasswordQuery, params...)
	if row.Err() != nil {
		return false, row.Err()
	}
	id := new(uint64)
	hashedPass := new(string)
	if err := row.Scan(id, hashedPass); err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return ComparePasswords(*hashedPass, password), nil
}
