package internal

import (
	"github.com/orchestra-mcp/plugin-devtools-api/internal/api"
	"github.com/orchestra-mcp/plugin-devtools-api/internal/storage"
	"github.com/orchestra-mcp/plugin-devtools-api/internal/tools"
	"github.com/orchestra-mcp/sdk-go/plugin"
)

// ToolsPlugin holds shared state and registers all tools.
type ToolsPlugin struct {
	Storage *storage.DataStorage
}

// RegisterTools registers all 15 API tools with the plugin builder.
func (tp *ToolsPlugin) RegisterTools(builder *plugin.PluginBuilder) {
	s := tp.Storage
	wsMgr := api.NewWSManager()

	// Request execution (4)
	builder.RegisterTool("api_request",
		"Execute an HTTP request with optional auth, headers, body, and environment variable interpolation",
		tools.ApiRequestSchema(), tools.ApiRequest(s))

	builder.RegisterTool("api_ws_connect",
		"Open a WebSocket connection to a URL",
		tools.ApiWsConnectSchema(), tools.ApiWsConnect(wsMgr))

	builder.RegisterTool("api_ws_send",
		"Send a message over an active WebSocket connection",
		tools.ApiWsSendSchema(), tools.ApiWsSend(wsMgr))

	builder.RegisterTool("api_ws_close",
		"Close a WebSocket connection",
		tools.ApiWsCloseSchema(), tools.ApiWsClose(wsMgr))

	// Collections (5)
	builder.RegisterTool("api_save_request",
		"Save an API request to a collection (creates collection if needed)",
		tools.ApiSaveRequestSchema(), tools.ApiSaveRequest(s))

	builder.RegisterTool("api_list_collections",
		"List all API collections",
		tools.ApiListCollectionsSchema(), tools.ApiListCollections(s))

	builder.RegisterTool("api_get_collection",
		"Get a collection with all its endpoints",
		tools.ApiGetCollectionSchema(), tools.ApiGetCollection(s))

	builder.RegisterTool("api_delete_collection",
		"Delete an API collection",
		tools.ApiDeleteCollectionSchema(), tools.ApiDeleteCollection(s))

	builder.RegisterTool("api_search_endpoints",
		"Search across all collections for matching endpoints",
		tools.ApiSearchEndpointsSchema(), tools.ApiSearchEndpoints(s))

	// OpenAPI (3)
	builder.RegisterTool("api_generate_openapi",
		"Generate an OpenAPI 3.0 JSON spec from a collection",
		tools.ApiGenerateOpenapiSchema(), tools.ApiGenerateOpenapi(s))

	builder.RegisterTool("api_import_openapi",
		"Import an OpenAPI spec into a new collection",
		tools.ApiImportOpenapiSchema(), tools.ApiImportOpenapi(s))

	builder.RegisterTool("api_export_openapi",
		"Export a collection as a shareable OpenAPI spec",
		tools.ApiExportOpenapiSchema(), tools.ApiExportOpenapi(s))

	// Environment (2)
	builder.RegisterTool("api_set_env",
		"Set environment variables for API request interpolation",
		tools.ApiSetEnvSchema(), tools.ApiSetEnv(s))

	builder.RegisterTool("api_get_env",
		"Get environment variables (secrets are masked)",
		tools.ApiGetEnvSchema(), tools.ApiGetEnv(s))

	// History (1)
	builder.RegisterTool("api_history",
		"View recent API request history",
		tools.ApiHistorySchema(), tools.ApiHistory(s))
}
