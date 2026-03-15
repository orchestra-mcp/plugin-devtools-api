package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"github.com/orchestra-mcp/plugin-devtools-api/internal/api"
	"github.com/orchestra-mcp/plugin-devtools-api/internal/storage"
	"google.golang.org/protobuf/types/known/structpb"
)

// ApiHistorySchema returns the JSON Schema for the api_history tool.
func ApiHistorySchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"limit":      map[string]any{"type": "number", "description": "Max entries to return (default 20)"},
			"date":       map[string]any{"type": "string", "description": "Filter by date (YYYY-MM-DD, defaults to today)"},
			"project_id": map[string]any{"type": "string", "description": "Project slug (optional)"},
		},
	})
	return s
}

// ApiHistory returns a handler that shows recent request history.
func ApiHistory(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		project := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))
		limit := helpers.GetInt(req.Arguments, "limit")
		if limit <= 0 {
			limit = 20
		}
		date := helpers.GetString(req.Arguments, "date")
		if date == "" {
			date = time.Now().UTC().Format("2006-01-02")
		}

		_, body, _, err := store.ReadHistory(ctx, project, date)
		if err != nil || body == "" {
			return helpers.TextResult(fmt.Sprintf("## API History (%s)\n\nNo requests recorded for this date.", date)), nil
		}

		// Parse entries (separated by ---)
		parts := strings.Split(body, "\n---\n")
		var entries []api.HistoryEntry
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			var entry api.HistoryEntry
			if err := json.Unmarshal([]byte(part), &entry); err != nil {
				continue
			}
			entries = append(entries, entry)
		}

		// Reverse to show newest first
		for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
			entries[i], entries[j] = entries[j], entries[i]
		}

		// Apply limit
		if len(entries) > limit {
			entries = entries[:limit]
		}

		var b strings.Builder
		fmt.Fprintf(&b, "## API History (%s) — %d entries\n\n", date, len(entries))
		b.WriteString("| Time | Method | URL | Status | Duration |\n")
		b.WriteString("|------|--------|-----|--------|----------|\n")
		for _, e := range entries {
			ts := e.Timestamp
			if len(ts) > 19 {
				ts = ts[11:19] // HH:MM:SS
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %d | %dms |\n",
				ts, e.Request.Method, truncate(e.Request.URL, 50), e.Response.StatusCode, e.Response.Duration)
		}

		return helpers.TextResult(b.String()), nil
	}
}
