package service

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	applicationv1 "github.com/mbaitar/gco/agent/gen/proto/application/v1"
	"github.com/stretchr/testify/assert"
)

// TestServiceWrapper_freshRequestPerCall guards against the request struct
// being shared between calls, which leaked fields from earlier requests.
func TestServiceWrapper_freshRequestPerCall(t *testing.T) {
	var received []*applicationv1.CreateApplicationRequest

	handler := serviceWrapper(func(ctx context.Context, req *applicationv1.CreateApplicationRequest) (*applicationv1.CreateApplicationResponse, error) {
		received = append(received, req)
		return &applicationv1.CreateApplicationResponse{}, nil
	})

	// first request sets labels and env
	first := httptest.NewRequest("POST", "/api/v1/applications.create", strings.NewReader(
		`{"application":{"name":"first","labels":{"team":"platform"},"env":{"MODE":"prod"}}}`))
	handler(httptest.NewRecorder(), first)

	// second request omits them entirely
	second := httptest.NewRequest("POST", "/api/v1/applications.create", strings.NewReader(
		`{"application":{"name":"second"}}`))
	handler(httptest.NewRecorder(), second)

	if assert.Equal(t, 2, len(received), "should have handled both requests") {
		assert.Equal(t, "first", received[0].Application.Name)
		assert.Equal(t, "platform", received[0].Application.Labels["team"])

		assert.Equal(t, "second", received[1].Application.Name)
		assert.Empty(t, received[1].Application.Labels, "labels should not leak from the previous request")
		assert.Empty(t, received[1].Application.Env, "env should not leak from the previous request")
	}
}

// TestServiceWrapper_acceptsBothFieldNamings verifies protojson accepts the
// proto names as well as the camelCase spellings used elsewhere.
func TestServiceWrapper_acceptsBothFieldNamings(t *testing.T) {
	var received []*applicationv1.CreateApplicationRequest

	handler := serviceWrapper(func(ctx context.Context, req *applicationv1.CreateApplicationRequest) (*applicationv1.CreateApplicationResponse, error) {
		received = append(received, req)
		return &applicationv1.CreateApplicationResponse{}, nil
	})

	snake := httptest.NewRequest("POST", "/", strings.NewReader(
		`{"application":{"name":"a","network_mode":"bridge","restart_policy":"always"}}`))
	handler(httptest.NewRecorder(), snake)

	camel := httptest.NewRequest("POST", "/", strings.NewReader(
		`{"application":{"name":"b","networkMode":"host","restartPolicy":"no"}}`))
	handler(httptest.NewRecorder(), camel)

	if assert.Equal(t, 2, len(received), "should have handled both requests") {
		assert.Equal(t, "bridge", received[0].Application.NetworkMode)
		assert.Equal(t, "always", received[0].Application.RestartPolicy)
		assert.Equal(t, "host", received[1].Application.NetworkMode)
		assert.Equal(t, "no", received[1].Application.RestartPolicy)
	}
}

// TestWriteHttpError_escapesMessage verifies the error body stays valid JSON
// when the message contains quotes or other JSON metacharacters.
func TestWriteHttpError_escapesMessage(t *testing.T) {
	handler := serviceWrapper(func(ctx context.Context, req *applicationv1.CreateApplicationRequest) (*applicationv1.CreateApplicationResponse, error) {
		return nil, assert.AnError
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"application":{"name":"x\"},\"injected\":true"}}`))
	handler(rec, req)

	body := rec.Body.String()
	assert.True(t, strings.HasPrefix(body, `{"message":`), "should be a message object, got: %s", body)
	assert.NotContains(t, body, `"injected":true`, "metacharacters should be escaped, got: %s", body)
}
