package observability

import (
	"fmt"
	"log/slog"
	"os"
	"time"
)

type Logger struct {
	startTime  time.Time
	attributes map[string]any
	err        error
	baseLogger *slog.Logger
}

func NewLogger(baseLogger *slog.Logger) *Logger {
	return &Logger{
		startTime:  time.Now(),
		attributes: map[string]any{},
		baseLogger: baseLogger,
	}
}

func NewLoggerDefault() *Logger {
	slogJsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	baseLogger := slog.New(slogJsonHandler)
	return NewLogger(baseLogger)
}

func (l *Logger) AddAttribute(key string, value any) {
	// Temporary debug to catch nil map
	if l.attributes == nil {
		panic(fmt.Sprintf("PANIC: attributes is nil! Logger state: startTime=%v, err=%v, baseLogger=%v", l.startTime, l.err, l.baseLogger != nil))
	}
	l.attributes[key] = value
}

func (l *Logger) AddError(err error) {
	l.err = err
}

func (l *Logger) Emit(msg string) {
	duration := time.Since(l.startTime).Milliseconds()
	args := []any{
		slog.Int64("duration_ms", duration),
	}

	for k, v := range l.attributes {
		args = append(args, slog.Any(k, v))
	}

	if l.err != nil {
		args = append(args, slog.String("error", l.err.Error()))
		args = append(args, slog.String("status", "FAILED"))
		l.baseLogger.Error(msg, args...)
	} else {
		args = append(args, slog.String("status", "SUCCESS"))
		l.baseLogger.Info(msg, args...)
	}
}
