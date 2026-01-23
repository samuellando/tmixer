package log_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"samuellando.com/tmixer/internal/log"
)

// Test that New creates a logger and initializes wide event
func TestLoggerNew(t *testing.T) {
	ctx := context.Background()
	ctx2, logger := log.New(ctx, &log.LoggerOptions{Level: log.LEVEL_INFO})

	if logger == nil {
		t.Fatal("logger should not be nil")
	}
	if ctx == ctx2 {
		t.Error("context should be modified")
	}

	type TestEvent struct{}
	log.Track(ctx2, "initTest", &TestEvent{}) // This should not panic
}

// Test AddSink adds a writer to the logger
func TestLoggerAddSink(t *testing.T) {
	ctx, logger := log.New(context.Background(), nil)

	var buf bytes.Buffer
	logger.AddSink(&buf)

	// Add event and check output
	type TestEvent struct{ Msg string }
	finish := log.Track(ctx, "test", &TestEvent{Msg: "added"})
	finish()

	logger.Info(ctx)

	output := buf.String()
	if output == "" {
		t.Error("expected output after adding sink")
	}
}

// Test Info logs the wide event to sinks
func TestLoggerInfo(t *testing.T) {
	ctx, logger := log.New(context.Background(), nil)
	var buf bytes.Buffer
	logger.AddSink(&buf)

	// Add some event
	type TestEvent struct{ Msg string }
	finish := log.Track(ctx, "test", &TestEvent{Msg: "hello"})
	finish()

	logger.Info(ctx)

	output := buf.String()
	if output == "" {
		t.Error("expected output, got empty")
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}
	if _, ok := result["test"]; !ok {
		t.Error("expected 'test' event in output")
	}
	if result["level"] != "INFO" {
		t.Errorf("expected level 'INFO', got %v", result["level"])
	}
}

// Test Error sets level to ERROR and adds error field
func TestLoggerError(t *testing.T) {
	ctx, logger := log.New(context.Background(), nil)
	var buf bytes.Buffer
	logger.AddSink(&buf)

	testErr := fmt.Errorf("test error")
	logger.Error(ctx, testErr)

	output := buf.String()
	var result map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}
	if result["level"] != "ERROR" {
		t.Errorf("expected level 'ERROR', got %v", result["level"])
	}
	if result["error"] != "test error" {
		t.Errorf("expected error 'test error', got %v", result["error"])
	}
}

// Test that Info writes to multiple sinks
func TestLoggerMultipleSinks(t *testing.T) {
	ctx, logger := log.New(context.Background(), nil)
	var buf1, buf2 bytes.Buffer
	logger.AddSink(&buf1)
	logger.AddSink(&buf2)

	type TestEvent struct{ Msg string }
	finish := log.Track(ctx, "test", &TestEvent{Msg: "multi"})
	finish()

	logger.Info(ctx)

	out1 := buf1.String()
	out2 := buf2.String()
	if out1 == "" || out2 == "" {
		t.Error("both sinks should have output")
	}
	if out1 != out2 {
		t.Error("both sinks should have identical output")
	}
}
