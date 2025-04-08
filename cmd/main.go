package main

import (
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"cochl-mcp-server/common"
	"cochl-mcp-server/tools"
)

func main() {
	apikey := common.GetCochlSenseProjectKey()
	if apikey == "" {
		log.Fatal("project-key is not set")
	}

	// Create a new MCP server
	s := server.NewMCPServer(
		"Cochl Sense",
		common.Version,
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	// Add a file processing tool
	cochlTool := mcp.NewTool("cochl",
		mcp.WithDescription("Analyze an audio file"),
		mcp.WithString("os_type",
			mcp.Required(),
			mcp.Description("Operating system type (e.g., 'windows' or 'unix')"),
		),
		mcp.WithString("file_absolute_path",
			mcp.Required(),
			mcp.Description(`Please provide the absolute path to the file,
formatted according to the operating system (e.g., Windows uses backslashes \, Unix uses slashes /).
Avoid using URL-encoded characters.`),
		),
	)

	s.AddTool(cochlTool, tools.CochlSenseTool)

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
