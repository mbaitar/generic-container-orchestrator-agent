package persistence

import (
	"dsync.io/gco/agent/internal/state"
	"errors"
	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"testing"
	"time"
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

	called := 0
	handler := func(c *state.Spec) {
		called += 1
	}
	watcher := NewWatcher(location, handler)

	go func() {
		watcher.watcher.Errors <- errors.New("test error")
		watcher.watcher.Errors <- errors.New("test error")
		assert.Equal(t, 0, called, "should not have called handler")

		// trigger read
		watcher.watcher.Events <- fsnotify.Event{Op: fsnotify.Write, Name: "test"}
		time.Sleep(time.Millisecond * 10) // give event some time to propagate
		assert.Equal(t, 1, called, "should have triggered the handler func")

		// trigger removal
		watcher.watcher.Events <- fsnotify.Event{Op: fsnotify.Remove, Name: "test"}
		time.Sleep(time.Millisecond * 10)
		assert.Equal(t, 1, called, "should not have triggered the handler func again")

		// trigger read of incorrect file
		watcher.file = "/does/not/exist"
		watcher.watcher.Events <- fsnotify.Event{Op: fsnotify.Write, Name: "test"}
		time.Sleep(time.Millisecond * 10)
		assert.Equal(t, 1, called, "should have resulted in a read error")

		// close watcher
		watcher.watcher.Close()
	}()

	watcher.Watch()
}
