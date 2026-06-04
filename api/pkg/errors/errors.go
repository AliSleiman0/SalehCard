package errors

import stderrors "errors"

// Sentinel errors for common failure conditions.
var (
	ErrNotFound     = stderrors.New("resource not found")
	ErrUnauthorized = stderrors.New("unauthorized")
	ErrForbidden    = stderrors.New("forbidden")
	ErrBadRequest   = stderrors.New("bad request")
	ErrConflict     = stderrors.New("conflict")
	ErrInternal     = stderrors.New("internal error")
)

// AppError is a structured application error that carries a machine-readable
// Code, a human-readable Message, and an optional wrapped cause.
type AppError struct {
	Code    string
	Message string
	Err     error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return e.Message
}

// Unwrap returns the wrapped cause so that errors.Is / errors.As work correctly.
func (e *AppError) Unwrap() error {
	return e.Err
}
