package tools

import (
	"context"
	"fmt"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"github.com/orchestra-mcp/plugin-devtools-api/internal/storage"
	"google.golang.org/protobuf/types/known/structpb"
)

// ApiSetEnvSchema returns the JSON Schema for the api_set_env tool.
func ApiSetEnvSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":       map[string]any{"type": "string", "description": "Environment name (e.g., 'production', 'staging', 'local')"},
			"variables":  map[string]any{"type": "object", "description": "Key-value pairs of environment variables"},
			"project_id": map[string]any{"type": "string", "description": "Project slug (optional)"},
		},
		"required": []any{"name", "variables"},
	})
	return s
}

// ApiSetEnv returns a handler that creates or updates an environment.
func ApiSetEnv(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "name", "variables"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		name := helpers.GetString(req.Arguments, "name")
		variables := helpers.GetStringMap(req.Arguments, "variables")
		project := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))
		now := helpers.NowISO()

		// Read existing to get version
		_, _, ver, _ := store.ReadEnvironment(ctx, project, name)

		// Build variables as any map for structpb
		varsAny := make(map[string]any, len(variables))
		for k, v := range variables {
			varsAny[k] = v
		}

		metadata, _ := structpb.NewStruct(map[string]any{
			"name":       name,
			"variables":  varsAny,
			"count":      len(variables),
			"updated_at": now,
		})

		// Build a readable body
		var body strings.Builder
		for k, v := range variables {
			fmt.Fprintf(&body, "%s=%s\n", k, v)
		}

		_, err := store.WriteEnvironment(ctx, project, name, metadata, body.String(), ver)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		md := fmt.Sprintf("## Environment Updated\n\n- **Name:** %s\n- **Variables:** %d\n\n", name, len(variables))
		for k := range variables {
			md += fmt.Sprintf("- `%s`\n", k)
		}

		return helpers.TextResult(md), nil
	}
}
