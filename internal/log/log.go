package log

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type Logger struct {
	w io.Writer
}

func New(w io.Writer) *Logger {
	return &Logger{w: w}
}

func (log *Logger) Info(ctx context.Context) {
	log.logEvent(GetWideEvent(ctx))
}

func (log *Logger) Error(ctx context.Context, err error) {
	event := GetWideEvent(ctx)
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
