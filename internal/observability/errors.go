package observability

import "fmt"

type AppError struct {
	Code    string
	Op      string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("code=%s op=%s message=%s", e.Code, e.Op, e.Message)
	}
	return fmt.Sprintf("code=%s op=%s message=%s cause=%v", e.Code, e.Op, e.Message, e.Err)
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewAppError(code, op, message string, err error) error {
	return &AppError{
		Code:    code,
		Op:      op,
		Message: message,
		Err:     err,
	}
}
