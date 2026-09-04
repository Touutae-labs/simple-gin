package server

import (
	"context"
	"log/slog"
)

// NewSlogHandler wraps inner so every record carries the per-request
// id read from ctx. Without it, every controller / repo / panic
// log would have to re-add the id by hand, and one would eventually
// be missed.
func NewSlogHandler(inner slog.Handler) slog.Handler {
	return &ctxRequestIDHandler{inner: inner}
}


type ctxRequestIDHandler struct{ inner slog.Handler }

func (h *ctxRequestIDHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.inner.Enabled(ctx, l)
}


func (h *ctxRequestIDHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := requestIDFromCtx(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}

	return h.inner.Handle(ctx, r)
}


func (h *ctxRequestIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ctxRequestIDHandler{inner: h.inner.WithAttrs(attrs)}
}


func (h *ctxRequestIDHandler) WithGroup(name string) slog.Handler {
	return &ctxRequestIDHandler{inner: h.inner.WithGroup(name)}
}
