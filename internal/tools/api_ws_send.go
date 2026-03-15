package tools

import (
	"context"
	"fmt"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"github.com/orchestra-mcp/plugin-devtools-api/internal/api"
	"google.golang.org/protobuf/types/known/structpb"
)

// ApiWsSendSchema returns the JSON Schema for the api_ws_send tool.
func ApiWsSendSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"connection_id": map[string]any{"type": "string", "description": "WebSocket connection ID from api_ws_connect"},
			"message":       map[string]any{"type": "string", "description": "Message to send"},
		},
		"required": []any{"connection_id", "message"},
	})
	return s
}

// ApiWsSend returns a handler that sends a message over a WebSocket connection.
func ApiWsSend(wsMgr *api.WSManager) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "connection_id", "message"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		connID := helpers.GetString(req.Arguments, "connection_id")
		message := helpers.GetString(req.Arguments, "message")

		if err := wsMgr.Send(connID, message); err != nil {
			return helpers.ErrorResult("ws_error", err.Error()), nil
		}

		// Get latest messages
		conn, err := wsMgr.Get(connID)
		if err != nil {
			return helpers.TextResult(fmt.Sprintf("Message sent to `%s`.", connID)), nil
		}

		md := fmt.Sprintf("## Message Sent\n\n- **Connection:** `%s`\n- **Sent:** %s\n\n### Recent Messages (%d total)\n", connID, truncate(message, 100), len(conn.Messages))
		last := conn.Messages
		if len(last) > 5 {
			last = last[len(last)-5:]
		}
		for _, m := range last {
			dir := "→"
			if m.Direction == "received" {
				dir = "←"
			}
			md += fmt.Sprintf("- %s `%s` %s\n", dir, truncate(m.Data, 80), m.Timestamp)
		}

		return helpers.TextResult(md), nil
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
