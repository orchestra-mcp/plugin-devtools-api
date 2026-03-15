package api

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// GenerateSpec creates an OpenAPI 3.0 JSON spec from a collection.
func GenerateSpec(col Collection, title, version string) ([]byte, error) {
	if title == "" {
		title = col.Name
	}
	if version == "" {
		version = "1.0.0"
	}

	paths := make(map[string]map[string]any)

	for _, ep := range col.Endpoints {
		path := extractPath(ep.URL, col.BaseURL)
		method := strings.ToLower(ep.Method)
		if method == "" {
			method = "get"
		}

		operation := map[string]any{
			"summary":     ep.Name,
			"operationId": toOperationID(method, path),
		}
		if ep.Description != "" {
			operation["description"] = ep.Description
		}

		// Path parameters
		params := extractPathParams(path)
		if len(params) > 0 {
			paramList := make([]map[string]any, 0, len(params))
			for _, p := range params {
				paramList = append(paramList, map[string]any{
					"name":     p,
					"in":       "path",
					"required": true,
					"schema":   map[string]any{"type": "string"},
				})
			}
			operation["parameters"] = paramList
		}

		// Request body
		if ep.Body != "" && (method == "post" || method == "put" || method == "patch") {
			ct := "application/json"
			switch ep.BodyType {
			case "xml":
				ct = "text/xml"
			case "form":
				ct = "application/x-www-form-urlencoded"
			case "text":
				ct = "text/plain"
			}
			operation["requestBody"] = map[string]any{
				"content": map[string]any{
					ct: map[string]any{
						"schema": map[string]any{"type": "object"},
					},
				},
			}
		}

		operation["responses"] = map[string]any{
			"200": map[string]any{"description": "Successful response"},
		}

		if _, ok := paths[path]; !ok {
			paths[path] = make(map[string]any)
		}
		paths[path][method] = operation
	}

	spec := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   title,
			"version": version,
		},
		"paths": paths,
	}

	if col.BaseURL != "" {
		spec["servers"] = []map[string]any{
			{"url": col.BaseURL},
		}
	}

	return json.MarshalIndent(spec, "", "  ")
}

// ParseSpec imports an OpenAPI 3.0 JSON spec into a Collection.
func ParseSpec(data []byte, collectionName string) (*Collection, error) {
	var spec map[string]any
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parse openapi spec: %w", err)
	}

	col := &Collection{
		Name: collectionName,
	}

	// Extract server base URL
	if servers, ok := spec["servers"].([]any); ok && len(servers) > 0 {
		if srv, ok := servers[0].(map[string]any); ok {
			if u, ok := srv["url"].(string); ok {
				col.BaseURL = u
			}
		}
	}

	// Extract info
	if info, ok := spec["info"].(map[string]any); ok {
		if col.Name == "" {
			if t, ok := info["title"].(string); ok {
				col.Name = t
			}
		}
		if desc, ok := info["description"].(string); ok {
			col.Description = desc
		}
	}

	// Extract paths
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		return col, nil
	}

	for path, methods := range paths {
		methodMap, ok := methods.(map[string]any)
		if !ok {
			continue
		}
		for method, opData := range methodMap {
			op, ok := opData.(map[string]any)
			if !ok {
				continue
			}
			endpoint := Request{
				Method: strings.ToUpper(method),
				URL:    path,
			}
			if summary, ok := op["summary"].(string); ok {
				endpoint.Name = summary
			} else {
				endpoint.Name = fmt.Sprintf("%s %s", strings.ToUpper(method), path)
			}
			if desc, ok := op["description"].(string); ok {
				endpoint.Description = desc
			}
			col.Endpoints = append(col.Endpoints, endpoint)
		}
	}

	return col, nil
}

var pathParamPattern = regexp.MustCompile(`\{(\w+)\}`)

func extractPathParams(path string) []string {
	matches := pathParamPattern.FindAllStringSubmatch(path, -1)
	params := make([]string, 0, len(matches))
	for _, m := range matches {
		params = append(params, m[1])
	}
	return params
}

func extractPath(rawURL, baseURL string) string {
	path := rawURL
	if baseURL != "" {
		path = strings.TrimPrefix(path, baseURL)
	}
	// Remove scheme+host if present
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		parts := strings.SplitN(path, "/", 4)
		if len(parts) >= 4 {
			path = "/" + parts[3]
		} else {
			path = "/"
		}
	}
	// Remove query string
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func toOperationID(method, path string) string {
	// Convert /users/{id}/posts to users_id_posts
	clean := strings.ReplaceAll(path, "/", "_")
	clean = strings.ReplaceAll(clean, "{", "")
	clean = strings.ReplaceAll(clean, "}", "")
	clean = strings.Trim(clean, "_")
	return method + "_" + clean
}
