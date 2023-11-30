package persistence

import (
	"bytes"
	"errors"
	"github.com/stretchr/testify/assert"
	"io"
	"revengy.io/gco/agent/pkg/resource"
	"testing"
	"testing/iotest"
)

const testJson = `
{
	"applications": [
		{
			"name": "my-app",
			"image": {"name": "nginx", "tag": "latest"},
			"ports": [
				{
					"containerPort": 80,
					"hostPort": 8080,
					"protocol": "tcp"
				}
			]
		}
	]
}
`

func TestReadJson_InvalidData(t *testing.T) {
	reader := io.NopCloser(bytes.NewBufferString("blah=blah")) // not readable for json reader
	config := ReadJson(reader)
	assert.Nil(t, config, "should not have been able to read config")
}

func TestReadJson_NilReader(t *testing.T) {
	reader := iotest.ErrReader(errors.New("test error"))
	config := ReadJson(io.NopCloser(reader))
	assert.Nil(t, config, "should not be able to read config from NIL stream")
}

func TestReadJson(t *testing.T) {
	reader := io.NopCloser(bytes.NewBufferString(testJson))
	config := ReadJson(reader)

	if assert.NotNil(t, config, "should not be nil") {
		app := config.Applications[0]
		assert.Equal(t, "my-app", app.Name)
		assert.Equal(t, "nginx", app.Image.Name)
		assert.Equal(t, "latest", app.Image.Tag)
		assert.Equal(t, uint16(80), app.Ports[0].ContainerPort)
		assert.Equal(t, uint16(8080), app.Ports[0].HostPort)
		assert.Equal(t, resource.TcpProtocol, app.Ports[0].Protocol)
	}
}
