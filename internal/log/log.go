package log

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

const (
	LEVEL_DEBUG = -1
	LEVEL_INFO  = 0
	LEVEL_ERROR = 1
)

type Logger struct {
	w io.Writer
}

type LoggerOptions struct {
	Level int
}

func New(ctx context.Context, w io.Writer, options *LoggerOptions) (context.Context, *Logger) {
	ctx = InitializeWideEvent(ctx, options)
	return ctx, &Logger{w: w}
}

func (log *Logger) Info(ctx context.Context) {
	log.logEvent(getWideEvent(ctx))
}

func (log *Logger) Error(ctx context.Context, err error) {
	event := getWideEvent(ctx)
	event["level"] = "ERROR"
	event["error"] = err.Error()
	log.logEvent(event)
}

func (log *Logger) logEvent(event WideEvent) {
	start := event["time"].(time.Time)
	event["duration"] = time.Since(start)
	b, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}
	fmt.Fprintln(log.w, string(b))
}
