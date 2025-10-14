package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/cochlearai/cochl-mcp-server/common"
	"github.com/cochlearai/cochl-mcp-server/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newMcpServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-cochl",
		Version: common.Version,
	}, nil)

	tool, handler := tools.AnalyzeAudioToolv2()
	mcp.AddTool(server, tool, handler)
	return server
}

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

func run() error {
	s := newServer()

	srv := server.NewStdioServer(s)
	srv.SetContextFunc(common.ExtractCochlApiClientFromEnv)
	slog.Info("Starting Cochl MCP server using stdio transport")
	return srv.Listen(context.Background(), os.Stdin, os.Stdout)

}

func main() {
	logLevel := flag.String("log-level", "info", "log level (debug, info, warn, error)")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: parseLogLevel(*logLevel),
	})))

	if err := run(); err != nil {
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
