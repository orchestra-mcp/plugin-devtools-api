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

// ApiGenerateOpenapiSchema returns the JSON Schema for the api_generate_openapi tool.
func ApiGenerateOpenapiSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"collection_id": map[string]any{"type": "string", "description": "Collection ID to generate spec from"},
			"title":         map[string]any{"type": "string", "description": "API title (defaults to collection name)"},
			"version":       map[string]any{"type": "string", "description": "API version (defaults to 1.0.0)"},
			"project_id":    map[string]any{"type": "string", "description": "Project slug (optional)"},
		},
		"required": []any{"collection_id"},
	})
	return s
}

// ApiGenerateOpenapi returns a handler that generates an OpenAPI 3.0 spec from a collection.
func ApiGenerateOpenapi(store *storage.DataStorage) ToolHandler {
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

		md := fmt.Sprintf("## OpenAPI 3.0 Spec Generated\n\n**Collection:** %s (%s)\n**Endpoints:** %d\n\n```json\n%s\n```",
			col.Name, colID, len(col.Endpoints), string(specJSON))
		return helpers.TextResult(md), nil
	}
}
