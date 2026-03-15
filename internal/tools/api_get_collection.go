package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"github.com/orchestra-mcp/plugin-devtools-api/internal/api"
	"github.com/orchestra-mcp/plugin-devtools-api/internal/storage"
	"google.golang.org/protobuf/types/known/structpb"
)

// ApiGetCollectionSchema returns the JSON Schema for the api_get_collection tool.
func ApiGetCollectionSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"collection_id": map[string]any{"type": "string", "description": "Collection ID"},
			"project_id":    map[string]any{"type": "string", "description": "Project slug (optional)"},
		},
		"required": []any{"collection_id"},
	})
	return s
}

// ApiGetCollection returns a handler that retrieves a collection with all endpoints.
func ApiGetCollection(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "collection_id"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		project := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))
		colID := helpers.GetString(req.Arguments, "collection_id")

		meta, body, _, err := store.ReadCollection(ctx, project, colID)
		if err != nil {
			return helpers.ErrorResult("not_found", fmt.Sprintf("collection %s not found: %v", colID, err)), nil
		}

		m := meta.AsMap()
		name, _ := m["name"].(string)
		desc, _ := m["description"].(string)
		baseURL, _ := m["base_url"].(string)

		var endpoints []api.Request
		json.Unmarshal([]byte(body), &endpoints)

		var b strings.Builder
		fmt.Fprintf(&b, "## %s (%s)\n\n", name, colID)
		if desc != "" {
			fmt.Fprintf(&b, "%s\n\n", desc)
		}
		if baseURL != "" {
			fmt.Fprintf(&b, "**Base URL:** %s\n\n", baseURL)
		}
		fmt.Fprintf(&b, "### Endpoints (%d)\n\n", len(endpoints))

		if len(endpoints) == 0 {
			b.WriteString("No endpoints saved yet.\n")
		} else {
			b.WriteString("| Method | URL | Name | Folder |\n")
			b.WriteString("|--------|-----|------|--------|\n")
			for _, ep := range endpoints {
				folder := ep.Folder
				if folder == "" {
					folder = "—"
				}
				fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", ep.Method, ep.URL, ep.Name, folder)
			}
		}

		return helpers.TextResult(b.String()), nil
	}
}
