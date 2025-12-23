package flags

import (
	"testing"

	"samuellando.com/tmixer/internal/configv2"
)

func TestParseArgs(t *testing.T) {
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
			Name: "four",
			Default: "ddd",
			ParseInput: func(s string, c *config.Config) error {
				if s != "ddd" {
					t.Fatal("incorrect input")
				}
				counter += 1
				return nil
			},
		},
	}
	args := []string{"tmixer", "--four", "--one", "aaa", "-t", "--three", "bbb", "hello"}
	remaining, err := ParseArgs(args, flags, nil)
	if err != nil {
		t.Fatal(err)
	}
	if counter != 4 {
		t.Fatal("Should've parsed 3 flags")
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
			Name:      "two",
			ShortName: "t",
			Description: "what it is",
			Usage: "How to use it",
			Default: "aaa",

		},
		{
			Name: "three",
		},
		{
			Name: "four",
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
