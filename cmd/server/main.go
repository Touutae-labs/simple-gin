// Command server boots the HTTP service. Configuration is read
// from APP_CONFIG (default: ./config.yml).
package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"

	"github.com/Touutae-labs/simple-gin/internal/configurations"
	"github.com/Touutae-labs/simple-gin/internal/di"
	"github.com/Touutae-labs/simple-gin/internal/server"
)

// Version is overwritten at link time via -ldflags="-X main.Version=…".
var Version = "dev"

const serverName = "simple-gin"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	app, cleanup, err := di.Initialize(cfg, server.ServerTitle(serverName), server.ServerVersion(Version))
	if err != nil {
		log.Fatalf("init: %v", err)
	}
	defer cleanup()

	go func() {
		<-ctx.Done()
		slog.Info("shutdown requested")
		cleanup()
		os.Exit(0)
	}()

	if err := app.Server.Start(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func loadConfig() (configurations.Config, error) {
	var cfg configurations.Config
	path := os.Getenv("APP_CONFIG")
	if path == "" {
		path = "config.yml"
	}
	k := koanf.New(".")
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return cfg, err
	}
	if err := k.Unmarshal("", &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
