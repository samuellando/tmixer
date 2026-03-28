package flags_test

import (
	"context"
	"os"
	"testing"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/flags"
	"samuellando.com/tmixer/internal/log/v2"
)

func TestParseArgs(t *testing.T) {
	ctx := log.ContextLogger(context.Background())
	counter := 0
	args := []string{"--four", "--one", "aaa", "-t", "--three", "bbb", "hello"}
	inFlags := map[string]flags.Flag{
		"one": {
			ShortName: "o",
			ParseInput: func(s string, c *config.Config) error {
				if s != "aaa" {
					t.Fatal("incorrect input")
				}
				counter += 1
				return nil
			},
		},
		"two": {
			ShortName: "t",
			Bare:      true,
			ParseInput: func(s string, c *config.Config) error {
				if s != "" {
					t.Fatal("incorrect input")
				}
				counter += 1
				return nil
			},
		},
		"three": {
			ParseInput: func(s string, c *config.Config) error {
				if s != "bbb" {
					t.Fatal("incorrect input")
				}
				counter += 1
				return nil
			},
		},
		"four": {
			Bare: true,
			ParseInput: func(s string, c *config.Config) error {
				if s != "" {
					t.Fatal("incorrect input")
				}
				counter += 1
				return nil
			},
		},
		"five": {
			EnvironmentVariable: "TMIXER_TEST_FIVE",
			Default:             "eee",
			ParseInput: func(s string, c *config.Config) error {
				if s != "env_var" {
					t.Fatal("incorrect input")
				}
				counter += 1
				return nil
			},
		},
		"six": {
			EnvironmentVariable: "TMIXER_TEST_SIX",
			Default:             "fff",
			ParseInput: func(s string, c *config.Config) error {
				if s != "fff" {
					t.Fatal("incorrect input")
				}
				counter += 1
				return nil
			},
		},
	}
	err := os.Setenv("TMIXER_TEST_FIVE", "env_var")
	if err != nil {
		t.Fatal(err)
	}
	_, remaining, err := flags.ParseArgs(ctx, args, inFlags)
	if err != nil {
		t.Fatal(err)
	}
	if counter != 6 {
		t.Fatalf("Should've parsed 6 flags got %d", counter)
	}
	if len(remaining) != 1 || remaining[0] != "hello" {
		t.Fatal("Should return remaining flags")
	}
}
