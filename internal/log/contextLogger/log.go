package log

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
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
	sinks     []io.WriteCloser
}

type LogOptions struct {
	Level int
}

type contextKey string

var CONTEXT_KEY = contextKey("logger")

// Initialize a logger on the context
// events can be logged with the log.Track(...) methods
// When the context is cancelled, the events will all be logged into a single
// wide event and the close function will be called on all sinks
func WithLogger(ctx context.Context, options LogOptions) (context.Context, *sync.WaitGroup) {
	logger := logger{options: options, wideEvent: newWideEvent()}
	ctx = context.WithValue(ctx, CONTEXT_KEY, &logger)
	wg := sync.WaitGroup{}
	wg.Add(1)
	context.AfterFunc(ctx, func() {
		defer wg.Done()
		logger := getLogger(ctx)
		logger.commit()
		for _, sink := range logger.sinks {
			err := sink.Close()
			if err != nil {
				panic(err)
			}
		}
	})
	return ctx, &wg
}

// Sets the context logger's options
func SetOptions(ctx context.Context, options LogOptions) {
	logger := getLogger(ctx)
	logger.options = options
}

// Add a sink to the logger, will automatically call close when the context is
// cancelled.
func AddSink(ctx context.Context, w io.WriteCloser) {
	logger := getLogger(ctx)
	logger.sinks = append(logger.sinks, w)
}

// Report a fatal error. Will set the level to "ERROR" on the wideEvent and will
// Log the error.
func Fatal(ctx context.Context, err error) {
	logger := getLogger(ctx)
	logger.wideEvent.fatal(err)
}

// Track an event.
// The returned object will be automatically logged when the context is cancelled
// as part of the wide event
func Track(ctx context.Context, eventName string) Event {
	logger := getLogger(ctx)
	e := newEvent(eventName)
	logger.wideEvent.add(e)
	return e
}

// Track an event.
// but will only actually include it in the wide event if the logger's level is
// below level.
// The returned object will be automatically logged when the context is cancelled
// as part of the wide event
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
	start := logger.wideEvent.time
	logger.wideEvent.duration = time.Since(start)
	b, err := json.Marshal(logger.wideEvent)
	if err != nil {
		panic(err)
	}
	for _, w := range logger.sinks {
		_, err = fmt.Fprintln(w, string(b))
		if err != nil {
			panic(err)
		}
	}
}
