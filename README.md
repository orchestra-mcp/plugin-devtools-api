# Orchestra Plugin: devtools-api

API Explorer — HTTP/WebSocket testing, collections, and OpenAPI generation for [Orchestra MCP](https://github.com/orchestra-mcp/framework).

A Postman-like API testing tool built over MCP. Execute HTTP requests, manage WebSocket connections, organize endpoints into collections, generate OpenAPI 3.0 specs, and interpolate environment variables — all from your IDE or dashboard.

## Install

```bash
go install github.com/orchestra-mcp/plugin-devtools-api/cmd@latest
```

## Usage

The plugin is bundled in-process with the Orchestra CLI. All 15 tools are available automatically when running `orchestra serve`.

For standalone usage, add to your `plugins.yaml`:

```yaml
- id: devtools.api
  binary: ./bin/devtools-api
  enabled: true
```

## Tools

### Request Execution

| Tool | Description |
|------|-------------|
| `api_request` | Execute HTTP request with auth, headers, body, and `{{variable}}` interpolation |
| `api_ws_connect` | Open a WebSocket connection |
| `api_ws_send` | Send a message over an active WebSocket connection |
| `api_ws_close` | Close a WebSocket connection |

### Collections

| Tool | Description |
|------|-------------|
| `api_save_request` | Save a request to a collection (creates collection if needed) |
| `api_list_collections` | List all API collections |
| `api_get_collection` | Get a collection with all its endpoints |
| `api_delete_collection` | Delete a collection |
| `api_search_endpoints` | Search across all collections for matching endpoints |

### OpenAPI

| Tool | Description |
|------|-------------|
| `api_generate_openapi` | Generate an OpenAPI 3.0 JSON spec from a collection |
| `api_import_openapi` | Import an OpenAPI spec into a new collection |
| `api_export_openapi` | Export a collection as a shareable OpenAPI spec |

### Environment

| Tool | Description |
|------|-------------|
| `api_set_env` | Set environment variables for request interpolation |
| `api_get_env` | Get environment variables (secrets are masked) |

### History

| Tool | Description |
|------|-------------|
| `api_history` | View recent API request history |

## Features

- **Multi-protocol**: HTTP (all methods) and WebSocket support
- **Variable interpolation**: Use `{{variable}}` syntax in URLs, headers, and body — resolved from named environments
- **Authentication**: Bearer token, Basic auth, and API key (header or query param)
- **Collections**: Group related endpoints, organize by folder
- **OpenAPI 3.0**: Generate specs from collections, import specs into collections
- **History**: Automatic request/response logging by date
- **Secret masking**: Environment variables with sensitive names are masked in output

## Frontend

The Dashboard includes a dedicated **API Explorer** UI at DevTools > API Explorer with:

- **Left panel**: Collections sidebar with OpenAPI generation
- **Center panel**: Request builder (method, URL, headers, body, auth) + response viewer
- **Right panel**: Environment variables, request history, OpenAPI preview

## Related Packages

- [sdk-go](https://github.com/orchestra-mcp/sdk-go) — Plugin SDK
- [gen-go](https://github.com/orchestra-mcp/gen-go) — Generated Protobuf types
