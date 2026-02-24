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
)

type Logger struct {
	w []io.Writer
}

type LoggerOptions struct {
	Level int
}

func New(ctx context.Context, options *LoggerOptions) (context.Context, *Logger) {
	ctx = InitializeWideEvent(ctx, options)
	return ctx, &Logger{w: []io.Writer{}}
}

func (log *Logger) AddSink(w io.Writer) {
	log.w = append(log.w, w)
}

func (log *Logger) Info(ctx context.Context) {
	log.logEvent(getWideEvent(ctx))
}

func (log *Logger) Error(ctx context.Context, err error) {
	event := getWideEvent(ctx)
	event.data["level"] = "ERROR"
	event.data["error"] = err.Error()
	log.logEvent(event)
}

func (log *Logger) logEvent(event WideEvent) {
	start := event.data["time"].(time.Time)
	event.data["duration"] = time.Since(start)
	b, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}
	for _, w := range log.w {
		_, err = fmt.Fprintln(w, string(b))
		if err != nil {
			panic(err)
		}
	}
}
