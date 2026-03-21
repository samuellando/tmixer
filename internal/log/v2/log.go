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

type logger struct {
	options   LogOptions
	wideEvent wideEvent
}

type LogOptions struct {
	Level int
	Sinks []io.WriteCloser
}

type contextKey string

var CONTEXT_KEY = contextKey("logger")

// Initialize a logger on the context
// events can be logged with the log.Track(...) methods
// When log.Done() is called, get log message will be written to all sinks.
func ContextLogger(ctx context.Context, options LogOptions) context.Context {
	logger := logger{options: options, wideEvent: newWideEvent()}
	ctx = context.WithValue(ctx, CONTEXT_KEY, &logger)
	return ctx
}

// Sets the context logger's options
func SetOptions(ctx context.Context, options LogOptions) {
	logger := getLogger(ctx)
	logger.options = options
}

// Add a sink to the logger
func AddSink(ctx context.Context, w io.WriteCloser) {
	logger := getLogger(ctx)
	logger.options.Sinks = append(logger.options.Sinks, w)
}

// Write the log event and close all sinks
func Done(ctx context.Context) {
	logger := getLogger(ctx)
	logger.commit()
	for _, sink := range logger.options.Sinks {
		err := sink.Close()
		if err != nil {
			panic(err)
		}
	}
}

// Report a fatal error. Will set the level to "ERROR" on the log message and will
// Log the error. Automatically calls Done
func Fatal(ctx context.Context, err error) {
	logger := getLogger(ctx)
	logger.wideEvent.fatal(err)
	Done(ctx)
}

// Track an event in the log message.
// The event will be automatically included in the log message when log.Done() is
// Called. The caller should also call the Done method on the event to keep Track
// of the duration.
func Track(ctx context.Context, eventName string) Event {
	logger := getLogger(ctx)
	e := newEvent(eventName)
	logger.wideEvent.add(e)
	return e
}

// Track an event.
// but will only actually include it in the log message if the log level is sufficient.
// Also see the log.Track
func TrackLevel(ctx context.Context, level int, eventName string) Event {
	logger := getLogger(ctx)
	e := event{name: eventName, time: time.Now()}
	if level >= logger.options.Level {
		logger.wideEvent.add(&e)
	}
	return &e
}

func getLogger(ctx context.Context) *logger {
	logger, ok := ctx.Value(CONTEXT_KEY).(*logger)
	if !ok {
		panic("Must call WithLogger on context first")
	}
	return logger
}

func (logger *logger) commit() {
	logger.wideEvent.done()
	b, err := json.Marshal(logger.wideEvent)
	if err != nil {
		panic(err)
	}
	for _, w := range logger.options.Sinks {
		_, err = fmt.Fprintln(w, string(b))
		if err != nil {
			panic(err)
		}
	}
}
