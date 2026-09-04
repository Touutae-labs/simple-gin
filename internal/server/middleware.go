package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// requestIDHeader is the canonical header the gateway (or curl) sets,
// and the middleware will generate one if it's missing.
const requestIDHeader = "X-Request-Id"

// requestIDKey is the gin.Context key under which the per-request id
// is stored. Both middleware and handlers use it.
const requestIDKey = "request_id"

// RequestIDFromContext returns the request id stored on the gin.Context,
// or "" if no middleware ran.
func RequestIDFromContext(c *gin.Context) string {
	if v, ok := c.Get(requestIDKey); ok {
		if s, ok := v.(string); ok { return s }
	}
	return ""
}

// requestIDMiddleware sets an X-Request-Id on the request (generates
// one if absent) and echoes it back on the response so the caller
// can correlate. Stored on the gin.Context and threaded into the
// per-request log line.
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(requestIDKey, id)
		c.Writer.Header().Set(requestIDHeader, id)
		c.Next()
	}
}

// slogRequestLogger emits one structured log line per request:
// method, path, status, duration, bytes, and request id. Level
// follows the status (5xx→Error, 4xx→Warn, else→Info).
func slogRequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		level := slog.LevelInfo
		switch {
		case c.Writer.Status() >= 500:
			level = slog.LevelError
		case c.Writer.Status() >= 400:
			level = slog.LevelWarn
		}
		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("duration", time.Since(start)),
			slog.Int("size", c.Writer.Size()),
			slog.String("request_id", RequestIDFromContext(c)),
		}
		slog.LogAttrs(c.Request.Context(), level, "http", attrs...)
	}
}

// ShutdownContext returns a context with a deadline for graceful
// shutdown. Used by main.go to drain in-flight requests.
func ShutdownContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// ShutdownTimeoutContext returns a context with a deadline for graceful
// shutdown. Used by main.go to drain in-flight requests. timeoutSec
// is clamped to [1, 300] seconds to avoid pathological configs.
func ShutdownTimeoutContext(parent context.Context, timeoutSec int) (context.Context, context.CancelFunc) {
	if timeoutSec <= 0 {
		timeoutSec = 25
	}
	if timeoutSec > 300 {
		timeoutSec = 300
	}
	return context.WithTimeout(parent, time.Duration(timeoutSec)*time.Second)
}
