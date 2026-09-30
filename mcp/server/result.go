package mcpserver

import (
	"encoding/json"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func toolError(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}
}

func jsonResult(value any) *mcp.CallToolResult {
	data, _ := json.Marshal(value)
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(data)}}, StructuredContent: map[string]any{"result": value}}
}
