package persistence

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"

	"github.com/mbaitar/gco/agent/internal/log"
	"github.com/mbaitar/gco/agent/internal/state"
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
