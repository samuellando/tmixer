package log

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type WideEvent struct {
	mu   *sync.Mutex
	data map[string]any
}

func Track[T any](ctx context.Context, eventName string, event *T) func() {
	startTime := time.Now()

	return func() {
		// Convert to a generic map[string]any so we can add additonal fields.
		eventMap := make(map[string]any)
		b, err := json.Marshal(event)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(b, &eventMap)
		if err != nil {
			panic(err)
		}
		// Add the time fields
		eventMap["time"] = startTime
		eventMap["duration"] = time.Since(startTime)
		// Add the map to the context
		getWideEvent(ctx).append(eventName, eventMap)
	}
}

func TrackLevel[T any](level int, ctx context.Context, eventName string, event *T) func() {
	minLevel := getWideEvent(ctx).data["minLevel"].(int)
	if level >= minLevel {
		return Track(ctx, eventName, event)
	}
	return func() {}
}

func InitializeWideEvent(ctx context.Context, options *LoggerOptions) context.Context {
	v := ctx.Value("wideEvent")
	var event WideEvent
	if v == nil {
		event = WideEvent{
			mu:   &sync.Mutex{},
			data: make(map[string]any),
		}
		ctx = context.WithValue(ctx, "wideEvent", event)
		event.data["time"] = time.Now()
		event.data["level"] = "INFO"
	} else {
		event = v.(WideEvent)
	}
	// Update the options
	event.mu.Lock()
	defer event.mu.Unlock()
	if options != nil {
		event.data["minLevel"] = options.Level
	} else {
		event.data["minLevel"] = LEVEL_INFO
	}
	return ctx
}

func getWideEvent(ctx context.Context) WideEvent {
	event, ok := ctx.Value("wideEvent").(WideEvent)
	if !ok {
		panic("Must call InitializeWideEvent first")
	}
	return event
}

func (event WideEvent) append(key string, value any) {
	event.mu.Lock()
	defer event.mu.Unlock()
	current, vok := event.data[key]
	currentarr, arrok := event.data[key+"s"]
	if !vok && !arrok {
		event.data[key] = value
	} else {
		delete(event.data, key)
		arr, ok := currentarr.([]any)
		if !ok {
			event.data[key+"s"] = []any{current, value}
		} else {
			event.data[key+"s"] = append(arr, value)
		}
	}
}

func (event WideEvent) MarshalJSON() ([]byte, error) {
	event.mu.Lock()
	defer event.mu.Unlock()
	forcedBegining := []string{"level", "time", "error", "minLevel"}
	forcedEnd := []string{"duration"}
	forced := make(map[string]bool)
	for _, k := range forcedBegining {
		forced[k] = true
	}
	for _, k := range forcedEnd {
		forced[k] = true
	}

	type kv struct {
		k string
		v any
	}

	var fields []kv
	for _, k := range forcedBegining {
		fields = append(fields, kv{k: k, v: event.data[k]})
	}
	for k, v := range event.data {
		if !forced[k] {
			fields = append(fields, kv{k: k, v: v})
		}
	}
	for _, k := range forcedEnd {
		fields = append(fields, kv{k: k, v: event.data[k]})
	}

	out := bytes.Buffer{}
	out.WriteByte('{')
	for i, field := range fields {
		b, err := json.Marshal(field.v)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&out, `"%s":%s`, field.k, string(b))
		if i < len(fields)-1 {
			out.WriteByte(',')
		}
	}
	out.WriteByte('}')
	return out.Bytes(), nil
}
