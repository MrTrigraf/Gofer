package api

import "errors"

var (
	ErrUnreachable        = errors.New("server unreachable")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrForbidden          = errors.New("forbidden")
	ErrServer             = errors.New("server error")
	ErrUnexpected         = errors.New("unexpected response")
	ErrBadRequest         = errors.New("bad request")
)
