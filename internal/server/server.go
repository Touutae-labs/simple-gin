// Package server builds the Gin engine, mounts middleware and
// Swagger, and registers every route.
//
// @title           simple-gin API
// @version         1.0
// @description     Product service — POST /product and PATCH /product/{id}.
// @description     All endpoints return the envelope {successful, error_code, data}.
// @BasePath        /
// @schemes         http https
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/Touutae-labs/simple-gin/internal/configurations"
	"github.com/Touutae-labs/simple-gin/internal/controllers"
)

// ServerTitle and ServerVersion are exported so the cmd entry point
// can pass them in without server reaching back into main.
type ServerTitle string
type ServerVersion string

// ServerConfig is the resolved engine config.
type ServerConfig struct {
	Title              ServerTitle
	Version            ServerVersion
	Port               string
	MaxPayloadSizeKB   int
	TimeoutSeconds     int
	BaseURL            string
	ShutdownTimeoutSec int
	CORSAllowedOrigins []string
	MetricsEnabled     bool
}


// Server wraps the gin engine and the resolved config. cmd/server
// holds one of these and calls Start / Shutdown.
type Server struct {
	App      *gin.Engine
	Config   *ServerConfig
	httpSrv  *http.Server
	registry *prometheus.Registry
}


// BuildServerConfig maps configurations.ServerConfig into the
// server.ServerConfig this package needs. Called from di.
func BuildServerConfig(cfg configurations.ServerConfig, title ServerTitle, version ServerVersion) ServerConfig {
	return ServerConfig{
		Title:              title,
		Version:            version,
		Port:               cfg.Port,
		MaxPayloadSizeKB:   cfg.MaxPayloadSizeKB,
		TimeoutSeconds:     cfg.TimeoutSeconds,
		BaseURL:            cfg.BaseURL,
		ShutdownTimeoutSec: cfg.ShutdownTimeoutSec,
		CORSAllowedOrigins: parseOrigins(cfg.CORSAllowedOrigins),
		MetricsEnabled:     cfg.MetricsEnabled,
	}
}


// parseOrigins turns a comma-separated YAML string into a slice,
// trimming whitespace and dropping empty entries.
func parseOrigins(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}

	return out
}


// Prometheus metrics. Local registry (not global) so re-running tests
// doesn't accumulate duplicates.
var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests processed, labelled by method, path, and status.",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 12),
	}, []string{"method", "path"})

	httpInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_in_flight_requests",
		Help: "Number of HTTP requests currently being processed.",
	})
)

// NewServer builds the engine, mounts middleware + Swagger, and
// registers every route against the supplied controllers.
func NewServer(cfg ServerConfig, c *controllers.Controllers) *Server {
	if cfg.ShutdownTimeoutSec == 0 {
		cfg.ShutdownTimeoutSec = 25
	}

	requestTimeout := 30 * time.Second
	if cfg.TimeoutSeconds > 0 {
		requestTimeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}

	bodyLimit := 4 * 1024 * 1024
	if cfg.MaxPayloadSizeKB > 0 {
		bodyLimit = cfg.MaxPayloadSizeKB * 1024
	}


	gin.SetMode(gin.ReleaseMode)
	app := gin.New()

	// Middleware order matters. requestID first so every later
	// middleware/log can see it, then body cap so oversized POSTs
	// are dropped before any work, then metrics, then access log,
	// then CORS, then recovery last so a panic still hits the log.
	app.Use(
		requestIDMiddleware(),
		bodyLimitMiddleware(int64(bodyLimit)),
		metricsMiddleware(),
		slogRequestLogger(),
		corsMiddleware(cfg.CORSAllowedOrigins),
		recoveryMiddleware(),
	)
	app.MaxMultipartMemory = int64(bodyLimit)

	app.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"successful": false, "error_code": "NOT_FOUND"})
	})

	s := &Server{App: app, Config: &cfg, registry: prometheus.NewRegistry()}
	if cfg.MetricsEnabled {
		s.registry.MustRegister(httpRequestsTotal, httpRequestDuration, httpInFlight)
		app.GET("/metrics", gin.WrapH(promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{})))
	}

	s.mountSwagger()
	newHandler(c).register(s.App)

	// ReadHeaderTimeout stops slow-loris. ReadTimeout caps the
	// whole request including body. WriteTimeout caps the response
	// so a slow client can't pin a worker. IdleTimeout reaps
	// keep-alive sockets. MaxHeaderBytes caps the header block.
	s.httpSrv = &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       requestTimeout + 5*time.Second,
		WriteTimeout:      requestTimeout + 5*time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	return s
}


// Start blocks until Shutdown is called or the listener errors.
func (s *Server) Start() error {
	addr := s.httpSrv.Addr
	slog.Info("server.start",
		slog.String("addr", addr),
		slog.String("docs", s.Config.BaseURL+"/api-docs/index.html"),
		slog.Bool("metrics", s.Config.MetricsEnabled),
	)
	if err := s.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}


// Shutdown drains in-flight requests up to the configured timeout.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("server.shutdown.start", slog.Duration("timeout", time.Duration(s.Config.ShutdownTimeoutSec)*time.Second))
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		return err
	}

	slog.Info("server.shutdown.done")
	return nil
}


func (s *Server) mountSwagger() {
	if s.Config.BaseURL == "" {
		return
	}

	s.App.GET("/api-docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
