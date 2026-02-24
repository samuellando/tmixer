package flags

import (
	"context"
	"os"
	"testing"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
)

func TestParseArgs(t *testing.T) {
	ctx, _ := log.New(context.Background(), nil)
	counter := 0
	flags := map[string]Flag{
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
			Default: "ddd",
			ParseInput: func(s string, c *config.Config) error {
				if s != "ddd" {
					t.Fatal("incorrect input")
				}
				counter += 1
				return nil
			},
		},
		"five": {
			EnvironmentVariable: "TMIXER_TEST_FIVE",
			ParseInput: func(s string, c *config.Config) error {
				if s != "env_var" {
					t.Fatal("incorrect input")
				}
				counter += 1
				return nil
			},
		},
	}
	args := []string{"--four", "--one", "aaa", "-t", "--three", "bbb", "hello"}
	err := os.Setenv("TMIXER_TEST_FIVE", "env_var")
	if err != nil {
		t.Fatal(err)
	}
	remaining, err := ParseArgs(ctx, args, flags, nil)
	if err != nil {
		t.Fatal(err)
	}
	if counter != 5 {
		t.Fatal("Should've parsed 5 flags")
	}
	if len(remaining) != 1 || remaining[0] != "hello" {
		t.Fatal("Should return remaining flags")
	}
}

func TestHelpMessage(t *testing.T) {
	flags := map[string]Flag{
		"one": {
			ShortName: "o",
		},
		"two": {
			ShortName:   "t",
			Description: "what it is",
			Usage:       "How to use it",
			Default:     "aaa",
		},
		"three": {},
		"four": {
			Default: "ddd",
		},
	}
	res := HelpMessage(flags)
	expected := `--four
	Default: ddd

--one OR -o

--three

--two OR -t
	what it is
	Usage: How to use it
	Default: aaa

`
	if res != expected {
		t.Fatalf("%s\n!=\n%s", res, expected)
	}
}
