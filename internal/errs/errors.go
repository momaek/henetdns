package errs

import "errors"

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrAuthRequired = errors.New("authentication required")
	ErrRemote       = errors.New("remote request failed")
	ErrParseChanged = errors.New("remote page structure changed")
	ErrStore        = errors.New("local store failure")
)
