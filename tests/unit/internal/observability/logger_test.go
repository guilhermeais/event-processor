package observability

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	testhelpers "github.com/guilhermeais/event-processor/tests/unit/testhelpers"
	"github.com/stretchr/testify/assert"
)

type logEntry map[string]interface{}

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

func TestLogger(t *testing.T) {
	t.Run("should log success", func(t *testing.T) {
		logger, buf := testhelpers.CreateLogger(t)
		logger.AddAttribute("client_id", "abc")
		logger.Emit("processing")

		logEntry := logStringToLogEntry(t, buf)

		assert.Equal(t, logEntry["status"], "SUCCESS")
		assert.Equal(t, logEntry["client_id"], "abc")
	})

	t.Run("should log error", func(t *testing.T) {
		logger, buf := testhelpers.CreateLogger(t)

		logger.AddAttribute("client_id", "abc")
		logger.AddError(errors.New("an error"))
		logger.Emit("processing")

		logEntry := logStringToLogEntry(t, buf)
		assert.Equal(t, logEntry["status"], "FAILED")
		assert.Equal(t, logEntry["client_id"], "abc")
		assert.Equal(t, logEntry["error"], "an error")
	})

	t.Run("should log correct duratino", func(t *testing.T) {
		logger, buf := testhelpers.CreateLogger(t)
		logger.AddAttribute("client_id", "abc")
		time.Sleep(50 * time.Millisecond)
		logger.Emit("processed")

		logEntry := logStringToLogEntry(t, buf)

		assert.Equal(t, logEntry["status"], "SUCCESS")
		assert.Equal(t, logEntry["client_id"], "abc")
		assert.Equal(t, logEntry["duration_ms"], float64(50))
	})
}
