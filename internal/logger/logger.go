// using standard library logger log/slog

package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/Suthar345Piyush/videoclippingpipeline/internal/config"
)

type ctxKey string

const (
	loggerCtxKey   ctxKey = "logger"
	requestIDField        = "request_id"
	jobIDField            = "job_id"
)

// logger build

func New(cfg config.LoggerConfig) (*slog.Logger, error) {

	level, err := parseLevel(cfg.Level)

	if err != nil {
		return nil, err
	}

	w, err := resolveWriter(cfg.OutputPath)

	if err != nil {
		return nil, err
	}

	// text or json handler

	handlerOps := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	var handler slog.Handler

	switch strings.ToLower(cfg.Format) {
	case "text":
		handler = slog.NewTextHandler(w, handlerOps)

	case "json", "":
		handler = slog.NewJSONHandler(w, handlerOps)

	default:
		return nil, fmt.Errorf("logger: unsupported format %q (want json|text)", cfg.Format)
	}

	return slog.New(handler), nil
}

// function to parse the log level

func parseLevel(level string) (slog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug, nil

	case "warn":
		return slog.LevelWarn, nil

	case "error":
		return slog.LevelError, nil

	case "info":
		return slog.LevelInfo, nil

	default:
		return slog.LevelInfo, fmt.Errorf("logger: wrong log level %q (want debug|info|warn|error)", level)

	}

}

// resolve output path - resolve writer
// stdout or stderr

func resolveWriter(outputPath string) (io.Writer, error) {
	switch strings.ToLower(outputPath) {
	case "", "stdout":
		return os.Stdout, nil

	case "stderr":
		return os.Stderr, nil

	default:
		f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)

		if err != nil {
			return nil, fmt.Errorf("logger: failed to open log file %q: %w", outputPath, err)
		}

		return f, nil

	}

}

// context with the logger

func WithContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey, l)
}

func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerCtxKey).(*slog.Logger); ok && l != nil {
		return l
	}

	return slog.Default()

}

// logger with the request id

func WithRequestID(l *slog.Logger, requestID string) *slog.Logger {
	return l.With(slog.String(requestIDField, requestID))
}

// logger with the job id

func WithJobID(l *slog.Logger, jobID string) *slog.Logger {
	return l.With(slog.String(jobIDField, jobID))
}

// with component

func WithComponent(l *slog.Logger, component string) *slog.Logger {
	return l.With(slog.String("component", component))
}
