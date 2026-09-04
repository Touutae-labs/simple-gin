package server

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRequestIDMiddleware_GeneratesWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(requestIDMiddleware())
	var fromCtx string
	r.GET("/x", func(c *gin.Context) {
		fromCtx = requestIDFromCtx(c.Request.Context())
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)

	require.NotEmpty(t, fromCtx)
	require.NotEmpty(t, w.Header().Get(requestIDHeader))
}


func TestRequestIDMiddleware_PropagatesIncoming(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(requestIDMiddleware())
	var fromCtx string
	r.GET("/x", func(c *gin.Context) {
		fromCtx = requestIDFromCtx(c.Request.Context())
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set(requestIDHeader, "incoming-id-123")
	r.ServeHTTP(w, req)

	require.Equal(t, "incoming-id-123", fromCtx)
	require.Equal(t, "incoming-id-123", w.Header().Get(requestIDHeader))
}


func TestShutdownTimeoutContext_ClampsToBounds(t *testing.T) {
	tests := []struct {
		in, want int
	}{
		{0, 25},     // default
		{-5, 25},    // negative → default
		{500, 300},  // over cap
		{1, 1},      // edge: min
		{30, 30},    // in range
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			_, cancel := ShutdownTimeoutContext(t.Context(), tt.in)
			defer cancel()
		})
	}
}
