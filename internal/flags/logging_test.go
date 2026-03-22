package flags_test

import (
	"context"
	"errors"
	"testing"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/flags"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/testutil"
)

func TestFlagParseEventFields(t *testing.T) {
	ctx := context.Background()
	ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

	testFlags := map[string]flags.Flag{
		"test": {
			ShortName: "t",
			ParseInput: func(s string, c *config.Config) error {
				return nil
			},
		},
	}

	args := []string{"--test", "value", "remaining"}
	_, err := flags.ParseArgs(ctx, args, testFlags, &config.Config{})
	if err != nil {
		t.Fatal(err)
	}

	res := testutil.GetLogEvent(ctx, logger, out)

	// Check flagParseEvent exists
	event, ok := res["flagParseEvent"].(map[string]any)
	if !ok {
		t.Fatal("flagParseEvent not found in log output")
	}

	// Check inputArgs field exists and is an array
	inputArgs, ok := event["inputArgs"]
	if !ok {
		t.Error("flagParseEvent missing 'inputArgs' field")
	}
	if args, ok := inputArgs.([]any); !ok {
		t.Error("'inputArgs' field should be an array")
	} else {
		if args[0] != "--test" {
			t.Error("'inputArgs' field should be --test")
		}
		if args[1] != "value" {
			t.Error("'inputArgs' field should be value")
		}
		if args[2] != "remaining" {
			t.Error("'inputArgs' field should be remaining")
		}
	}

	// Check parsedFlags field exists and is an array
	parsedFlags, ok := event["parsedFlags"]
	if !ok {
		t.Error("flagParseEvent missing 'parsedFlags' field")
	}
	if flags, ok := parsedFlags.([]any); !ok {
		t.Error("'parsedFlags' field should be an array")
	} else {
		if len(flags) != 1 {
			t.Fatal("Should contain parsed flags")
		}
		flag := flags[0].([]any)
		if flag[0].(string) != "test" || flag[1].(string) != "value" {
			t.Error("'parsedFlags' field should contain values")
		}
	}

	// Check remainingArgs field exists and is an array
	remainingArgs, ok := event["remainingArgs"]
	if !ok {
		t.Error("flagParseEvent missing 'remainingArgs' field")
	}
	if remaining, ok := remainingArgs.([]any); !ok {
		t.Error("'remainingArgs' field should be an array")
	} else {
		if len(remaining) != 1 {
			t.Error("Should have one remaining value")
		}
		if remaining[0] != "remaining" {
			t.Error("Remaining value should be remaining")
		}
	}

	// Check result field exists
	if result, ok := event["result"]; !ok {
		t.Error("flagParseEvent missing 'result' field")
	} else {
		conf := result.(map[string]any)
		if _, ok := conf["Projects"]; !ok {
			t.Error("the result should not be something")
		}
	}

	// errors field should be omitted when there are no errors
	if _, ok = event["errors"]; ok {
		t.Error("'errors' field should be empty when there are no errors")
	}
}

func TestFlagParseEventWithError(t *testing.T) {
	ctx := context.Background()
	ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

	testFlags := map[string]flags.Flag{
		"failing": {
			ShortName: "f",
			ParseInput: func(s string, c *config.Config) error {
				return errors.New("parse failed")
			},
		},
	}

	args := []string{"tmixer", "--failing", "value"}
	_, err := flags.ParseArgs(ctx, args, testFlags, &config.Config{})
	if err == nil {
		t.Fatal("Should return an error")
	}

	res := testutil.GetLogEvent(ctx, logger, out)
	event := res["flagParseEvent"].(map[string]any)

	// errors field should be present
	errorsField, ok := event["errors"]
	if !ok {
		t.Fatal("'errors' field should be present when errors occur")
	}

	// Verify errors is an array
	errorsList, ok := errorsField.([]any)
	if !ok {
		t.Fatalf("'errors' field should be an array, got: %T", errorsField)
	}

	// Should have exactly 1 error
	if len(errorsList) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errorsList))
	}

	// Error should be a string
	errStr, ok := errorsList[0].(string)
	if !ok {
		t.Error("Error should be a string")
	}

	// Error should contain context about which flag failed
	if errStr == "" {
		t.Error("Error string should not be empty")
	}
}
