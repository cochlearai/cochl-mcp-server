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
	defaultOpts := []server.ServerOption{
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
		server.WithRecovery(),
	}

	s := server.NewMCPServer(
		"mcp-cochl",
		common.Version,
		defaultOpts...,
	)

	s.AddTool(tools.AnalyzeAudio())
	return s
}

func run(transport, port string) error {
	s := newServer()

	switch transport {
	case "http":
		srv := server.NewStreamableHTTPServer(s,
			server.WithHTTPContextFunc(common.HttpContextFunc),
		)
		slog.Info("Starting Cochl MCP server using streamable http transport", "port", port)
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
	flag.StringVar(&transport, "transport", "stdio", "transport (stdio or http)")
	flag.StringVar(&transport, "t", "stdio", "transport (stdio or http)")
	logLevel := flag.String("log-level", "info", "log level (debug, info, warn, error)")
	port := flag.String("http-port", "8080", "port to listen on (required for streamable http transport)")
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
