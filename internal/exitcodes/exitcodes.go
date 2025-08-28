package exitcodes

import (
	"errors"
	"os/exec"
)

// Stable exit codes for agents/CI.
// 0 is success; everything else is a specific class of failure.
const (
	OK               = 0
	InvalidArgs      = 2 // bad/missing flags, wrong arity, etc.
	ProfileNotFound  = 3 // named profile not found
	ConfigReadWrite  = 4 // read/write failure (files on disk)
	ConnectionFailed = 5 // --test-conn failed
	EnvWriteFailed   = 6 // --env-file write/merge failure
	JSONError        = 7 // JSON output failure

	ChildSpawnFailed = 20 // exec could not start (not found, perms, etc.)

	Unknown = 1 // default fallback
)

// CodeError wraps an error with an exit code.
// Msg is optional; if empty and Err is set, Err.Error() is printed.
type CodeError struct {
	Code int
	Msg  string
	Err  error
}

func (e *CodeError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return ""
}
func (e *CodeError) Unwrap() error { return e.Err }

// New constructs an error with code + message.
func New(code int, msg string) error {
	return &CodeError{Code: code, Msg: msg}
}

// Wrap attaches an exit code to an existing error.
func Wrap(code int, err error) error {
	if err == nil {
		return &CodeError{Code: code}
	}
	return &CodeError{Code: code, Err: err}
}

// WithCode returns an error carrying only an exit code (no message).
func WithCode(code int) error { return &CodeError{Code: code} }

// FromError extracts the exit code to return to the OS.
func FromError(err error) int {
	if err == nil {
		return OK
	}
	var ce *CodeError
	if errors.As(err, &ce) {
		if ce.Code != 0 {
			return ce.Code
		}
		return Unknown
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode() // propagate child's exit code
	}
	return Unknown
}

// Message returns the human-readable message to print to stderr.
func Message(err error) string {
	if err == nil {
		return ""
	}
	var ce *CodeError
	if errors.As(err, &ce) {
		return ce.Error()
	}
	return err.Error()
}
