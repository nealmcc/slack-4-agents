package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mcconachie.co/slack-4-agents/internal/slack"
	"go.mcconachie.co/slack-4-agents/internal/slackapi"
	"go.mcconachie.co/slack-4-agents/internal/slackmcp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println(version)
		return
	}

	token := os.Getenv("SLACK_TOKEN")
	cookie := os.Getenv("SLACK_COOKIE")
	logLevel := os.Getenv("LOG_LEVEL")

	if token == "" {
		log.Fatal("SLACK_TOKEN is required")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	workDir := filepath.Join(homeDir, ".claude", "servers", "slack-4-agents")
	logDir := filepath.Join(workDir, "logs")

	initWorkDir(workDir)
	logger := initLogger(logLevel, logDir)
	defer logger.Sync()

	server := initServer(logger, token, cookie, workDir)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		logger.Fatal("Server error", zap.Error(err))
	}
}

func initWorkDir(workDir string) {
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		log.Fatalf("Failed to create work directory: %v", err)
	}
	responseDir := filepath.Join(workDir, "responses")
	if err := os.MkdirAll(responseDir, 0o755); err != nil {
		log.Fatalf("Failed to create responses directory: %v", err)
	}
}

func initServer(logger *zap.Logger, token, cookie, workDir string) *mcp.Server {
	logger.Info("Creating Slack client")

	responseDir := filepath.Join(workDir, "responses")
	responses := slack.NewFileResponseWriter(responseDir)

	api := slackapi.NewClient(token, cookie, logger)
	client := slack.NewService(api, logger, responses)

	server := slackmcp.NewServer(logger, client)
	return server
}

func initLogger(level string, logDir string) *zap.Logger {
	logLevel := interpretLogLevel(level)

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}
	logFileName := fmt.Sprintf("slack-4-agents-%s.log", time.Now().Format("2006-01-02"))
	logFilePath := filepath.Join(logDir, logFileName)
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	stderrCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stderr),
		logLevel,
	)

	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(logFile),
		logLevel,
	)

	core := zapcore.NewTee(stderrCore, fileCore)

	logger := zap.New(core, zap.AddCaller())
	return logger
}

func interpretLogLevel(level string) zapcore.Level {
	var logLevel zapcore.Level

	switch level {
	case "debug":
		logLevel = zapcore.DebugLevel
	case "warn":
		logLevel = zapcore.WarnLevel
	case "error":
		logLevel = zapcore.ErrorLevel
	default:
		logLevel = zapcore.InfoLevel
	}
	return logLevel
}
