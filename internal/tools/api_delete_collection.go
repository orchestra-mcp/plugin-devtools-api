package tools

import (
	"context"
	"fmt"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"github.com/orchestra-mcp/plugin-devtools-api/internal/storage"
	"google.golang.org/protobuf/types/known/structpb"
)

// ApiDeleteCollectionSchema returns the JSON Schema for the api_delete_collection tool.
func ApiDeleteCollectionSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"collection_id": map[string]any{"type": "string", "description": "Collection ID to delete"},
			"project_id":    map[string]any{"type": "string", "description": "Project slug (optional)"},
		},
		"required": []any{"collection_id"},
	})
	return s
}

// ApiDeleteCollection returns a handler that deletes a collection.
func ApiDeleteCollection(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "collection_id"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		project := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))
		colID := helpers.GetString(req.Arguments, "collection_id")

		if err := store.DeleteCollection(ctx, project, colID); err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		return helpers.TextResult(fmt.Sprintf("Deleted collection `%s`.", colID)), nil
	}
}
