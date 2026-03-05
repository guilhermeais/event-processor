package testhelpers

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/guilhermeais/event-processor/internal/observability"
)

func CreateLogger(t *testing.T) (*observability.Logger, *bytes.Buffer) {
	t.Helper()

	var buf bytes.Buffer
	slogJsonHandler := slog.NewJSONHandler(&buf, nil)
	baseLogger := slog.New(slogJsonHandler)
	logger := observability.NewLogger(baseLogger)

	return logger, &buf
}
