package errors

import "fmt"

type MultiError struct {
	Errors []error
}

func (e *MultiError) Error() string {
	return fmt.Sprintf("%v", e.Errors)
}

func NewMultiError(errors []error) *MultiError {
	return &MultiError{Errors: errors}
}
