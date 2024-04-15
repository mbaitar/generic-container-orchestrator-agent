package persistence

import (
	"os"

	"github.com/mbaitar/gco/agent/internal/log"
	"github.com/mbaitar/gco/agent/internal/state"
)

type LocalController struct {
	channel       chan state.Spec
	stateLocation string
	watcher       *Watcher
}

func NewLocalController(stateFile string) *LocalController {
	controller := &LocalController{
		channel:       make(chan state.Spec),
		stateLocation: stateFile,
	}

	// create watcher
	watcher := NewWatcher(stateFile, controller.handleStateChange)
	controller.watcher = watcher

	// prepare local controller
	_, err := controller.getStateFile()
	if err != nil {
		log.Errorf("Unable to initialize local persistence controller: %v", err)
		os.Exit(1)
	}

	// init watcher
	err = watcher.Init()
	if err != nil {
		log.Errorf("Unable to initialize file watcher: %v", err)
		os.Exit(1)
	}

	// start watching file for changes
	go watcher.Watch()

	log.Debugf("Local state controller initialized (location=%s)", stateFile)
	return controller
}

func (l *LocalController) GetChangeChannel() ChangeChannel {
	return l.channel
}

func (l *LocalController) Persist(spec *state.Spec) error {
	file, err := l.getStateFile()
	if err != nil {
		return err
	}

	// truncate before writing
	err = file.Truncate(0)
	if err != nil {
		return err
	}

	err = WriteJson(file, spec)
	if err != nil {
		return err
	}

	return nil
}

func (l *LocalController) Read() (*state.Spec, error) {
	file, err := l.getStateFile()
	if err != nil {
		return nil, err
	}

	spec := ReadJson(file)
	if spec == nil {
		return nil, nil
	}

	return spec, nil
}

func (l *LocalController) getStateFile() (*os.File, error) {
	if _, err := os.Stat(l.stateLocation); err != nil {
		if os.IsNotExist(err) {
			return os.Create(l.stateLocation)
		} else {
			return nil, err
		}
	} else {
		return os.OpenFile(l.stateLocation, os.O_RDWR, 0644)
	}
}

func (l *LocalController) handleStateChange(s *state.Spec) {
	log.Debugf("Received a state change from state '%s'", l.stateLocation)
	l.channel <- *s
}
