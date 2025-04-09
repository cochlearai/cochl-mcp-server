package main

import (
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	"cochl-mcp-server/common"
	"cochl-mcp-server/tools"
)

var rootCmd = &cobra.Command{
	Use:   "cochl-mcp-server",
	Short: "Cochl MCP Server",
	Long:  `Cochl MCP Server that analyzes audio data using Cochl's API.`,
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the Cochl MCP server",
	Long: `Start the Cochl MCP server and listen for connections.
Environment variables:
  COCHL_SENSE_PROJECT_KEY  Your Cochl Sense project key (required)
  COCHL_SENSE_BASE_URL    Cochl API base URL (optional)`,
	Run: func(cmd *cobra.Command, args []string) {
		apikey := common.GetCochlSenseProjectKey()
		if apikey == "" {
			log.Fatal("COCHL_SENSE_PROJECT_KEY is not set")
		}

		baseUrl := common.GetCochlSenseBaseURL()
		log.Printf("Connecting to %s", baseUrl)

		mcpServer := newServer()
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Printf("Server error: %v\n", err)
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Display the version number of Cochl MCP Server`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("Cochl MCP Server %s\n", common.Version)
	},
}

func newServer() *server.MCPServer {
	s := server.NewMCPServer(
		"mcp-cochl",
		common.Version,
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	s.AddTool(tools.Sense())

	return s
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}
