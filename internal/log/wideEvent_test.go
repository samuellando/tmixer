package log_test

import (
	"context"
	"testing"
	"time"

	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/testutil"
)

// Base case for the InitializeWideEvent function
func TestInitializeWideEvent(t *testing.T) {
	ctx := context.Background()
	ctx2 := log.InitializeWideEvent(ctx, nil)
	if ctx == ctx2 {
		t.Error("Should have initialized the context")
	}
}

// Recalling on an alredy initialied context should return the same context
func TestInitializeWideEventAlreadyInitialized(t *testing.T) {
	ctx := context.Background()
	ctx = log.InitializeWideEvent(ctx, nil)
	ctx2 := log.InitializeWideEvent(ctx, nil)
	if ctx != ctx2 {
		t.Error("Should have re-initialized the context")
	}
}

// InitializeWideEvent needs to be called on the context before any logging can occur
// If it is not, then the Track method needs to panic, this is a critical coding mistake
func TestTrackNotInitialized(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when Track is called on uninitialized context, but did not panic")
		}
	}()
	type TestEvent struct{}
	log.Track(context.Background(), "testEvent", &TestEvent{})()
}

// Base case for the Track method, test that the fields are transfered to the
// logs and that the time and durations asre added
func TestTrack(t *testing.T) {
	ctx := context.Background()
	ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

	type TestEvent struct {
		Value string
	}

	// Track first event
	event1 := &TestEvent{Value: "first"}
	finish := log.Track(ctx, "testEvent", event1)
	finish()

	res := testutil.GetLogEvent(ctx, logger, out)

	event, ok := res["testEvent"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'testEvent', got: %v", res)
	}

	// Verify first event
	if event["Value"] != "first" {
		t.Errorf("expected event Value='first', got '%v'", event["Value"])
	}
}

// Test that the tme and duration fields are valid
func TestTrackTimeDuration(t *testing.T) {
	ctx := context.Background()
	ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

	type TestEvent struct{}

	// Track first event
	event1 := &TestEvent{}
	finish1 := log.Track(ctx, "testEvent", event1)
	time.Sleep(time.Second)
	finish1()

	res := testutil.GetLogEvent(ctx, logger, out)

	event, ok := res["testEvent"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'testEvent', got: %v", res)
	}

	et, err := time.Parse(time.RFC3339, event["time"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(et) >= 2*time.Second || time.Since(et) <= 0 {
		t.Errorf("expected event Value='first', got '%v'", event["Value"])
	}

	duration := event["duration"].(float64)
	if time.Duration(duration) < time.Second {
		t.Errorf("duration should be ~ a secound")
	}
}

// Test that calling Track twice with the same event name converts singular to plural key
func TestTrackMultipleSameName(t *testing.T) {
	ctx := context.Background()
	ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

	type TestEvent struct {
		Value string
	}

	// Track first event
	event1 := &TestEvent{Value: "first"}
	finish1 := log.Track(ctx, "testEvent", event1)
	finish1()

	// Track second event with same name
	event2 := &TestEvent{Value: "second"}
	finish2 := log.Track(ctx, "testEvent", event2)
	finish2()

	res := testutil.GetLogEvent(ctx, logger, out)

	// Should have plural key "testEvents" with array of two events
	events, ok := res["testEvents"].([]any)
	if !ok {
		t.Fatalf("expected 'testEvents' array, got: %v", res)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// Verify first event
	first, ok := events[0].(map[string]any)
	if !ok {
		t.Fatal("first event should be a map")
	}
	if first["Value"] != "first" {
		t.Errorf("expected first event Value='first', got '%v'", first["Value"])
	}

	// Verify second event
	second, ok := events[1].(map[string]any)
	if !ok {
		t.Fatal("second event should be a map")
	}
	if second["Value"] != "second" {
		t.Errorf("expected second event Value='second', got '%v'", second["Value"])
	}
}

// Test that nested struct fields are properly marshaled
func TestTrackNestedStruct(t *testing.T) {
	ctx := context.Background()
	ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

	type Inner struct {
		Name  string
		Count int
	}
	type Outer struct {
		ID     int
		Nested Inner
		Items  []string
	}

	event := &Outer{
		ID: 42,
		Nested: Inner{
			Name:  "test",
			Count: 5,
		},
		Items: []string{"a", "b", "c"},
	}
	finish := log.Track(ctx, "nestedEvent", event)
	finish()

	res := testutil.GetLogEvent(ctx, logger, out)

	logged, ok := res["nestedEvent"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'nestedEvent' map, got: %v", res)
	}

	// Check top-level field
	if logged["ID"] != float64(42) {
		t.Errorf("expected ID=42, got %v", logged["ID"])
	}

	// Check nested struct
	nested, ok := logged["Nested"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'Nested' to be a map, got: %T", logged["Nested"])
	}
	if nested["Name"] != "test" {
		t.Errorf("expected Nested.Name='test', got '%v'", nested["Name"])
	}
	if nested["Count"] != float64(5) {
		t.Errorf("expected Nested.Count=5, got %v", nested["Count"])
	}

	// Check slice field
	items, ok := logged["Items"].([]any)
	if !ok {
		t.Fatalf("expected 'Items' to be a slice, got: %T", logged["Items"])
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

// Test that an empty struct still gets time and duration fields added
func TestTrackEmptyStruct(t *testing.T) {
	ctx := context.Background()
	ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

	type EmptyEvent struct{}

	event := &EmptyEvent{}
	finish := log.Track(ctx, "emptyEvent", event)
	finish()

	res := testutil.GetLogEvent(ctx, logger, out)

	logged, ok := res["emptyEvent"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'emptyEvent' map, got: %v", res)
	}

	// Should have time field
	if _, ok := logged["time"]; !ok {
		t.Error("empty struct event should have 'time' field")
	}

	// Should have duration field
	if _, ok := logged["duration"]; !ok {
		t.Error("empty struct event should have 'duration' field")
	}
}

// Test that noting is done until the commit function is called
func TestTrackNoFinish(t *testing.T) {
	ctx := context.Background()
	ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

	type TestEvent struct {
		Value string
	}

	// Track event but don't call finish
	event := &TestEvent{Value: "shouldNotLog"}
	log.Track(ctx, "testEvent", event)

	res := testutil.GetLogEvent(ctx, logger, out)

	// Should not have the event since finish was not called
	if _, ok := res["testEvent"]; ok {
		t.Error("event should not be logged since finish was not called")
	}
}

// Logging at or above the level should include it
func TestTrackLevelIncludes(t *testing.T) {
	ctx := context.Background()
	ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_INFO)

	type TestEvent struct {
		Value string
	}

	event := &TestEvent{Value: "included"}
	finish := log.TrackLevel(log.LEVEL_INFO, ctx, "testEvent", event)
	finish()

	res := testutil.GetLogEvent(ctx, logger, out)

	if _, ok := res["testEvent"]; !ok {
		t.Error("event should be included since level matches min level")
	}
}

// Logging below the level should exclude it
func TestTrackLevelExcludes(t *testing.T) {
	ctx := context.Background()
	ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_INFO)

	type TestEvent struct {
		Value string
	}

	event := &TestEvent{Value: "excluded"}
	finish := log.TrackLevel(log.LEVEL_DEBUG, ctx, "testEvent", event)
	finish()

	res := testutil.GetLogEvent(ctx, logger, out)

	if _, ok := res["testEvent"]; ok {
		t.Error("event should be excluded since level is below min level")
	}
}
