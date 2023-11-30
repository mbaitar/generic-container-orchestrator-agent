package resource

import applicationv1 "revengy.io/gco/agent/gen/proto/application/v1"

type Protocol string

const (
	TcpProtocol Protocol = "tcp"
	UdpProtocol Protocol = "udp"
)

type Port struct {
	ContainerPort uint16   `json:"containerPort"`
	HostPort      uint16   `json:"hostPort"`
	Protocol      Protocol `json:"protocol"`
}

func (p *Port) ToPortV1() *applicationv1.Port {
	v1 := &applicationv1.Port{}
	v1.ContainerPort = uint32(p.ContainerPort)
	v1.HostPort = uint32(p.HostPort)

	if p.Protocol == TcpProtocol {
		v1.Protocol = applicationv1.Protocol_PROTOCOL_TCP
	} else if p.Protocol == UdpProtocol {
		v1.Protocol = applicationv1.Protocol_PROTOCOL_UDP
	} else {
		v1.Protocol = applicationv1.Protocol_PROTOCOL_UNSPECIFIED
	}

	return v1
}

func ToPortsV1(ports []Port) []*applicationv1.Port {
	v1s := make([]*applicationv1.Port, 0)

	for _, p := range ports {
		v1s = append(v1s, p.ToPortV1())
	}

	return v1s
}

func FromPortV1(v1 *applicationv1.Port) *Port {
	if v1 == nil {
		return nil
	}

	return &Port{
		ContainerPort: uint16(v1.ContainerPort),
		HostPort:      uint16(v1.HostPort),
		Protocol:      FromProtocolV1(v1.Protocol),
	}
}

func FromPortsV1(v1 []*applicationv1.Port) []Port {
	if v1 == nil {
		return make([]Port, 0)
	}

	ports := make([]Port, len(v1))
	for i, p := range v1 {
		ports[i] = *FromPortV1(p)
	}
	return ports
}

func FromProtocolV1(v1 applicationv1.Protocol) Protocol {
	if v1 == applicationv1.Protocol_PROTOCOL_TCP {
		return TcpProtocol
	} else if v1 == applicationv1.Protocol_PROTOCOL_UDP {
		return UdpProtocol
	} else {
		return "unknown"
	}
}
