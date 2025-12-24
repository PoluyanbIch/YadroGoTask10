package core

import "errors"

var (
	ErrBadArguments       = errors.New("arguments are not acceptable")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrInternal           = errors.New("internal error")
	ErrUpdateInProgress   = errors.New("update in progress")

	ErrUserNotFound       = errors.New("User not found")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidUserID      = errors.New("invalid user id")
)
