package errors

import "errors"

var (
	ErrNotFound     = errors.New("resource not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrInternal     = errors.New("internal error")
)

// PermanentError represents an error that should not be retried (e.g., malformed input).
// Messages causing this error go directly to the DLQ.
type PermanentError struct {
	Reason string
	Err    error
}

func (e *PermanentError) Error() string {
	if e.Err != nil {
		return e.Reason + ": " + e.Err.Error()
	}
	return e.Reason
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

func NewPermanentError(reason string, err error) *PermanentError {
	return &PermanentError{Reason: reason, Err: err}
}

// TransientError represents a temporary error that may succeed on retry (e.g., DB timeout).
type TransientError struct {
	Reason string
	Err    error
}

func (e *TransientError) Error() string {
	if e.Err != nil {
		return e.Reason + ": " + e.Err.Error()
	}
	return e.Reason
}

func (e *TransientError) Unwrap() error {
	return e.Err
}

func NewTransientError(reason string, err error) *TransientError {
	return &TransientError{Reason: reason, Err: err}
}

// IsPermanent checks if an error is a PermanentError.
func IsPermanent(err error) bool {
	var pe *PermanentError
	return errors.As(err, &pe)
}

// IsTransient checks if an error is a TransientError.
func IsTransient(err error) bool {
	var te *TransientError
	return errors.As(err, &te)
}
