// Package tools registers MCP tools backed by the HubSpot client wrapper.
package tools

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// JSONResult wraps an already-serialized JSON payload as a successful tool result.
func JSONResult(payload []byte) *mcp.CallToolResult {
	return mcp.NewToolResultText(string(payload))
}

// APIError converts a HubSpot SDK error into a tool error result with the
// canonical "HubSpot API error: " prefix.
func APIError(err error) *mcp.CallToolResult {
	return mcp.NewToolResultError("HubSpot API error: " + err.Error())
}

// RequiredStringArg extracts a required string argument from the request.
func RequiredStringArg(req mcp.CallToolRequest, key string) (string, error) {
	return req.RequireString(key)
}

// OptionalStringArg returns a string argument, falling back to def when absent.
func OptionalStringArg(req mcp.CallToolRequest, key string, def string) string {
	return req.GetString(key, def)
}

// OptionalIntArg returns an int argument, falling back to def when absent.
func OptionalIntArg(req mcp.CallToolRequest, key string, def int) int {
	return req.GetInt(key, def)
}

// OptionalStringArrayArg returns a string slice argument, or nil when the
// argument is absent or not a string array.
func OptionalStringArrayArg(req mcp.CallToolRequest, key string) []string {
	args := req.GetArguments()
	if args == nil {
		return nil
	}
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		out := make([]string, len(v))
		copy(out, v)
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil
			}
			out = append(out, s)
		}
		return out
	default:
		return nil
	}
}

// OptionalObjectArg returns the value of a JSON-object argument:
//   - (nil, nil) when absent
//   - (map, nil) when present and an object
//   - (nil, err) when present but not an object
func OptionalObjectArg(req mcp.CallToolRequest, key string) (map[string]any, error) {
	args := req.GetArguments()
	if args == nil {
		return nil, nil
	}
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil, nil
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("argument %q must be a JSON object", key)
	}
	return obj, nil
}
