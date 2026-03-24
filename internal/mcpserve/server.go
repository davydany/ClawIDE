package mcpserve

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/davydany/ClawIDE/internal/version"
)

const (
	jsonrpcVersion  = "2.0"
	protocolVersion = "2024-11-05"
	serverName      = "clawide"
)

// JSON-RPC request/response types

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"` // can be number, string, or null
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCP protocol types

type initializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    capabilities `json:"capabilities"`
	ServerInfo      serverInfo   `json:"serverInfo"`
}

type capabilities struct {
	Tools *toolsCapability `json:"tools,omitempty"`
}

type toolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type toolsListResult struct {
	Tools []toolDefinition `json:"tools"`
}

type toolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

type inputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

type toolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type toolCallResult struct {
	Content []contentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Run starts the MCP stdio server. It reads JSON-RPC requests from stdin
// and writes responses to stdout. All logging goes to stderr.
func Run() {
	// Redirect log output to stderr so it doesn't corrupt the JSON-RPC stream
	log.SetOutput(os.Stderr)
	log.SetPrefix("[clawide-mcp] ")

	client := NewClient()
	scanner := bufio.NewScanner(os.Stdin)
	// MCP messages can be large; increase buffer to 1MB
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req jsonRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			log.Printf("Failed to parse request: %v", err)
			writeError(encoder, nil, -32700, "Parse error")
			continue
		}

		resp := handleRequest(&req, client)
		if resp != nil {
			if err := encoder.Encode(resp); err != nil {
				log.Printf("Failed to write response: %v", err)
			}
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		log.Printf("Scanner error: %v", err)
	}
}

func handleRequest(req *jsonRPCRequest, client *Client) *jsonRPCResponse {
	switch req.Method {
	case "initialize":
		return &jsonRPCResponse{
			JSONRPC: jsonrpcVersion,
			ID:      req.ID,
			Result: initializeResult{
				ProtocolVersion: protocolVersion,
				Capabilities: capabilities{
					Tools: &toolsCapability{},
				},
				ServerInfo: serverInfo{
					Name:    serverName,
					Version: version.Version,
				},
			},
		}

	case "notifications/initialized":
		// This is a notification (no ID), no response needed
		return nil

	case "tools/list":
		return &jsonRPCResponse{
			JSONRPC: jsonrpcVersion,
			ID:      req.ID,
			Result:  toolsListResult{Tools: getToolDefinitions()},
		}

	case "tools/call":
		return handleToolCall(req, client)

	case "ping":
		return &jsonRPCResponse{
			JSONRPC: jsonrpcVersion,
			ID:      req.ID,
			Result:  map[string]interface{}{},
		}

	default:
		log.Printf("Unknown method: %s", req.Method)
		return &jsonRPCResponse{
			JSONRPC: jsonrpcVersion,
			ID:      req.ID,
			Error:   &jsonRPCError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)},
		}
	}
}

func handleToolCall(req *jsonRPCRequest, client *Client) *jsonRPCResponse {
	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &jsonRPCResponse{
			JSONRPC: jsonrpcVersion,
			ID:      req.ID,
			Error:   &jsonRPCError{Code: -32602, Message: "Invalid params"},
		}
	}

	result, err := dispatchTool(params.Name, params.Arguments, client)
	if err != nil {
		return &jsonRPCResponse{
			JSONRPC: jsonrpcVersion,
			ID:      req.ID,
			Result: toolCallResult{
				Content: []contentBlock{{Type: "text", Text: err.Error()}},
				IsError: true,
			},
		}
	}

	return &jsonRPCResponse{
		JSONRPC: jsonrpcVersion,
		ID:      req.ID,
		Result:  result,
	}
}

func writeError(encoder *json.Encoder, id json.RawMessage, code int, message string) {
	encoder.Encode(&jsonRPCResponse{
		JSONRPC: jsonrpcVersion,
		ID:      id,
		Error:   &jsonRPCError{Code: code, Message: message},
	})
}
