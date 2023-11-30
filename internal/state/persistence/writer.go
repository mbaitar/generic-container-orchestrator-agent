package persistence

import (
	"dsync.io/gco/agent/internal/log"
	"dsync.io/gco/agent/internal/state"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
)

// WriteJson writes the given state to the io.Writer.
func WriteJson(writer io.Writer, state *state.Spec) error {
	if state == nil {
		return errors.New("unable to write nil state")
	}

	marshalled, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	n, err := writer.Write(marshalled)
	if err != nil {
		return err
	}

	log.Debugf("Successfully wrote %d bytes (%s)", n, hex.EncodeToString(marshalled))
	return nil
}
