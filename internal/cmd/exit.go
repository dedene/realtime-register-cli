package cmd

import "errors"

// Exit code constants.
const (
	CodeSuccess   = 0
	CodeError     = 1
	CodeUsage     = 2
	CodeAuth      = 3
	CodeAPI       = 4
	CodeRateLimit = 5
)

// ExitError wraps an error with a process exit code.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "exit"
}

func (e *ExitError) Unwrap() error { return e.Err }

// ExitCode returns the exit code for an error.
// Returns CodeSuccess for nil, or CodeError if not an ExitError.
func ExitCode(err error) int {
	if err == nil {
		return CodeSuccess
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return CodeError
}
