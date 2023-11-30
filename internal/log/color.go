package log

import "fmt"

type Color int

// Foreground text colors
const (
	FgBlack Color = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite
)

// ColorMessage returns a colored variant of the message using the ASCII color escape.
func ColorMessage(msg string, color Color) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color, msg)
}
