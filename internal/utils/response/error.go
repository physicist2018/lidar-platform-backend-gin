package response

import "fmt"

type AppError struct {
	Op  string
	Err error
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func InternalError(op string, err error) error {
	return &AppError{Op: op, Err: err}
}
