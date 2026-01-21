package config_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
)

func setupLogging(ctx context.Context, level int) (context.Context, *log.Logger, *bytes.Buffer) {
	ctx, logger := log.New(ctx, &log.LoggerOptions{Level: level})
	out := &bytes.Buffer{}
	logger.AddSink(out)
	return ctx, logger, out
}

func getLogEvent(ctx context.Context, logger *log.Logger, out *bytes.Buffer) map[string]any {
	logger.Info(ctx)
	res := make(map[string]any)
	json.Unmarshal(out.Bytes(), &res)
	return res
}

func TestConfigLoadEventFields(t *testing.T) {
	ctx := context.Background()
	ctx, logger, out := setupLogging(ctx, log.LEVEL_DEBUG)

	// Create a valid config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yml")
	configContent := `
defaultProject: test
projects:
  test:
    directory: /tmp/test
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.New()
	cfg.ConfigFiles = []string{configFile}
	cfg.LoadFiles(ctx)

	res := getLogEvent(ctx, logger, out)

	// Check configLoadEvent exists
	event, ok := res["configLoadEvent"].(map[string]any)
	if !ok {
		t.Fatal("configLoadEvent not found in log output")
	}

	// Check result field exists
	if _, ok := event["result"]; !ok {
		t.Error("configLoadEvent missing 'result' field")
	}

	// Verify result is an object (Config struct)
	if _, ok := event["result"].(map[string]any); !ok {
		t.Error("'result' field should be an object")
	}
}

func TestConfigLoadEventNoErrors(t *testing.T) {
	ctx := context.Background()
	ctx, logger, out := setupLogging(ctx, log.LEVEL_DEBUG)

	// Create a valid config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yml")
	configContent := `
defaultProject: test
projects:
  test:
    directory: /tmp/test
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.New()
	cfg.ConfigFiles = []string{configFile}
	cfg.LoadFiles(ctx)

	res := getLogEvent(ctx, logger, out)
	event := res["configLoadEvent"].(map[string]any)

	// errors field should be omitted when there are no errors
	if _, ok := event["errors"]; ok {
		t.Error("'errors' field should be omitted when there are no errors")
	}
}

func TestConfigLoadEventWithSingleError(t *testing.T) {
	ctx := context.Background()
	ctx, logger, out := setupLogging(ctx, log.LEVEL_DEBUG)

	// Use a non-existent file to trigger an error
	cfg := config.New()
	cfg.ConfigFiles = []string{"/nonexistent/path/config.yml"}
	cfg.LoadFiles(ctx)

	res := getLogEvent(ctx, logger, out)
	event := res["configLoadEvent"].(map[string]any)

	// errors field should be present
	errors, ok := event["errors"]
	if !ok {
		t.Fatal("'errors' field should be present when errors occur")
	}

	// Verify errors is an array
	errorsList, ok := errors.([]any)
	if !ok {
		t.Fatalf("'errors' field should be an array, got: %T", errors)
	}

	// Should have exactly 1 error
	if len(errorsList) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errorsList))
	}

	// Each error should be a string
	if _, ok := errorsList[0].(string); !ok {
		t.Error("Error should be a string")
	}
}

func TestConfigLoadEventWithMultipleErrors(t *testing.T) {
	ctx := context.Background()
	ctx, logger, out := setupLogging(ctx, log.LEVEL_DEBUG)

	// Create multiple scenarios that will cause errors:
	// 1. Non-existent file
	// 2. Another non-existent file
	// 3. Invalid YAML file
	tmpDir := t.TempDir()
	invalidYamlFile := filepath.Join(tmpDir, "invalid.yml")
	invalidContent := `
defaultProject: test
projects:
  - this is invalid yaml structure
	indentation is broken
`
	err := os.WriteFile(invalidYamlFile, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.New()
	cfg.ConfigFiles = []string{
		"/nonexistent/path1/config.yml",
		"/nonexistent/path2/config.yml",
		invalidYamlFile,
	}
	cfg.LoadFiles(ctx)

	res := getLogEvent(ctx, logger, out)
	event := res["configLoadEvent"].(map[string]any)

	// errors field should be present
	errors, ok := event["errors"]
	if !ok {
		t.Fatal("'errors' field should be present when errors occur")
	}

	// Verify errors is an array
	errorsList, ok := errors.([]any)
	if !ok {
		t.Fatalf("'errors' field should be an array, got: %T", errors)
	}

	// Should have 3 errors
	if len(errorsList) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(errorsList))
	}

	// Each error should be a string
	for i, err := range errorsList {
		if _, ok := err.(string); !ok {
			t.Errorf("Error at index %d should be a string, got: %T", i, err)
		}
	}
}

func TestConfigLoadEventResultPopulatedWithErrors(t *testing.T) {
	ctx := context.Background()
	ctx, logger, out := setupLogging(ctx, log.LEVEL_DEBUG)

	// Use a non-existent file to trigger an error
	cfg := config.New()
	cfg.ConfigFiles = []string{"/nonexistent/path/config.yml"}
	cfg.LoadFiles(ctx)

	res := getLogEvent(ctx, logger, out)
	event := res["configLoadEvent"].(map[string]any)

	// result field should still exist even with errors
	result, ok := event["result"]
	if !ok {
		t.Fatal("'result' field should exist even when errors occur")
	}

	// Verify result is an object (Config struct)
	resultObj, ok := result.(map[string]any)
	if !ok {
		t.Fatal("'result' field should be an object")
	}

	// Verify result has some expected Config fields (basic sanity check)
	if _, ok := resultObj["Projects"]; !ok {
		t.Error("result Config should have 'Projects' field")
	}
}
