package log

import (
	"log"
	"os"
)

const flags = log.Ldate | log.Ltime | log.Lshortfile

const (
	WarningColor = FgYellow
	InfoColor    = FgGreen
	ErrorColor   = FgRed
	DebugColor   = FgWhite
)

var (
	warning = log.New(os.Stdout, "WARNING: ", flags)
	info    = log.New(os.Stdout, "INFO: ", flags)
	err     = log.New(os.Stderr, "ERROR: ", flags)
	debug   = log.New(os.Stdout, "DEBUG: ", flags)
)
