package log

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type WideEvent map[string]any

func (event WideEvent) MarshalJSON() ([]byte, error) {
	forcedBegining := []string{"level", "time", "error"}
	forcedEnd := []string{"duration"}
	forced := make(map[string]bool)
	for _, k := range forcedBegining {
		forced[k] = true
	}
	for _, k := range forcedEnd {
		forced[k] = true
	}
	out := bytes.Buffer{}
	i := 0
	writeField := func(k string, v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		fmt.Fprintf(&out, `"%s":%s`, k, string(b))
		i++
		if i < len(event) {
			out.WriteByte(',')
		}
		return nil
	}
	out.WriteByte('{')
	for _, k := range forcedBegining {
		err := writeField(k, event[k])
		if err != nil {
			return nil, err
		}
	}
	for k, v := range event {
		if !forced[k] {
			err := writeField(k, v)
			if err != nil {
				return nil, err
			}
		}
	}
	for _, k := range forcedEnd {
		err := writeField(k, event[k])
		if err != nil {
			return nil, err
		}
	}
	out.WriteByte('}')
	return out.Bytes(), nil
}

func InitializeWideEvent(ctx context.Context) context.Context {
	event := make(map[string]any)
	event["time"] = time.Now()
	event["level"] = "INFO"
	return context.WithValue(ctx, "wideEvent", event)
}

func GetWideEvent(ctx context.Context) WideEvent {
	event, ok := ctx.Value("wideEvent").(map[string]any)
	if !ok {
		panic("Must call InitializeWideEvent first")
	}
	return event
}

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
