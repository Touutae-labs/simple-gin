package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// requestIDHeader is the canonical header the gateway (or curl) sets;
// the middleware generates one when it's missing.
const requestIDHeader = "X-Request-Id"

// requestIDCtxKey is the unexported context.Context key used to thread
// the request id into handlers' ctx, so slog context-aware logs carry
// it without each call site having to add it.
type requestIDCtxKey struct{}

func requestIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(requestIDCtxKey{}).(string)
	return v
}


// requestIDMiddleware sets X-Request-Id on the response and threads
// the id into the request's context.
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}

		c.Writer.Header().Set(requestIDHeader, id)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestIDCtxKey{}, id))
		c.Next()
	}
}


// bodyLimitMiddleware caps request body size via http.MaxBytesReader.
// Without it an attacker can OOM the process with one multi-GB POST.
func bodyLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil && c.Request.ContentLength != 0 {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}

		c.Next()
	}
}


// metricsMiddleware records every request into Prometheus. Path is
// the route pattern (e.g. /product/:id), not the raw URL, so
// per-id cardinality doesn't blow up the time-series database.
func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		httpInFlight.Inc()
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}

		status := c.Writer.Status()
		httpRequestsTotal.WithLabelValues(c.Request.Method, path, http.StatusText(status)).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
		httpInFlight.Dec()
	}
}


// slogRequestLogger emits one structured access log per request.
// request_id is auto-added by NewSlogHandler.
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

		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}

		slog.LogAttrs(c.Request.Context(), level, "http",
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("duration", time.Since(start)),
			slog.Int("size", c.Writer.Size()),
		)
	}
}


// corsMiddleware turns the configured allowlist into gin-contrib/cors.
// Empty list = no CORS headers, which is safer than "allow everything"
// for a server-to-server API.
func corsMiddleware(allowed []string) gin.HandlerFunc {
	if len(allowed) == 0 {
		return func(c *gin.Context) { c.Next() }
	}

	return cors.New(cors.Config{
		AllowOrigins:     allowed,
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", requestIDHeader, "Authorization"},
		ExposeHeaders:    []string{requestIDHeader},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}


// recoveryMiddleware turns panics into a structured 500 and logs the
// stack through slog. request_id is auto-added by NewSlogHandler.
func recoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.LogAttrs(c.Request.Context(), slog.LevelError, "http.panic",
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
					slog.String("panic", fmt.Sprintf("%v", rec)),
					slog.String("stack", string(debug.Stack())),
				)
				if !c.Writer.Written() {
					c.JSON(http.StatusInternalServerError, gin.H{
						"successful": false,
						"error_code": "INTERNAL_ERROR",
					})
				}

				c.Abort()
			}
		}()
		c.Next()
	}
}


// ShutdownTimeoutContext returns a context with a deadline for graceful
// shutdown. timeoutSec is clamped to [1, 300] seconds to reject
// pathological configs without refusing to start.
func ShutdownTimeoutContext(parent context.Context, timeoutSec int) (context.Context, context.CancelFunc) {
	switch {
	case timeoutSec <= 0:
		timeoutSec = 25
	case timeoutSec > 300:
		timeoutSec = 300
	}

	return context.WithTimeout(parent, time.Duration(timeoutSec)*time.Second)
}
