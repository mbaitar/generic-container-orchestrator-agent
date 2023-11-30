package config

import "fmt"

type Http struct {
	// Enabled is used to enable or disable the HTTP server.
	Enabled bool
	// Port specifies the TCP port to use for listening to HTTP connections.
	Port int
	// Address specifies the address to use for listening to HTTP connections.
	Address string
}

func (h *Http) GetNetworkAddress() string {
	return fmt.Sprintf("%s:%d", h.Address, h.Port)
}
