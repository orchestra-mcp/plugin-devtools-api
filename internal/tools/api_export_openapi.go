package tools

import (
	"context"
	"encoding/json"
	"fmt"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	apiPkg "github.com/orchestra-mcp/plugin-devtools-api/internal/api"
	"github.com/orchestra-mcp/plugin-devtools-api/internal/storage"
	"google.golang.org/protobuf/types/known/structpb"
)

// ApiExportOpenapiSchema returns the JSON Schema for the api_export_openapi tool.
func ApiExportOpenapiSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"collection_id": map[string]any{"type": "string", "description": "Collection ID to export"},
			"title":         map[string]any{"type": "string", "description": "API title (defaults to collection name)"},
			"version":       map[string]any{"type": "string", "description": "API version (defaults to 1.0.0)"},
			"project_id":    map[string]any{"type": "string", "description": "Project slug (optional)"},
		},
		"required": []any{"collection_id"},
	})
	return s
}

// ApiExportOpenapi returns a handler that exports a collection as an OpenAPI spec
// and saves it to storage for sharing.
func ApiExportOpenapi(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "collection_id"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		project := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))
		colID := helpers.GetString(req.Arguments, "collection_id")
		title := helpers.GetString(req.Arguments, "title")
		version := helpers.GetString(req.Arguments, "version")

		meta, body, _, err := store.ReadCollection(ctx, project, colID)
		if err != nil {
			return helpers.ErrorResult("not_found", fmt.Sprintf("collection %s not found: %v", colID, err)), nil
		}

		m := meta.AsMap()
		col := apiPkg.Collection{
			ID:   colID,
			Name: m["name"].(string),
		}
		if desc, ok := m["description"].(string); ok {
			col.Description = desc
		}
		if base, ok := m["base_url"].(string); ok {
			col.BaseURL = base
		}
		json.Unmarshal([]byte(body), &col.Endpoints)

		specJSON, err := apiPkg.GenerateSpec(col, title, version)
		if err != nil {
			return helpers.ErrorResult("generation_error", err.Error()), nil
		}

		// Save spec to storage for sharing
		specMeta, _ := structpb.NewStruct(map[string]any{
			"collection_id": colID,
			"title":         col.Name,
			"version":       version,
			"exported_at":   helpers.NowISO(),
		})
		specPath := fmt.Sprintf("api/specs/%s-openapi.md", colID)
		_ = specPath
		// Store as a shareable file
		store.WriteCollection(ctx, project, colID+"-openapi", specMeta, string(specJSON), 0)

		md := fmt.Sprintf("## OpenAPI Spec Exported\n\n- **Collection:** %s (`%s`)\n- **Endpoints:** %d\n- **Stored as:** `%s-openapi`\n\nThe spec can be shared with team members or imported into other API tools.",
			col.Name, colID, len(col.Endpoints), colID)
		return helpers.TextResult(md), nil
	}
}
