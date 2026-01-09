package log

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type WideEvent map[string]any

func Track[T any](ctx context.Context, eventName string, event *T) func() {
	startTime := time.Now()

	return func() {
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
	minLevel := getWideEvent(ctx)["minLevel"].(int)
	if level >= minLevel {
		return Track(ctx, eventName, event)
	}
	return func() {}
}

func InitializeWideEvent(ctx context.Context, options *LoggerOptions) context.Context {
	v := ctx.Value("wideEvent")
	var event map[string]any
	if v == nil {
		event = make(map[string]any)
		ctx = context.WithValue(ctx, "wideEvent", event)
		event["time"] = time.Now()
		event["level"] = "INFO"
	} else {
		event = v.(map[string]any)
	}
	if options != nil {
		event["minLevel"] = options.Level
	} else {
		event["minLevel"] = LEVEL_INFO
	}
	return ctx
}

func getWideEvent(ctx context.Context) WideEvent {
	event, ok := ctx.Value("wideEvent").(map[string]any)
	if !ok {
		panic("Must call InitializeWideEvent first")
	}
	return event
}

func (event WideEvent) append(key string, value any) {
	current, vok := event[key]
	currentarr, arrok := event[key+"s"]
	if !vok && !arrok {
		event[key] = value
	} else {
		delete(event, key)
		arr, ok := currentarr.([]any)
		if !ok {
			event[key+"s"] = []any{current, value}
		} else {
			event[key+"s"] = append(arr, value)
		}
	}
}

func (event WideEvent) MarshalJSON() ([]byte, error) {
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
		fields = append(fields, kv{k: k, v: event[k]})
	}
	for k, v := range event {
		if !forced[k] {
			fields = append(fields, kv{k: k, v: v})
		}
	}
	for _, k := range forcedEnd {
		fields = append(fields, kv{k: k, v: event[k]})
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
