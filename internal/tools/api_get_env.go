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

// ApiGetEnvSchema returns the JSON Schema for the api_get_env tool.
func ApiGetEnvSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":       map[string]any{"type": "string", "description": "Environment name"},
			"project_id": map[string]any{"type": "string", "description": "Project slug (optional)"},
		},
		"required": []any{"name"},
	})
	return s
}

// secretKeys are variable names that should be masked in output.
var secretKeys = map[string]bool{
	"password": true, "secret": true, "token": true, "api_key": true,
	"apikey": true, "private_key": true, "access_token": true,
}

// ApiGetEnv returns a handler that retrieves an environment (masking secrets).
func ApiGetEnv(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "name"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		name := helpers.GetString(req.Arguments, "name")
		project := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		meta, _, _, err := store.ReadEnvironment(ctx, project, name)
		if err != nil {
			return helpers.ErrorResult("not_found", fmt.Sprintf("environment %s not found: %v", name, err)), nil
		}

		m := meta.AsMap()
		vars, _ := m["variables"].(map[string]any)

		var b strings.Builder
		fmt.Fprintf(&b, "## Environment: %s\n\n", name)
		fmt.Fprintf(&b, "**Variables:** %d\n\n", len(vars))

		b.WriteString("| Key | Value |\n")
		b.WriteString("|-----|-------|\n")
		for k, v := range vars {
			val := fmt.Sprintf("%v", v)
			// Mask secret values
			lowerKey := strings.ToLower(k)
			for secretKey := range secretKeys {
				if strings.Contains(lowerKey, secretKey) {
					if len(val) > 4 {
						val = val[:2] + strings.Repeat("*", len(val)-4) + val[len(val)-2:]
					} else {
						val = "****"
					}
					break
				}
			}
			fmt.Fprintf(&b, "| `%s` | `%s` |\n", k, val)
		}

		return helpers.TextResult(b.String()), nil
	}
}
