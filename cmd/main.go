package main

import (
	"context"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"cochl-mcp-server/common"
	"cochl-mcp-server/tools"
)

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

func run() error {
	s := newServer()
	srv := server.NewStdioServer(s)
	srv.SetContextFunc(common.ExtractCochlSenseApiClientFromEnv)

	return srv.Listen(context.Background(), os.Stdin, os.Stdout)
}

func main() {
	if err := run(); err != nil {
		log.Printf("Server error: %v\n", err)
		os.Exit(1)
	}
}
