package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	slackclient "github.com/nealmcconachie/slack-mcp/internal/slack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Determine working directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}
	workDir := filepath.Join(homeDir, ".claude", "slack-mcp")

	// Initialize logger
	logger := initLogger(workDir)
	defer logger.Sync()

	logger.Info("Starting Slack MCP server")

	// Create Slack client
	cfg := slackclient.Config{
		Token:   os.Getenv("SLACK_TOKEN"),
		Cookie:  os.Getenv("SLACK_COOKIE"),
		WorkDir: workDir,
	}
	client, err := slackclient.NewClient(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create Slack client", zap.Error(err))
	}

	// Create MCP server
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "slack-mcp",
			Version: "1.0.0",
		},
		nil,
	)

	// Register Slack tools
	client.RegisterTools(server)

	logger.Info("Slack MCP server initialized, starting transport")

	// Run on STDIO transport
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		logger.Fatal("Server error", zap.Error(err))
	}
}

// initLogger creates a zap logger that writes to both stderr and a file
func initLogger(workDir string) *zap.Logger {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	level := zapcore.InfoLevel
	switch logLevel {
	case "debug":
		level = zapcore.DebugLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}

	// Set up encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Ensure directories exist
	if err := os.MkdirAll(workDir, 0755); err != nil {
		log.Fatalf("Failed to create work directory: %v", err)
	}
	cacheDir := filepath.Join(workDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Fatalf("Failed to create cache directory: %v", err)
	}

	logFilePath := filepath.Join(workDir, "slack-mcp.log")

	// Open log file
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// Create cores for stderr and file
	stderrCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stderr),
		level,
	)

	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(logFile),
		level,
	)

	// Combine both cores
	core := zapcore.NewTee(stderrCore, fileCore)

	logger := zap.New(core, zap.AddCaller())
	return logger
}
