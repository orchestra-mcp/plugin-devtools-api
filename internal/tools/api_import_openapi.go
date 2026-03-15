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

// ApiImportOpenapiSchema returns the JSON Schema for the api_import_openapi tool.
func ApiImportOpenapiSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"spec":            map[string]any{"type": "string", "description": "OpenAPI spec as JSON string"},
			"collection_name": map[string]any{"type": "string", "description": "Name for the imported collection"},
			"project_id":      map[string]any{"type": "string", "description": "Project slug (optional)"},
		},
		"required": []any{"spec"},
	})
	return s
}

// ApiImportOpenapi returns a handler that imports an OpenAPI spec into a new collection.
func ApiImportOpenapi(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "spec"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		specStr := helpers.GetString(req.Arguments, "spec")
		colName := helpers.GetString(req.Arguments, "collection_name")
		project := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		col, err := apiPkg.ParseSpec([]byte(specStr), colName)
		if err != nil {
			return helpers.ErrorResult("parse_error", err.Error()), nil
		}

		colID := newCollectionID()
		now := helpers.NowISO()
		col.ID = colID
		col.CreatedAt = now
		col.UpdatedAt = now

		endpointsJSON, _ := json.Marshal(col.Endpoints)
		metadata, _ := structpb.NewStruct(map[string]any{
			"id":          colID,
			"name":        col.Name,
			"description": col.Description,
			"base_url":    col.BaseURL,
			"count":       len(col.Endpoints),
			"created_at":  now,
			"updated_at":  now,
		})

		_, err = store.WriteCollection(ctx, project, colID, metadata, string(endpointsJSON), 0)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		md := fmt.Sprintf("## OpenAPI Imported\n\n- **Collection:** %s (`%s`)\n- **Base URL:** %s\n- **Endpoints imported:** %d",
			col.Name, colID, col.BaseURL, len(col.Endpoints))
		return helpers.TextResult(md), nil
	}
}
