// Command server boots the HTTP service. Configuration is read
// from APP_CONFIG (default: ./config.yml). Log format: APP_LOG=json
// forces JSON, anything else (including unset) is the colored tint
// text handler for dev.
//
// Lifecycle:
//   1. Load config + set up logging
//   2. Run DI initialization (db, repos, services, controllers, engine)
//   3. Start the HTTP server in a goroutine
//   4. Block on a SIGINT/SIGTERM context
//   5. On signal: ask the server to Shutdown with a deadline, drain
//      in-flight requests, then exit. In-flight requests get up to
//      server.shutdown_timeout_sec to finish before the process dies.
package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/lmittmann/tint"

	"github.com/Touutae-labs/simple-gin/internal/configurations"
	"github.com/Touutae-labs/simple-gin/internal/di"
	"github.com/Touutae-labs/simple-gin/internal/server"
)

// Version is overwritten at link time via -ldflags="-X main.Version=…".
var Version = "dev"

const serverName = "simple-gin"

func main() {
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
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

	// Run the HTTP server in a goroutine so we can block on the
	// signal context here and then trigger a graceful shutdown.
	serverErrCh := make(chan error, 1)
	go func() {
		slog.Info("server.starting")
		serverErrCh <- app.Server.Start()
	}()

	select {
	case err := <-serverErrCh:
		// Start() returned on its own (e.g. port already in use).
		if err != nil {
			slog.Error("server.failed_to_start", slog.String("err", err.Error()))
		}
		// Run cleanup anyway so db handles are released.
		cleanup()
		os.Exit(1)
		return
	case <-rootCtx.Done():
		slog.Info("signal.received", slog.String("signal", "SIGINT/SIGTERM"))
	}

	// Give in-flight requests a deadline to finish. Anything still
	// running after ShutdownTimeoutSec gets cut off.
	shutdownCtx, cancelShutdown := server.ShutdownTimeoutContext(rootCtx, cfg.Server.ShutdownTimeoutSec)
	defer cancelShutdown()

	if err := app.Server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server.shutdown_error", slog.String("err", err.Error()))
	}

	cleanup()
	slog.Info("server.exit_clean")
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
