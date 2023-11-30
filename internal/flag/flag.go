package flag

type Mask uint64

const (
	// IgnoreInstanceDiff is used to enable or disable the checking of the application instances.
	// Some provider.Provider do not support multiple application instances.
	IgnoreInstanceDiff Mask = 1 << iota

	// RemoveAllOnStartup resets the external container provider system when the agent launches.
	RemoveAllOnStartup

	// ColoredLogs enables colors for log messages.
	ColoredLogs
)

var bits Mask

// Reset will clear all the active flags.
func Reset() {
	bits = 0
}

// Set will enable the specified Mask.
func Set(mask Mask) {
	bits = bits | mask
}

// Has will check if the specified Mask has been enabled.
func Has(mask Mask) bool {
	return bits&mask == mask
}

// Clear will remove the specified Mask.
func Clear(mask Mask) {
	bits = bits &^ mask
}
