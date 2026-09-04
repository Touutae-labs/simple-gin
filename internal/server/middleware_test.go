package server

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDMiddleware_GeneratesWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(requestIDMiddleware())
	var seen string
	r.GET("/x", func(c *gin.Context) {
		seen = RequestIDFromContext(c)
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)

	if seen == "" {
		t.Fatal("expected request id on context, got empty")
	}
	if got := w.Header().Get(requestIDHeader); got == "" {
		t.Fatal("expected X-Request-Id on response header, got empty")
	}
}

func TestRequestIDMiddleware_PropagatesIncoming(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(requestIDMiddleware())
	var seen string
	r.GET("/x", func(c *gin.Context) {
		seen = RequestIDFromContext(c)
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set(requestIDHeader, "incoming-id-123")
	r.ServeHTTP(w, req)

	if seen != "incoming-id-123" {
		t.Fatalf("expected incoming id, got %q", seen)
	}
	if got := w.Header().Get(requestIDHeader); got != "incoming-id-123" {
		t.Fatalf("expected echoed header, got %q", got)
	}
}
