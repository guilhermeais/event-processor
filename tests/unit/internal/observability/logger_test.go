package observability

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/guilhermeais/event-processor/internal/observability"
)

type logEntry map[string]interface{}

func createLogger(t *testing.T) (*observability.Logger, *bytes.Buffer) {
	t.Helper()

	var buf bytes.Buffer
	slogJsonHandler := slog.NewJSONHandler(&buf, nil)
	baseLogger := slog.New(slogJsonHandler)
	logger := observability.NewLogger(baseLogger)

	return logger, &buf
}

func logStringToLogEntry(t *testing.T, buf *bytes.Buffer) logEntry {
	t.Helper()
	output := buf.String()

	var logEntry logEntry
	err := json.Unmarshal([]byte(output), &logEntry)
	if err != nil {
		t.Fatalf("failed to parse log: %v", err)
	}
	return logEntry
}

func assertStringField(t *testing.T, l logEntry, key, expected string) {
	t.Helper()
	actual, ok := l[key].(string)
	if !ok || actual != expected {
		t.Errorf("expected field %q to be %v, fot %v (%T)", key, expected, l[key], l[key])
	}
}

func TestLogger(t *testing.T) {
	t.Run("should log success", func(t *testing.T) {
		logger, buf := createLogger(t)
		logger.AddAttribute("client_id", "abc")
		logger.Emit("processing")

		logEntry := logStringToLogEntry(t, buf)

		assertStringField(t, logEntry, "status", "SUCCESS")
		assertStringField(t, logEntry, "client_id", "abc")
	})

	t.Run("should log error", func(t *testing.T) {
		logger, buf := createLogger(t)

		logger.AddAttribute("client_id", "abc")
		logger.AddError(errors.New("an error"))
		logger.Emit("processing")

		logEntry := logStringToLogEntry(t, buf)

		assertStringField(t, logEntry, "status", "FAILED")
		assertStringField(t, logEntry, "client_id", "abc")
		assertStringField(t, logEntry, "error", "an error")
	})

	t.Run("should log correct duratino", func(t *testing.T) {
		logger, buf := createLogger(t)
		logger.AddAttribute("client_id", "abc")
		time.Sleep(50 * time.Millisecond)
		logger.Emit("processed")

		logEntry := logStringToLogEntry(t, buf)

		assertStringField(t, logEntry, "status", "SUCCESS")
		assertStringField(t, logEntry, "client_id", "abc")

		actual, ok := logEntry["duration_ms"].(float64)
		if !ok || actual != 50 {
			t.Fatalf("expected duration_ms of 50ms got %f", actual)
		}
	})
}
