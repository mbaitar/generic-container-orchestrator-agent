package config

import "fmt"

type Grpc struct {
	// Enabled is used to enable or disable the gRPC server.
	Enabled bool
	// Port specifies the TCP port to use for listening to gRPC connections.
	Port int
	// Address specifies the address to use for listening to gRPC connections.
	Address string
	// EnableReflection enables gRPC reflection mode (useful for development).
	EnableReflection bool
}

func (g *Grpc) GetNetworkAddress() string {
	return fmt.Sprintf("%s:%d", g.Address, g.Port)
}
