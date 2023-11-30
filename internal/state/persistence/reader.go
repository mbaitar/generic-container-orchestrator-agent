package persistence

import (
	"dsync.io/gco/agent/internal/log"
	"dsync.io/gco/agent/internal/state"
	"encoding/json"
	"io"
)

// ReadJson reads the state from the io.ReadCloser.
func ReadJson(reader io.ReadCloser) *state.Spec {

	value, err := io.ReadAll(reader)
	if err != nil {
		log.Errorf("failed to read config: %v", err)
		return nil
	}

	config := &state.Spec{}
	err = json.Unmarshal(value, config)
	if err != nil {
		log.Errorf("failed to unmarshal data: %v", err)
		return nil
	}

	return config
}
