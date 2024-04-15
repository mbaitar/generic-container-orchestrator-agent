package persistence

import (
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/mbaitar/gco/agent/internal/log"
	"github.com/mbaitar/gco/agent/internal/state"
)

type ChangeHandler func(s *state.Spec)

type Watcher struct {
	handler ChangeHandler
	file    string
	watcher *fsnotify.Watcher
}

func NewWatcher(file string, handler ChangeHandler) *Watcher {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("unable to start watching: %v", err)
		os.Exit(1)
	}

	return &Watcher{
		handler: handler,
		file:    file,
		watcher: watcher,
	}
}

func (w *Watcher) read() (*state.Spec, error) {
	file, err := os.Open(w.file)
	if err != nil {
		return nil, err
	}

	return ReadJson(file), nil
}

func (w *Watcher) Init() error {
	err := w.watcher.Add(w.file)
	if err != nil {
		log.Warnf("Unable to watch file: %v", err)
		return err
	}

	return nil
}

func (w *Watcher) Watch() {
	defer w.watcher.Close()

	for {
		select {
		case err, ok := <-w.watcher.Errors:
			if !ok {
				log.Debug("Error channel has been closed")
				return
			}

			log.Warnf("Error occurred while watching file: %v", err)
		case event, ok := <-w.watcher.Events:
			if !ok {
				log.Debug("Event channel has been closed")
				return
			}

			if event.Has(fsnotify.Write) {
				log.Debug("File modification detected")
				config, err := w.read()
				if err != nil {
					log.Warnf("Unable to read config from changed file: %v", err)
				} else if config != nil {
					w.handler(config)
				} else {
					log.Warn("File change detected but unable to read config")
				}
			}
		}
	}
}
