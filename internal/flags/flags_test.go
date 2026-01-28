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
	flags := []Flag{
		{
			Name:      "one",
			ShortName: "o",
			ParseInput: func(s string, c *config.Config) error {
				if s != "aaa" {
					t.Fatal("incorrect input")
				}
				counter += 1
				return nil
			},
		},
		{
			Name:      "two",
			ShortName: "t",
			ParseInput: func(s string, c *config.Config) error {
				if s != "" {
					t.Fatal("incorrect input")
				}
				counter += 1
				return nil
			},
		},
		{
			Name: "three",
			ParseInput: func(s string, c *config.Config) error {
				if s != "bbb" {
					t.Fatal("incorrect input")
				}
				counter += 1
				return nil
			},
		},
		{
			Name:    "four",
			Default: "ddd",
			ParseInput: func(s string, c *config.Config) error {
				if s != "ddd" {
					t.Fatal("incorrect input")
				}
				counter += 1
				return nil
			},
		},
		{
			Name:                "five",
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
	args := []string{"tmixer", "--four", "--one", "aaa", "-t", "--three", "bbb", "hello"}
	os.Setenv("TMIXER_TEST_FIVE", "env_var")
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
	flags := []Flag{
		{
			Name:      "one",
			ShortName: "o",
		},
		{
			Name:        "two",
			ShortName:   "t",
			Description: "what it is",
			Usage:       "How to use it",
			Default:     "aaa",
		},
		{
			Name: "three",
		},
		{
			Name:    "four",
			Default: "ddd",
		},
	}
	res := HelpMessage(flags)
	expected := `--one OR -o

--two OR -t
	what it is
	Usage: How to use it
	Default: aaa

--three

--four
	Default: ddd

`
	if res != expected {
		t.Fatalf("%s\n!=\n%s", res, expected)
	}
}
