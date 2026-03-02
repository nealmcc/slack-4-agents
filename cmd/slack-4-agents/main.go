package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	slackmcp "go.mcconachie.co/slack-4-agents/internal/mcp"
	slackclient "go.mcconachie.co/slack-4-agents/internal/slack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println(version)
		return
	}
	cfg := createConfig()
	initWorkDir(cfg.WorkDir)
	logger := initLogger(cfg.LogLevel, cfg.LogDir)
	defer logger.Sync()

	server := newServer(logger, cfg)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		logger.Fatal("Server error", zap.Error(err))
	}
}

func createConfig() slackclient.Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	baseDir := filepath.Join(homeDir, ".claude", "servers", "slack-4-agents")
	cfg := slackclient.Config{
		Token:    os.Getenv("SLACK_TOKEN"),
		Cookie:   os.Getenv("SLACK_COOKIE"),
		LogLevel: os.Getenv("LOG_LEVEL"),
		WorkDir:  baseDir,
		LogDir:   filepath.Join(baseDir, "logs"),
	}
	return cfg
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

func newServer(logger *zap.Logger, cfg slackclient.Config) *mcp.Server {
	responseDir := filepath.Join(cfg.WorkDir, "responses")
	responses := slackclient.NewFileResponseWriter(responseDir)

	logger.Info("Creating Slack client")
	client, err := slackclient.NewClient(cfg, logger, responses)
	if err != nil {
		logger.Fatal("Failed to create Slack client", zap.Error(err))
	}

	server := slackmcp.CreateServer(logger, client)
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

	// Create cores for stderr and file
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

	// Combine both cores
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
