// Command server boots the HTTP service. Configuration is read
// from APP_CONFIG (default: ./config.yml). Log format: APP_LOG=json
// forces JSON, anything else (including unset) is the colored tint
// text handler for dev.
package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lmittmann/tint"
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

	slog.SetDefault(slog.New(newLogHandler()))

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

// newLogHandler returns the slog handler for the binary. APP_LOG=json
// gives structured JSON (production). Anything else gives tinted
// text (development): colored level, bold key=value pairs.
func newLogHandler() slog.Handler {
	level := slog.LevelInfo
	if v := os.Getenv("APP_LOG_LEVEL"); v != "" {
		_ = level.UnmarshalText([]byte(v))
	}
	opts := &tint.Options{Level: level, TimeFormat: "15:04:05.000", NoColor: false}
	if strings.EqualFold(os.Getenv("APP_LOG"), "json") {
		return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	return tint.NewHandler(os.Stdout, opts)
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
