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

// ToolHandler is the function signature for all tool handlers.
type ToolHandler = func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error)

// ApiRequestSchema returns the JSON Schema for the api_request tool.
func ApiRequestSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"method":    map[string]any{"type": "string", "description": "HTTP method (GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD)"},
			"url":       map[string]any{"type": "string", "description": "Request URL (supports {{variable}} interpolation)"},
			"headers":   map[string]any{"type": "object", "description": "Request headers as key-value pairs"},
			"body":      map[string]any{"type": "string", "description": "Request body content"},
			"body_type": map[string]any{"type": "string", "description": "Body content type: json, xml, form, text, graphql"},
			"auth": map[string]any{
				"type":        "object",
				"description": "Auth config: {type: bearer|basic|api_key, token, user, pass, key, value, in}",
			},
			"timeout":     map[string]any{"type": "number", "description": "Request timeout in seconds (default 30)"},
			"environment": map[string]any{"type": "string", "description": "Environment name for variable interpolation"},
			"project_id":  map[string]any{"type": "string", "description": "Project slug (optional)"},
		},
		"required": []any{"method", "url"},
	})
	return s
}

// ApiRequest returns a handler that executes an HTTP request and records history.
func ApiRequest(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "method", "url"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		apiReq := api.Request{
			Method:   helpers.GetString(req.Arguments, "method"),
			URL:      helpers.GetString(req.Arguments, "url"),
			Body:     helpers.GetString(req.Arguments, "body"),
			BodyType: helpers.GetString(req.Arguments, "body_type"),
			Timeout:  helpers.GetInt(req.Arguments, "timeout"),
			Headers:  helpers.GetStringMap(req.Arguments, "headers"),
		}

		// Parse auth config
		if req.Arguments != nil {
			if authField, ok := req.Arguments.Fields["auth"]; ok {
				if authStruct := authField.GetStructValue(); authStruct != nil {
					apiReq.Auth = &api.AuthConfig{
						Type:  helpers.GetString(authStruct, "type"),
						Token: helpers.GetString(authStruct, "token"),
						User:  helpers.GetString(authStruct, "user"),
						Pass:  helpers.GetString(authStruct, "pass"),
						Key:   helpers.GetString(authStruct, "key"),
						Value: helpers.GetString(authStruct, "value"),
						In:    helpers.GetString(authStruct, "in"),
					}
				}
			}
		}

		// Load environment if specified
		var env *api.Environment
		envName := helpers.GetString(req.Arguments, "environment")
		if envName != "" {
			project := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))
			meta, _, _, err := store.ReadEnvironment(ctx, project, envName)
			if err == nil && meta != nil {
				m := meta.AsMap()
				vars := make(map[string]string)
				if v, ok := m["variables"].(map[string]any); ok {
					for k, val := range v {
						vars[k] = fmt.Sprintf("%v", val)
					}
				}
				env = &api.Environment{Name: envName, Variables: vars}
			}
		}

		resp, err := api.ExecuteRequest(apiReq, env)
		if err != nil {
			return helpers.ErrorResult("request_error", err.Error()), nil
		}

		// Record to history
		project := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))
		date := time.Now().UTC().Format("2006-01-02")
		entry := api.HistoryEntry{
			Timestamp: helpers.NowISO(),
			Request:   apiReq,
			Response:  *resp,
		}
		entryJSON, _ := json.Marshal(entry)

		// Read existing history, append
		_, existingBody, ver, _ := store.ReadHistory(ctx, project, date)
		newBody := existingBody
		if newBody != "" {
			newBody += "\n---\n"
		}
		newBody += string(entryJSON)

		histMeta, _ := structpb.NewStruct(map[string]any{
			"date":       date,
			"updated_at": helpers.NowISO(),
		})
		store.AppendHistory(ctx, project, date, histMeta, newBody, ver)

		// Format response
		var b strings.Builder
		fmt.Fprintf(&b, "## %s %s\n\n", apiReq.Method, apiReq.URL)
		fmt.Fprintf(&b, "**Status:** %d %s\n", resp.StatusCode, resp.Status)
		fmt.Fprintf(&b, "**Duration:** %dms\n", resp.Duration)
		fmt.Fprintf(&b, "**Size:** %d bytes\n\n", resp.Size)

		if len(resp.Headers) > 0 {
			b.WriteString("### Response Headers\n")
			for k, v := range resp.Headers {
				fmt.Fprintf(&b, "- `%s`: %s\n", k, v)
			}
			b.WriteString("\n")
		}

		b.WriteString("### Response Body\n```\n")
		body := resp.Body
		if len(body) > 2000 {
			body = body[:2000] + "\n... (truncated)"
		}
		b.WriteString(body)
		b.WriteString("\n```\n")

		return helpers.TextResult(b.String()), nil
	}
}
