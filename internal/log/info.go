package log

import (
	"dsync.io/gco/agent/internal/flag"
	"fmt"
)

// Info prints out an informative message to StdOut.
func Info(v ...any) {
	msg := fmt.Sprint(v...)

	if flag.Has(flag.ColoredLogs) {
		msg = ColorMessage(msg, InfoColor)
	}

	info.Output(2, msg)
}

// Infof prints out a formatted informative message to StdOut.
func Infof(msg string, args ...any) {
	output := fmt.Sprintf(msg, args...)

	if flag.Has(flag.ColoredLogs) {
		output = ColorMessage(output, InfoColor)
	}

	info.Output(2, output)
}
