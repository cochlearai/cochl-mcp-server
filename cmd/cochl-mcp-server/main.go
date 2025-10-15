package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cochlearai/cochl-mcp-server/common"
	"github.com/cochlearai/cochl-mcp-server/tools"
)

func newMcpServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-cochl",
		Version: common.Version,
	}, nil)

	tool, handler := tools.AnalyzeAudioTool()
	mcp.AddTool(server, tool, handler)
	return server
}

func run() error {
	server := newMcpServer()
	ctx := common.ExtractCochlApiClientFromEnv(context.Background())
	return server.Run(ctx, &mcp.StdioTransport{})
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
