package app

import (
	"io"
	"log/slog"
)

// NewLogger builds the process logger: log/slog, structured, to standard error
// (architecture 9).
//
// The response-time header is deliberately not a log line. behaviours 1.9
// measured that the reference's two configuration flags gate a slow-response
// log line and not the header, which is unconditional — so nothing here may
// grow into a place where a level decides whether a header is sent.
func NewLogger(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level}))
}
