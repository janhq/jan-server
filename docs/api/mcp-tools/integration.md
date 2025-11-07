# MCP Tools Service - Integration Notes

## Current State

The MCP Tools service has been scaffolded with Clean Architecture following platform patterns. However, the `mcp-go` library version 0.7.0 has a different API than expected.

## API Changes in mcp-go v0.7.0

The current version of `github.com/mark3labs/mcp-go` (v0.7.0) has the following differences from platform's reference:

1. **No `WithToolCapabilities` option** - Tools are registered directly, no capability declaration needed
2. **No `WithRecovery` option** - Recovery middleware doesn't exist
3. **No `NewStreamableHTTPServer` function** - Use `NewStreamableHTTPServer(server, ...options)` instead
4. **SSEServer doesn't implement http.Handler** - Has `.Start(addr)` method instead
5. **CallToolRequest API changed** - No `RequireString`, `GetString` helper methods

## Recommended Approach

Based on mcp-go examples, there are two approaches:

### Option 1: Standalone MCP Server (Recommended)

Don't try to integrate MCP into Gin. Run a separate MCP server:

```go
func main() {
    // Create MCP server
    mcpServer := server.NewMCPServer("mcp-tools", "1.0.0")
    
    // Register tools
    serperMCP.RegisterTools(mcpServer)
    
    // Start HTTP server
    httpServer := server.NewStreamableHTTPServer(mcpServer)
    httpServer.Start(":8091")
}
```

### Option 2: Use stdio transport

For simpler integration with llm-api:

```go
func main() {
    mcpServer := server.NewMCPServer("mcp-tools", "1.0.0")
    serperMCP.RegisterTools(mcpServer)
    server.ServeStdio(mcpServer)
}
```

## Files to Fix

1. **`interfaces/httpserver/routes/serper_mcp.go`**
   - Fix `CallToolRequest` parameter extraction
   - Use `request.Params.Arguments` map directly
   - No helper methods like `RequireString` available

2. **`interfaces/httpserver/routes/mcp_route.go`**
   - Simplify to just create and start MCP server
   - Don't try to integrate into Gin router

3. **`main.go`**
   - Choose between HTTP or stdio transport
   - Don't mix Gin and MCP servers

## Next Steps

1. Decide on transport: HTTP or stdio
2. Rewrite tool handlers to use `Params.Arguments` map
3. Simplify main.go to use one of the recommended approaches
4. Test with MCP client (stdio or HTTP)
