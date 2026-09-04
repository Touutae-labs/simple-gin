package server

import (
	"context"
	"log/slog"
)

// NewSlogHandler wraps inner so every record carries the per-request
// id read from ctx. Used by the request logger and by anything that
// logs through slog inside a request — controllers' bind errors,
// repo errors, panic stacks — all show up with the same correlation
// id as the access log line.
//
// Implemented as a Handler (not a Handler middleware) so it sits
// at the bottom of the chain: cheaper than a wrapper that allocates
// a new attr slice on every Enabled() call.
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
