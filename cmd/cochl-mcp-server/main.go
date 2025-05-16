package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/cochlearai/cochl-mcp-server/common"
	"github.com/cochlearai/cochl-mcp-server/tools"
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

func run(transport, port string) error {
	s := newServer()

	switch transport {
	case "sse":
		srv := server.NewSSEServer(s,
			server.WithHTTPContextFunc(common.HTTPContextFunc),
		)
		slog.Info("Starting Cochl MCP server using sse transport", "port", port)
		return srv.Start(":" + port)

	case "stdio":
		srv := server.NewStdioServer(s)
		srv.SetContextFunc(common.ExtractCochlApiClientFromEnv)
		slog.Info("Starting Cochl MCP server using stdio transport")
		return srv.Listen(context.Background(), os.Stdin, os.Stdout)

	default:
		return fmt.Errorf("invalid transport: %s", transport)
	}

}

func main() {
	var transport string
	flag.StringVar(&transport, "transport", "stdio", "transport (stdio or sse)")
	flag.StringVar(&transport, "t", "stdio", "transport (stdio or sse)")
	logLevel := flag.String("log-level", "info", "log level (debug, info, warn, error)")
	port := flag.String("sse-port", "8080", "port to listen on (required for sse transport)")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: parseLogLevel(*logLevel),
	})))

	if err := run(transport, *port); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

func parseLogLevel(logLevel string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(logLevel)); err != nil {
		return slog.LevelInfo
	}
	return l
}
