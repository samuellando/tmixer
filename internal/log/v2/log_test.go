package log_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"

	"samuellando.com/tmixer/internal/log/v2"
	"samuellando.com/tmixer/internal/testutil"
)

type StringWriteCloser struct {
	b strings.Builder
}

func (s *StringWriteCloser) Write(p []byte) (int, error) {
	return s.b.Write(p)
}

func (s *StringWriteCloser) Close() error {
	return nil
}

func (s *StringWriteCloser) String() string {
	return s.b.String()
}

var tsRegex = regexp.MustCompile(
	`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+(?:Z|[+-]\d{2}:\d{2})`,
)

var durationRegex = regexp.MustCompile(
	`"duration"\s*:\s*\d+`,
)

func normalize(s string) string {
	s = tsRegex.ReplaceAllString(s, "TIMESTAMP")
	s = durationRegex.ReplaceAllString(s, `"duration":0`)
	return s
}

func TestLogger(t *testing.T) {
	ctx := context.Background()
	sink := &StringWriteCloser{}
	ctx = log.ContextLogger(ctx, log.LogOptions{
		Level: log.LEVEL_INFO,
		Sinks: []io.WriteCloser{sink},
	})
	logEvent := log.Track(ctx, "test")
	logEvent.Log("hello", "World")
	logEvent.Log("abc", []int{1, 2, 3})
	logEvent = log.Track(ctx, "test2")
	logEvent.Log("hello", "World")
	logEvent.Log("def", []int{4, 5, 6})
	log.Done(ctx)
	testutil.GoldenTest(t, normalize(sink.String()))
}

func TestTrackLevel(t *testing.T) {
	ctx := context.Background()
	sink := &StringWriteCloser{}
	ctx = log.ContextLogger(ctx, log.LogOptions{
		Level: log.LEVEL_INFO,
		Sinks: []io.WriteCloser{sink},
	})
	log.Track(ctx, "test")
	log.TrackLevel(ctx, log.LEVEL_DEBUG, "test2")
	log.Done(ctx)

	out := make(map[string]any)
	err := json.Unmarshal([]byte(sink.String()), &out)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["test2"]; ok {
		t.Fatal("Should not have debug level log")
	}
}

func TestSetOptions(t *testing.T) {
	ctx := context.Background()
	sink := &StringWriteCloser{}
	ctx = log.ContextLogger(ctx, log.LogOptions{
		Level: log.LEVEL_INFO,
		Sinks: []io.WriteCloser{sink},
	})
	log.Track(ctx, "test")
	log.TrackLevel(ctx, log.LEVEL_DEBUG, "test2")
	log.SetOptions(ctx, log.LogOptions{
		Level: log.LEVEL_DEBUG,
		Sinks: []io.WriteCloser{sink},
	})
	log.TrackLevel(ctx, log.LEVEL_DEBUG, "test3")
	log.Done(ctx)

	out := make(map[string]any)
	err := json.Unmarshal([]byte(sink.String()), &out)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["test2"]; ok {
		t.Fatal("Should not have first debug level log")
	}
	if _, ok := out["test3"]; !ok {
		t.Fatal("Should have second debug level log")
	}
}

func TestAddSinks(t *testing.T) {
	ctx := context.Background()
	sink := &StringWriteCloser{}
	sink2 := &StringWriteCloser{}
	ctx = log.ContextLogger(ctx, log.LogOptions{
		Level: log.LEVEL_INFO,
	})
	log.AddSink(ctx, sink)
	log.AddSink(ctx, sink2)
	log.Track(ctx, "test")
	log.Done(ctx)
	testutil.GoldenTest(t, normalize(sink.String()))
	testutil.GoldenTest(t, normalize(sink2.String()))
}

func TestFatal(t *testing.T) {
	ctx := context.Background()
	sink := &StringWriteCloser{}
	ctx = log.ContextLogger(ctx, log.LogOptions{
		Level: log.LEVEL_INFO,
		Sinks: []io.WriteCloser{sink},
	})
	log.Fatal(ctx, fmt.Errorf("This is an error"))
	testutil.GoldenTest(t, normalize(sink.String()))
}

func TestTimestampAndDuration(t *testing.T) {
	ctx := context.Background()
	sink := &StringWriteCloser{}
	ctx = log.ContextLogger(ctx, log.LogOptions{
		Level: log.LEVEL_INFO,
		Sinks: []io.WriteCloser{sink},
	})
	time.Sleep(time.Second)
	log.Done(ctx)

	out := make(map[string]any)
	err := json.Unmarshal([]byte(sink.String()), &out)
	if err != nil {
		t.Fatal(err)
	}
	timeValue, err := time.Parse(time.RFC3339Nano, out["time"].(string))
	if err != nil {
		t.Fatal(err)
	}
	durationValue := time.Duration(out["duration"].(float64))
	if time.Since(timeValue) < time.Second || time.Since(timeValue) > 2*time.Second {
		t.Error("Timestamp should be one sec ago")
	}
	if durationValue < time.Second || durationValue > 2*time.Second {
		t.Error("Timestamp should be one sec ago")
	}
}

func TestLogEventErrors(t *testing.T) {
	ctx := context.Background()
	sink := &StringWriteCloser{}
	ctx = log.ContextLogger(ctx, log.LogOptions{
		Level: log.LEVEL_INFO,
		Sinks: []io.WriteCloser{sink},
	})
	logEvent := log.Track(ctx, "test")
	logEvent.Log("hello", "World")
	logEvent.Log("abc", []int{1, 2, 3})
	logEvent.Error(fmt.Errorf("Error1"))
	logEvent.Error(fmt.Errorf("Error2"))
	log.Done(ctx)
	testutil.GoldenTest(t, normalize(sink.String()))
}

func TestLogEventTimestampAndDuration(t *testing.T) {
	ctx := context.Background()
	sink := &StringWriteCloser{}
	ctx = log.ContextLogger(ctx, log.LogOptions{
		Level: log.LEVEL_INFO,
		Sinks: []io.WriteCloser{sink},
	})
	logEvent := log.Track(ctx, "test")
	time.Sleep(time.Second)
	logEvent.Done()
	time.Sleep(time.Second)
	log.Done(ctx)

	out := make(map[string]any)
	err := json.Unmarshal([]byte(sink.String()), &out)
	if err != nil {
		t.Fatal(err)
	}
	out = out["test"].(map[string]any)
	timeValue, err := time.Parse(time.RFC3339Nano, out["time"].(string))
	if err != nil {
		t.Fatal(err)
	}
	durationValue := time.Duration(out["duration"].(float64))
	if time.Since(timeValue) < 2*time.Second || time.Since(timeValue) > 3*time.Second {
		t.Error("Timestamp should be one sec ago")
	}
	if durationValue < time.Second || durationValue > 2*time.Second {
		t.Error("Timestamp should be one sec ago")
	}
}

func TestDisplay(t *testing.T) {
	testutil.GoldenTest(t, normalize(testutil.CaptureStderr(func() {
		ctx := context.Background()
		sink := &StringWriteCloser{}
		ctx = log.ContextLogger(ctx, log.LogOptions{
			Level: log.LEVEL_INFO,
			Sinks: []io.WriteCloser{sink},
		})
		logEvent := log.Track(ctx, "test")
		logEvent.Log("hello", "World")
		logEvent.Log("abc", []int{1, 2, 3})
		logEvent = log.Track(ctx, "test2")
		logEvent.Log("hello", "World")
		logEvent.Log("def", []int{4, 5, 6})
		log.Done(ctx)
		log.Display(ctx)
	})))
}
