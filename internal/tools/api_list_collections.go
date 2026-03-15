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

// ApiListCollectionsSchema returns the JSON Schema for the api_list_collections tool.
func ApiListCollectionsSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug (optional, defaults to global)"},
		},
	})
	return s
}

// ApiListCollections returns a handler that lists all API collections.
func ApiListCollections(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		project := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		entries, err := store.ListCollections(ctx, project)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		if len(entries) == 0 {
			return helpers.TextResult("## API Collections\n\nNo collections found. Use `api_save_request` to create one."), nil
		}

		type colInfo struct {
			id, name, baseURL string
			count             int
		}
		var cols []colInfo

		for _, entry := range entries {
			colID := storage.IDFromPath(entry.Path)
			meta, body, _, err := store.ReadCollection(ctx, project, colID)
			if err != nil {
				continue
			}
			m := meta.AsMap()
			name, _ := m["name"].(string)
			baseURL, _ := m["base_url"].(string)

			var endpoints []api.Request
			json.Unmarshal([]byte(body), &endpoints)

			cols = append(cols, colInfo{
				id:      colID,
				name:    name,
				baseURL: baseURL,
				count:   len(endpoints),
			})
		}

		var b strings.Builder
		fmt.Fprintf(&b, "## API Collections — %d found\n\n", len(cols))
		b.WriteString("| ID | Name | Base URL | Endpoints |\n")
		b.WriteString("|----|------|----------|-----------|\n")
		for _, c := range cols {
			base := c.baseURL
			if base == "" {
				base = "—"
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %d |\n", c.id, c.name, base, c.count)
		}

		return helpers.TextResult(b.String()), nil
	}
}
