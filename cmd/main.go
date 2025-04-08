package main

import (
	"log"

	"github.com/mark3labs/mcp-go/server"

	"cochl-mcp-server/common"
	"cochl-mcp-server/tools"
)

func newServer() *server.MCPServer {
	// Create a new MCP server
	s := server.NewMCPServer(
		"mcp-cochl",
		common.Version,
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	s.AddTool(tools.Sense())

	return s
}

func main() {
	apikey := common.GetCochlSenseProjectKey()
	if apikey == "" {
		log.Fatal("project-key is not set")
	}

	baseUrl := common.GetCochlSenseBaseURL()
	log.Printf("Connecting to %s", baseUrl)

	mcpServer := newServer()
	// Start the server
	if err := server.ServeStdio(mcpServer); err != nil {
		log.Printf("Server error: %v\n", err)
	}
}
