package log

import (
	"dsync.io/gco/agent/internal/flag"
	"fmt"
)

// Error prints out an error message to StdErr.
func Error(v ...any) {
	msg := fmt.Sprint(v...)

	if flag.Has(flag.ColoredLogs) {
		msg = ColorMessage(msg, ErrorColor)
	}

	err.Output(2, msg)
}

// Errorf prints out a formatted error message to StdErr.
func Errorf(msg string, args ...any) {
	output := fmt.Sprintf(msg, args...)

	if flag.Has(flag.ColoredLogs) {
		output = ColorMessage(output, ErrorColor)
	}

	err.Output(2, output)
}
