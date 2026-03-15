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

// ApiSearchEndpointsSchema returns the JSON Schema for the api_search_endpoints tool.
func ApiSearchEndpointsSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query":      map[string]any{"type": "string", "description": "Search query (matches name, URL, description)"},
			"method":     map[string]any{"type": "string", "description": "Filter by HTTP method (optional)"},
			"project_id": map[string]any{"type": "string", "description": "Project slug (optional)"},
		},
		"required": []any{"query"},
	})
	return s
}

// ApiSearchEndpoints returns a handler that searches across all collections.
func ApiSearchEndpoints(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "query"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		project := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))
		query := strings.ToLower(helpers.GetString(req.Arguments, "query"))
		methodFilter := strings.ToUpper(helpers.GetString(req.Arguments, "method"))

		entries, err := store.ListCollections(ctx, project)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		type match struct {
			colName, colID string
			endpoint       api.Request
		}
		var matches []match

		for _, entry := range entries {
			colID := storage.IDFromPath(entry.Path)
			meta, body, _, err := store.ReadCollection(ctx, project, colID)
			if err != nil {
				continue
			}
			m := meta.AsMap()
			colName, _ := m["name"].(string)

			var endpoints []api.Request
			json.Unmarshal([]byte(body), &endpoints)

			for _, ep := range endpoints {
				if methodFilter != "" && strings.ToUpper(ep.Method) != methodFilter {
					continue
				}
				nameLower := strings.ToLower(ep.Name)
				urlLower := strings.ToLower(ep.URL)
				descLower := strings.ToLower(ep.Description)
				if strings.Contains(nameLower, query) || strings.Contains(urlLower, query) || strings.Contains(descLower, query) {
					matches = append(matches, match{colName: colName, colID: colID, endpoint: ep})
				}
			}
		}

		if len(matches) == 0 {
			return helpers.TextResult(fmt.Sprintf("## Search Results\n\nNo endpoints matching \"%s\" found.", query)), nil
		}

		var b strings.Builder
		fmt.Fprintf(&b, "## Search Results — %d endpoints matching \"%s\"\n\n", len(matches), query)
		b.WriteString("| Collection | Method | URL | Name |\n")
		b.WriteString("|------------|--------|-----|------|\n")
		for _, m := range matches {
			fmt.Fprintf(&b, "| %s (%s) | %s | %s | %s |\n", m.colName, m.colID, m.endpoint.Method, m.endpoint.URL, m.endpoint.Name)
		}

		return helpers.TextResult(b.String()), nil
	}
}
