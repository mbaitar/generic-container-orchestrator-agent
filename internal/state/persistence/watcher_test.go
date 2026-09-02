package persistence

import (
	"errors"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mbaitar/gco/agent/internal/state"
	"github.com/stretchr/testify/assert"
)

func TestWatcher_Init(t *testing.T) {
	dir := t.TempDir()
	location := path.Join(dir, "config.json")
	err := os.WriteFile(location, []byte(testJson), os.ModePerm)
	if err != nil {
		t.Fatalf("unable to create test file: %v", err)
	}

	handler := func(c *state.Spec) {}

	watcher := NewWatcher(location, handler)
	err = watcher.Init()
	assert.Nil(t, err, "should not have thrown an initial error")

	// init after close
	watcher.watcher.Close()

	err = watcher.Init()
	assert.NotNil(t, err, "should have thrown an error on second Init()")
}

func TestWatcher_Watch_JSON(t *testing.T) {
	dir := t.TempDir()
	location := path.Join(dir, "config.json")
	err := os.WriteFile(location, []byte(testJson), os.ModePerm)
	if err != nil {
		t.Fatalf("unable to create test file: %v", err)
	}

	var called atomic.Int32
	handler := func(c *state.Spec) {
		called.Add(1)
	}
	watcher := &Watcher{handler: handler, file: location}

	events := make(chan fsnotify.Event)
	errorChannel := make(chan error)

	done := make(chan struct{})
	go func() {
		defer close(done)
		watcher.watch(events, errorChannel)
	}()

	// errors should not trigger the handler
	errorChannel <- errors.New("test error")
	errorChannel <- errors.New("test error")
	assert.Equal(t, int32(0), called.Load(), "should not have called handler")

	// trigger read
	events <- fsnotify.Event{Op: fsnotify.Write, Name: "test"}
	assert.Eventually(t, func() bool {
		return called.Load() == 1
	}, time.Second, time.Millisecond*5, "should have triggered the handler func")

	// trigger removal
	events <- fsnotify.Event{Op: fsnotify.Remove, Name: "test"}
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, int32(1), called.Load(), "should not have triggered the handler func again")

	// trigger read of incorrect file
	watcher.file = "/does/not/exist"
	events <- fsnotify.Event{Op: fsnotify.Write, Name: "test"}
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, int32(1), called.Load(), "should have resulted in a read error")

	// closing the event channel should stop the watch loop
	close(events)
	<-done
}
