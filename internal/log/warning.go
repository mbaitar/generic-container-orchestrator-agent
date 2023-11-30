package log

import (
	"dsync.io/gco/agent/internal/flag"
	"fmt"
)

// Warn prints out a warning message to StdOut.
func Warn(v ...any) {
	msg := fmt.Sprint(v...)

	if flag.Has(flag.ColoredLogs) {
		msg = ColorMessage(msg, WarningColor)
	}

	warning.Output(2, msg)
}

// Warnf prints out a formatted warning message to StdOut.
func Warnf(msg string, args ...any) {
	output := fmt.Sprintf(msg, args...)

	if flag.Has(flag.ColoredLogs) {
		output = ColorMessage(output, WarningColor)
	}

	warning.Output(2, output)
}
