package domain

import "errors"

var (
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrUserAlreadyExists       = errors.New("user already exists")
	ErrUserNotFound            = errors.New("user not found")
	ErrGroupNotFound           = errors.New("group not found")
	ErrUsernameIsLong          = errors.New("username is long")
	ErrChannelNameIsLong       = errors.New("channel name is long")
	ErrDirectChatNotFound      = errors.New("direct chat not found")
	ErrDirectChatAlreadyExists = errors.New("direct chat already exists")
	ErrMessageNotFound         = errors.New("message not found")
	ErrEmptyMessage            = errors.New("empty message")
	ErrAlreadyMember           = errors.New("already member")
	ErrNotMember               = errors.New("not member")
	ErrTokenExpired            = errors.New("token expired")
	ErrTokenInvalid            = errors.New("token invalid")
	ErrCannotDMYourself        = errors.New("cannot start DM with yourself")
	ErrForbidden               = errors.New("forbidden")
	ErrNotFound                = errors.New("not found")
	ErrPasswordTooShort        = errors.New("password too short")
	ErrPasswordTooLong         = errors.New("password too long")
)
