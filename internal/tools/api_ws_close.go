package tools

import (
	"context"
	"fmt"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"github.com/orchestra-mcp/plugin-devtools-api/internal/api"
	"google.golang.org/protobuf/types/known/structpb"
)

// ApiWsCloseSchema returns the JSON Schema for the api_ws_close tool.
func ApiWsCloseSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"connection_id": map[string]any{"type": "string", "description": "WebSocket connection ID to close"},
		},
		"required": []any{"connection_id"},
	})
	return s
}

// ApiWsClose returns a handler that closes a WebSocket connection.
func ApiWsClose(wsMgr *api.WSManager) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "connection_id"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		connID := helpers.GetString(req.Arguments, "connection_id")
		count, err := wsMgr.Close(connID)
		if err != nil {
			return helpers.ErrorResult("ws_error", err.Error()), nil
		}

		md := fmt.Sprintf("## WebSocket Closed\n\n- **Connection:** `%s`\n- **Messages exchanged:** %d", connID, count)
		return helpers.TextResult(md), nil
	}
}
