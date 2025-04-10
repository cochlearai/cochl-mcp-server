package main

import (
	"context"
	"flag"
	"log/slog"
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
