package log

import (
	"dsync.io/gco/agent/internal/flag"
	"fmt"
)

// Debug prints out a debug message to StdOut.
func Debug(v ...any) {
	msg := fmt.Sprint(v...)

	if flag.Has(flag.ColoredLogs) {
		msg = ColorMessage(msg, DebugColor)
	}

	debug.Output(2, msg)
}

// Debugf prints out a formatted debug message to StdOut.
func Debugf(msg string, args ...any) {
	output := fmt.Sprintf(msg, args...)

	if flag.Has(flag.ColoredLogs) {
		output = ColorMessage(output, DebugColor)
	}

	debug.Output(2, output)
}
