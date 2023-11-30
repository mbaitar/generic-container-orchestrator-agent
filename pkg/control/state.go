package control

import (
	"os"
	"path"
	"revengy.io/gco/agent/internal/files"
	"revengy.io/gco/agent/internal/flag"
	"revengy.io/gco/agent/internal/log"
	"revengy.io/gco/agent/internal/state"
	"revengy.io/gco/agent/internal/state/persistence"
	"revengy.io/gco/agent/pkg/resource"
)

// StateController is a controller structure which manages the internal desired state of the application.
type StateController struct {
	// desired defines the current internal desired state as it is known in memory.
	desired *state.Spec
	// ctrl defines the control loop which will eventually apply the required changes.
	ctrl *Control
	// persisted represents the persistent state controller used to keep configuration after restarts.
	persisted persistence.Controller
}

func NewStateController(ctrl *Control) *StateController {
	controller := &StateController{ctrl: ctrl}

	dir, err := files.GetDirectory()
	if err != nil {
		log.Errorf("Unable to retrieve configuration directory: %v", err)
		os.Exit(1)
	}

	// create local persistence controller
	persisted := persistence.NewLocalController(path.Join(dir, "gco.state"))

	// get initial state from persisted state
	initial, err := persisted.Read()
	if err != nil {
		log.Errorf("Unable to read initial configuration: %v", err)
		os.Exit(1)
	}

	if initial == nil {
		controller.desired = state.EmptySpec()
	} else {
		controller.desired = initial
	}

	if flag.Has(flag.RemoveAllOnStartup) {
		log.Warn("Applying empty state specification to reset provider")
		ctrl.Apply(*state.EmptySpec())
	}

	// apply the currently known state from the persisted state
	ctrl.Apply(*controller.desired)

	// register change channel to listen for updates
	// TODO: transform to handler reference instead of channel
	go func() {
		for {
			select {
			case spec := <-persisted.GetChangeChannel():
				{
					controller.handleChange(spec)
				}
			}
		}
	}()

	controller.persisted = persisted
	return controller
}

func (s *StateController) CreateApplication(application resource.Application) (*state.Spec, error) {
	err := s.desired.AddApplication(application)
	if err != nil {
		return nil, err
	}

	err = s.persisted.Persist(s.desired)
	if err != nil {
		return nil, err
	}

	return s.desired, nil
}

func (s *StateController) UpdateApplication(application resource.Application) (*state.Spec, error) {

	err := s.desired.UpdateApplication(application)
	if err != nil {
		return nil, err
	}

	err = s.persisted.Persist(s.desired)
	if err != nil {
		return nil, err
	}

	return s.desired, nil
}

func (s *StateController) DeleteApplication(name string) (*state.Spec, error) {

	err := s.desired.RemoveApplication(name)
	if err != nil {
		return nil, err
	}

	err = s.persisted.Persist(s.desired)
	if err != nil {
		return nil, err
	}

	return s.desired, nil
}

func (s *StateController) GetCurrentState() *state.Spec {
	return s.desired
}

func (s *StateController) handleChange(update state.Spec) {
	s.ctrl.Apply(update)
}
