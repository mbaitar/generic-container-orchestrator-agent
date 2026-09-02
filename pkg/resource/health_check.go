package resource

import applicationv1 "github.com/mbaitar/gco/agent/gen/proto/application/v1"

// HealthCheck describes how the container provider should verify that the
// application is healthy. The test follows the docker convention, e.g.
// ["CMD", "curl", "-f", "http://localhost/"] or ["CMD-SHELL", "curl -f http://localhost/"].
type HealthCheck struct {
	Test               []string `json:"test"`
	IntervalSeconds    uint32   `json:"intervalSeconds,omitempty"`
	TimeoutSeconds     uint32   `json:"timeoutSeconds,omitempty"`
	Retries            uint32   `json:"retries,omitempty"`
	StartPeriodSeconds uint32   `json:"startPeriodSeconds,omitempty"`
}

func (h *HealthCheck) ToHealthCheckV1() *applicationv1.HealthCheck {
	if h == nil {
		return nil
	}

	return &applicationv1.HealthCheck{
		Test:               h.Test,
		IntervalSeconds:    h.IntervalSeconds,
		TimeoutSeconds:     h.TimeoutSeconds,
		Retries:            h.Retries,
		StartPeriodSeconds: h.StartPeriodSeconds,
	}
}

func FromHealthCheckV1(v1 *applicationv1.HealthCheck) *HealthCheck {
	if v1 == nil {
		return nil
	}

	return &HealthCheck{
		Test:               v1.Test,
		IntervalSeconds:    v1.IntervalSeconds,
		TimeoutSeconds:     v1.TimeoutSeconds,
		Retries:            v1.Retries,
		StartPeriodSeconds: v1.StartPeriodSeconds,
	}
}
