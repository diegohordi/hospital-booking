package auth

// UnauthorizedError represents the errors returned if the user is not authorized.
type UnauthorizedError struct{}

func NewUnauthorizedError() *UnauthorizedError {
	return &UnauthorizedError{}
}

func (v UnauthorizedError) Error() string {
	return "not authorized"
}
