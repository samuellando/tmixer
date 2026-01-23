package log

import (
	"encoding/json"
	"sync"
	"testing"
)

// Test that forced fields appear in correct order: level, time, error, minLevel first; duration last
func TestMarshalJSONFieldOrdering(t *testing.T) {
	event := WideEvent{
		mu: &sync.Mutex{},
		data: map[string]any{
			"level":    "INFO",
			"time":     "2024-01-01T00:00:00Z",
			"error":    "some error",
			"minLevel": 0,
			"duration": 100,
			"custom1":  "value1",
			"custom2":  "value2",
		},
	}

	b, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(b)

	// Check that forced beginning fields appear before custom fields
	levelIdx := indexOf(jsonStr, `"level"`)
	timeIdx := indexOf(jsonStr, `"time"`)
	errorIdx := indexOf(jsonStr, `"error"`)
	minLevelIdx := indexOf(jsonStr, `"minLevel"`)
	custom1Idx := indexOf(jsonStr, `"custom1"`)
	custom2Idx := indexOf(jsonStr, `"custom2"`)
	durationIdx := indexOf(jsonStr, `"duration"`)

	// Forced beginning order: level, time, error, minLevel
	if levelIdx > timeIdx {
		t.Error("'level' should appear before 'time'")
	}
	if timeIdx > errorIdx {
		t.Error("'time' should appear before 'error'")
	}
	if errorIdx > minLevelIdx {
		t.Error("'error' should appear before 'minLevel'")
	}

	// Forced beginning should be before custom fields
	if minLevelIdx > custom1Idx || minLevelIdx > custom2Idx {
		t.Error("forced beginning fields should appear before custom fields")
	}

	// Custom fields should appear before duration
	if custom1Idx > durationIdx || custom2Idx > durationIdx {
		t.Error("custom fields should appear before 'duration'")
	}
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Test marshaling an empty/minimal wide event
func TestMarshalJSONEmptyEvent(t *testing.T) {
	event := WideEvent{
		mu:   &sync.Mutex{},
		data: make(map[string]any),
	}

	b, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal empty event: %v", err)
	}

	// Should produce valid JSON with nil values for forced fields
	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// Empty event should still have the forced field keys (with nil values)
	expectedKeys := []string{"level", "time", "error", "minLevel", "duration"}
	for _, key := range expectedKeys {
		if _, exists := result[key]; !exists {
			t.Errorf("expected key '%s' to exist in marshaled output", key)
		}
	}
}

// Test that nil values in the event map are handled correctly
func TestMarshalJSONNilValues(t *testing.T) {
	event := WideEvent{
		mu: &sync.Mutex{},
		data: map[string]any{
			"level":     "INFO",
			"time":      nil,
			"error":     nil,
			"minLevel":  0,
			"duration":  nil,
			"customNil": nil,
			"customVal": "present",
		},
	}

	b, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event with nil values: %v", err)
	}

	// Should produce valid JSON
	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// Nil values should be preserved as null in JSON
	if result["time"] != nil {
		t.Errorf("expected 'time' to be nil, got %v", result["time"])
	}
	if result["customNil"] != nil {
		t.Errorf("expected 'customNil' to be nil, got %v", result["customNil"])
	}

	// Non-nil values should be preserved
	if result["customVal"] != "present" {
		t.Errorf("expected 'customVal' to be 'present', got %v", result["customVal"])
	}
}
