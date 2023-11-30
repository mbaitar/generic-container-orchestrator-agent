package persistence

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"revengy.io/gco/agent/internal/state"
	"revengy.io/gco/agent/pkg/resource"
	"testing"
	"time"
)

func NewTestLocalController(t *testing.T) *LocalController {
	tmp := t.TempDir()
	file := path.Join(tmp, "gco.state")

	return NewLocalController(file)
}

func TestLocalController_GetChangeChannel(t *testing.T) {
	c := NewTestLocalController(t)
	channel := c.GetChangeChannel()

	assert.NotNil(t, channel, "should not have returned nil")
}

func TestLocalController_EmitChangeChannel(t *testing.T) {

	c := NewTestLocalController(t)
	channel := c.GetChangeChannel()

	timeout := make(chan interface{})

	spec := &state.Spec{
		Applications: []resource.Application{
			{
				Name:  "nginx",
				Image: resource.Image{Name: "nginx", Tag: "latest"},
			},
		},
	}

	err := c.Persist(spec)
	if err != nil {
		t.Errorf("should not have thrown an error upon persisting")
		t.FailNow()
	}

	go func() {
		// timeout after 500ms
		time.Sleep(time.Millisecond * 500)
		timeout <- nil
	}()

	// wait for read channel
	select {
	case <-timeout:
		{
			t.Errorf("timeout reached")
			t.FailNow()
		}
	case <-channel:
		{
			t.Logf("successfully received event from read channel")
			return
		}
	}

}

func TestLocalController_Persist(t *testing.T) {
	c := NewTestLocalController(t)

	spec := &state.Spec{
		Applications: []resource.Application{
			{
				Name:  "nginx",
				Image: resource.Image{Name: "nginx", Tag: "latest"},
			},
		},
	}

	err := c.Persist(spec)
	if assert.Nil(t, err, "should not have thrown") {

		file, _ := os.Open(c.stateLocation)
		persisted := ReadJson(file)

		if assert.Equal(t, 1, len(persisted.Applications), "should have persisted 1 application") {
			name := persisted.Applications[0].Name
			img := persisted.Applications[0].Image
			assert.Equal(t, "nginx", name)
			assert.Equal(t, "nginx", img.Name)
			assert.Equal(t, "latest", img.Tag)
		}
	}

}

func TestLocalController_Read(t *testing.T) {
	c := NewTestLocalController(t)

	spec := &state.Spec{
		Applications: []resource.Application{
			{
				Name:  "nginx",
				Image: resource.Image{Name: "nginx", Tag: "latest"},
			},
		},
	}

	err := c.Persist(spec)
	if err != nil {
		t.Errorf("Should not have thrown '%v' on Persist()", err)
		t.FailNow()
	}

	read, err := c.Read()
	if assert.Nil(t, err, "should not have thrown when reading") {
		assert.NotNil(t, read, "should have returned a valid state specification")
		assert.Equal(t, 1, len(read.Applications))

		expectedHash := spec.Applications[0].CalculateHash()
		actualHash := read.Applications[0].CalculateHash()
		assert.Equal(t, expectedHash, actualHash)
	}
}
