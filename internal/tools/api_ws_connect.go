package tools

import (
	"context"
	"fmt"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"github.com/orchestra-mcp/plugin-devtools-api/internal/api"
	"google.golang.org/protobuf/types/known/structpb"
)

// ApiWsConnectSchema returns the JSON Schema for the api_ws_connect tool.
func ApiWsConnectSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url":       map[string]any{"type": "string", "description": "WebSocket URL (ws:// or wss://)"},
			"headers":   map[string]any{"type": "object", "description": "Connection headers as key-value pairs"},
			"protocols": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Subprotocols to negotiate"},
		},
		"required": []any{"url"},
	})
	return s
}

// ApiWsConnect returns a handler that opens a WebSocket connection.
func ApiWsConnect(wsMgr *api.WSManager) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "url"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		wsURL := helpers.GetString(req.Arguments, "url")
		headers := helpers.GetStringMap(req.Arguments, "headers")

		// Add subprotocol header if provided
		protocols := helpers.GetStringSlice(req.Arguments, "protocols")
		if len(protocols) > 0 {
			if headers == nil {
				headers = make(map[string]string)
			}
			for _, p := range protocols {
				headers["Sec-WebSocket-Protocol"] = p
			}
		}

		connID, err := wsMgr.Connect(wsURL, headers)
		if err != nil {
			return helpers.ErrorResult("ws_error", err.Error()), nil
		}

		md := fmt.Sprintf("## WebSocket Connected\n\n- **Connection ID:** `%s`\n- **URL:** %s\n\nUse `api_ws_send` to send messages and `api_ws_close` to disconnect.", connID, wsURL)
		return helpers.TextResult(md), nil
	}
}
