package mcpserve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleRequest_Initialize(t *testing.T) {
	client := NewClient()
	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}`),
	}

	resp := handleRequest(req, client)
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.Equal(t, json.RawMessage(`1`), resp.ID)

	result, ok := resp.Result.(initializeResult)
	require.True(t, ok)
	assert.Equal(t, protocolVersion, result.ProtocolVersion)
	assert.Equal(t, "clawide", result.ServerInfo.Name)
	assert.NotNil(t, result.Capabilities.Tools)
}

func TestHandleRequest_Initialized(t *testing.T) {
	client := NewClient()
	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}

	resp := handleRequest(req, client)
	assert.Nil(t, resp, "notifications/initialized is a notification and should not produce a response")
}

func TestHandleRequest_ToolsList(t *testing.T) {
	client := NewClient()
	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
	}

	resp := handleRequest(req, client)
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(toolsListResult)
	require.True(t, ok)
	require.Len(t, result.Tools, 1)
	assert.Equal(t, "clawide_notify", result.Tools[0].Name)
	assert.Contains(t, result.Tools[0].InputSchema.Required, "title")
}

func TestHandleRequest_Ping(t *testing.T) {
	client := NewClient()
	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`3`),
		Method:  "ping",
	}

	resp := handleRequest(req, client)
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)
}

func TestHandleRequest_UnknownMethod(t *testing.T) {
	client := NewClient()
	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`4`),
		Method:  "unknown/method",
	}

	resp := handleRequest(req, client)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32601, resp.Error.Code)
}

func TestHandleToolCall_Notify(t *testing.T) {
	// Start a mock ClawIDE API server
	var receivedBody map[string]interface{}
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/notifications", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"test-id"}`))
	}))
	defer mockServer.Close()

	// Set env vars
	os.Setenv("CLAWIDE_API_URL", mockServer.URL)
	os.Setenv("CLAWIDE_PROJECT_ID", "proj-123")
	os.Setenv("CLAWIDE_SESSION_ID", "sess-456")
	os.Setenv("CLAWIDE_PANE_ID", "pane-789")
	os.Setenv("CLAWIDE_FEATURE_ID", "feat-abc")
	defer func() {
		os.Unsetenv("CLAWIDE_API_URL")
		os.Unsetenv("CLAWIDE_PROJECT_ID")
		os.Unsetenv("CLAWIDE_SESSION_ID")
		os.Unsetenv("CLAWIDE_PANE_ID")
		os.Unsetenv("CLAWIDE_FEATURE_ID")
	}()

	client := NewClient()
	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`5`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"clawide_notify","arguments":{"title":"Build Complete","body":"All tests passed","level":"success","source":"test-runner"}}`),
	}

	resp := handleRequest(req, client)
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	// Verify the notification was sent with correct data
	assert.Equal(t, "Build Complete", receivedBody["title"])
	assert.Equal(t, "All tests passed", receivedBody["body"])
	assert.Equal(t, "success", receivedBody["level"])
	assert.Equal(t, "test-runner", receivedBody["source"])
	assert.Equal(t, "proj-123", receivedBody["project_id"])
	assert.Equal(t, "sess-456", receivedBody["session_id"])
	assert.Equal(t, "pane-789", receivedBody["pane_id"])
	assert.Equal(t, "feat-abc", receivedBody["feature_id"])

	// Verify the response content
	resultJSON, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var result toolCallResult
	require.NoError(t, json.Unmarshal(resultJSON, &result))
	assert.False(t, result.IsError)
	assert.Len(t, result.Content, 1)
	assert.Contains(t, result.Content[0].Text, "Build Complete")
}

func TestHandleToolCall_NotifyMissingTitle(t *testing.T) {
	client := NewClient()
	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`6`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"clawide_notify","arguments":{"body":"no title"}}`),
	}

	resp := handleRequest(req, client)
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error) // Tool errors go in the result, not the JSON-RPC error

	resultJSON, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var result toolCallResult
	require.NoError(t, json.Unmarshal(resultJSON, &result))
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "title is required")
}

func TestHandleToolCall_UnknownTool(t *testing.T) {
	client := NewClient()
	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`7`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"nonexistent_tool","arguments":{}}`),
	}

	resp := handleRequest(req, client)
	require.NotNil(t, resp)

	resultJSON, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var result toolCallResult
	require.NoError(t, json.Unmarshal(resultJSON, &result))
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "unknown tool")
}

func TestHandleToolCall_NotifyDefaultValues(t *testing.T) {
	var receivedBody map[string]interface{}
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"test-id"}`))
	}))
	defer mockServer.Close()

	os.Setenv("CLAWIDE_API_URL", mockServer.URL)
	defer os.Unsetenv("CLAWIDE_API_URL")

	client := NewClient()
	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`8`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"clawide_notify","arguments":{"title":"Simple notification"}}`),
	}

	resp := handleRequest(req, client)
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	// Verify defaults
	assert.Equal(t, "info", receivedBody["level"])
	assert.Equal(t, "claude", receivedBody["source"])
}

func TestHandleToolCall_NotifyServerError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal error")
	}))
	defer mockServer.Close()

	os.Setenv("CLAWIDE_API_URL", mockServer.URL)
	defer os.Unsetenv("CLAWIDE_API_URL")

	client := NewClient()
	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`9`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"clawide_notify","arguments":{"title":"Will fail"}}`),
	}

	resp := handleRequest(req, client)
	require.NotNil(t, resp)

	resultJSON, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var result toolCallResult
	require.NoError(t, json.Unmarshal(resultJSON, &result))
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "status 500")
}

func TestNewClient_DefaultURL(t *testing.T) {
	os.Unsetenv("CLAWIDE_API_URL")
	client := NewClient()
	assert.Equal(t, "http://localhost:9800", client.baseURL)
}

func TestNewClient_CustomURL(t *testing.T) {
	os.Setenv("CLAWIDE_API_URL", "http://localhost:5555")
	defer os.Unsetenv("CLAWIDE_API_URL")
	client := NewClient()
	assert.Equal(t, "http://localhost:5555", client.baseURL)
}
