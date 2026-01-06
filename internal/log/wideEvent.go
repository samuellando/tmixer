package log

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type WideEvent map[string]any

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
		GetWideEvent(ctx)[eventName] = eventMap
	}
}

func Lock[T any](v *T) *T {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	out := new(T)
	err = json.Unmarshal(b, out)
	if err != nil {
		panic(err)
	}
	return out
}

func LockArray[T any](v []T) []T {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	out := make([]T, 0)
	err = json.Unmarshal(b, &out)
	if err != nil {
		panic(err)
	}
	return out
}

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
