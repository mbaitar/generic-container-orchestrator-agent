package persistence

import (
	"bytes"
	"errors"
	"regexp"
	"testing"

	"github.com/mabaitar/gco/agent/internal/state"
	"github.com/mabaitar/gco/agent/pkg/resource"
	"github.com/stretchr/testify/assert"
)

type ErrWriter struct{}

func (e ErrWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("error writer")
}

var testState = &state.Spec{
	Applications: []resource.Application{
		{
			Name:  "nginx",
			Image: resource.Image{Name: "nginx", Tag: "latest"},
		},
	},
}

func TestWriteJson(t *testing.T) {
	buffer := bytes.NewBufferString("")
	err := WriteJson(buffer, testState)

	if assert.Nil(t, err, "should not have thrown an error") {
		value := buffer.String()

		// remove newlines to make matching more readable
		re := regexp.MustCompile("\n +")
		value = re.ReplaceAllString(value, "")

		expected := "{\"applications\": [{\"name\": \"nginx\",\"image\": {\"name\": \"nginx\",\"tag\": \"latest\"},\"instances\": 0}],\"feature\": {}\n}"
		assert.Equal(t, expected, value)
	}

}

func TestWriteJson_ErrWriter(t *testing.T) {
	writer := &ErrWriter{}
	err := WriteJson(writer, testState)

	assert.NotNil(t, err, "should have thrown an error upon writing")
}

func TestWriteJson_NilState(t *testing.T) {
	buffer := bytes.NewBufferString("")
	err := WriteJson(buffer, nil)

	assert.NotNil(t, err, "should have thrown an error upon marshalling")
}
