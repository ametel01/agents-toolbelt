// Package logx provides minimal shared logging helpers.
package logx

import (
	"fmt"
	"io"
	"log"
)

// Logger writes user-facing output streams.
type Logger struct {
	stdout *log.Logger
	stderr *log.Logger
}

// New constructs a Logger that writes informational and error output separately.
func New(stdout, stderr io.Writer) *Logger {
	return &Logger{
		stdout: log.New(stdout, "", 0),
		stderr: log.New(stderr, "error: ", 0),
	}
}

// Infof writes formatted informational output.
func (l *Logger) Infof(format string, args ...any) {
	l.stdout.Printf(format, args...)
}

// Errorf writes formatted error output.
func (l *Logger) Errorf(format string, args ...any) {
	l.stderr.Printf(format, args...)
}

// UserError formats a message for user-facing errors without wrapping execution errors.
func UserError(format string, args ...any) error {
	return userError{message: fmt.Sprintf(format, args...)}
}

type userError struct {
	message string
}

func (e userError) Error() string {
	return e.message
}
