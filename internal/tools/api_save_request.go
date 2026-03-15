package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"github.com/orchestra-mcp/plugin-devtools-api/internal/api"
	"github.com/orchestra-mcp/plugin-devtools-api/internal/storage"
	"google.golang.org/protobuf/types/known/structpb"
)

func newCollectionID() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return "COL-" + hex.EncodeToString(b)
}

// ApiSaveRequestSchema returns the JSON Schema for the api_save_request tool.
func ApiSaveRequestSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"collection_id":   map[string]any{"type": "string", "description": "Collection ID to add to (creates new if omitted)"},
			"collection_name": map[string]any{"type": "string", "description": "Name for new collection (used when collection_id is omitted)"},
			"name":            map[string]any{"type": "string", "description": "Endpoint name"},
			"method":          map[string]any{"type": "string", "description": "HTTP method"},
			"url":             map[string]any{"type": "string", "description": "Request URL"},
			"headers":         map[string]any{"type": "object", "description": "Default headers"},
			"body":            map[string]any{"type": "string", "description": "Default body"},
			"body_type":       map[string]any{"type": "string", "description": "Body type: json, xml, form, text, graphql"},
			"description":     map[string]any{"type": "string", "description": "Endpoint description"},
			"folder":          map[string]any{"type": "string", "description": "Folder within collection"},
			"project_id":      map[string]any{"type": "string", "description": "Project slug (optional)"},
		},
		"required": []any{"name", "method", "url"},
	})
	return s
}

// ApiSaveRequest returns a handler that saves a request to a collection.
func ApiSaveRequest(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "name", "method", "url"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		project := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))
		colID := helpers.GetString(req.Arguments, "collection_id")

		endpoint := api.Request{
			Name:        helpers.GetString(req.Arguments, "name"),
			Method:      helpers.GetString(req.Arguments, "method"),
			URL:         helpers.GetString(req.Arguments, "url"),
			Headers:     helpers.GetStringMap(req.Arguments, "headers"),
			Body:        helpers.GetString(req.Arguments, "body"),
			BodyType:    helpers.GetString(req.Arguments, "body_type"),
			Description: helpers.GetString(req.Arguments, "description"),
			Folder:      helpers.GetString(req.Arguments, "folder"),
		}

		now := helpers.NowISO()
		var col api.Collection
		var ver int64

		if colID != "" {
			// Load existing collection
			meta, body, v, err := store.ReadCollection(ctx, project, colID)
			if err != nil {
				return helpers.ErrorResult("storage_error", fmt.Sprintf("collection %s not found: %v", colID, err)), nil
			}
			ver = v
			m := meta.AsMap()
			col.ID, _ = m["id"].(string)
			col.Name, _ = m["name"].(string)
			col.Description, _ = m["description"].(string)
			col.BaseURL, _ = m["base_url"].(string)
			col.CreatedAt, _ = m["created_at"].(string)

			// Parse existing endpoints from body
			if body != "" {
				json.Unmarshal([]byte(body), &col.Endpoints)
			}
		} else {
			colID = newCollectionID()
			colName := helpers.GetString(req.Arguments, "collection_name")
			if colName == "" {
				colName = "Untitled Collection"
			}
			col = api.Collection{
				ID:        colID,
				Name:      colName,
				CreatedAt: now,
			}
		}

		endpoint.ID = fmt.Sprintf("ep-%s", helpers.NewUUID()[:8])
		col.Endpoints = append(col.Endpoints, endpoint)
		col.UpdatedAt = now

		endpointsJSON, _ := json.Marshal(col.Endpoints)
		metadata, _ := structpb.NewStruct(map[string]any{
			"id":          col.ID,
			"name":        col.Name,
			"description": col.Description,
			"base_url":    col.BaseURL,
			"count":       len(col.Endpoints),
			"created_at":  col.CreatedAt,
			"updated_at":  now,
		})

		_, err := store.WriteCollection(ctx, project, colID, metadata, string(endpointsJSON), ver)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		md := fmt.Sprintf("## Request Saved\n\n- **Endpoint:** %s `%s %s`\n- **Collection:** %s (%s)\n- **Endpoints in collection:** %d",
			endpoint.Name, endpoint.Method, endpoint.URL, col.Name, colID, len(col.Endpoints))
		return helpers.TextResult(md), nil
	}
}
