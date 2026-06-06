package log

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type logger struct {
	Sinks     []io.WriteCloser
	wideEvent wideEvent
	done      bool
}

type contextKey string

var CONTEXT_KEY = contextKey("logger")

// Initialize a logger on the context
// events can be logged with the log.Track(...) methods
// When log.Done() is called, get log message will be written to all sinks.
func ContextLogger(ctx context.Context) context.Context {
	logger := logger{wideEvent: newWideEvent()}
	ctx = context.WithValue(ctx, CONTEXT_KEY, &logger)
	return ctx
}

// Add a sink to the logger
func AddSink(ctx context.Context, w io.WriteCloser) error {
	logger, err := getLogger(ctx)
	if err != nil {
		return err
	}
	logger.Sinks = append(logger.Sinks, w)
	return nil
}

// Write the log event and close all sinks
func Done(ctx context.Context) error {
	logger, err := getLogger(ctx)
	if err != nil {
		return err
	}
	if logger.done {
		return nil
	}
	err = logger.commit()
	if err != nil {
		return err
	}
	for _, sink := range logger.Sinks {
		err := sink.Close()
		if err != nil {
			return err
		}
	}
	logger.done = true
	return err
}

// Report a fatal error. Will set the level to "ERROR" on the log message and will
// Log the error. Automatically calls Done
func Fatal(ctx context.Context, e error) error {
	logger, err := getLogger(ctx)
	if err != nil {
		return err
	}
	logger.wideEvent.fatal(e)
	return Done(ctx)
}

// Displays the formatted log to stderr
func Display(ctx context.Context) error {
	logger, err := getLogger(ctx)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(logger.wideEvent, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, string(b))
	return nil
}

// Track an event in the log message.
// The event will be automatically included in the log message when log.Done() is
// Called. The caller should also call the Done method on the event to keep Track
// of the duration.
// If no logger is initialized on the context does noting
func Track(ctx context.Context, eventName string) Event {
	e := newEvent(eventName)
	logger, err := getLogger(ctx)
	if err == nil {
		logger.wideEvent.add(e)
	}
	return e
}

func getLogger(ctx context.Context) (*logger, error) {
	logger, ok := ctx.Value(CONTEXT_KEY).(*logger)
	if !ok {
		return nil, fmt.Errorf("must call WithLogger on context first")
	}
	return logger, nil
}

func (logger *logger) commit() error {
	logger.wideEvent.done()
	b, err := json.Marshal(logger.wideEvent)
	if err != nil {
		return err
	}
	for _, w := range logger.Sinks {
		_, err = fmt.Fprintln(w, string(b))
		if err != nil {
			return err
		}
	}
	return nil
}
