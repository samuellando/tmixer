package flags_test

import (
	"testing"

	"samuellando.com/tmixer/internal/flags"
)

func TestHelpMessage(t *testing.T) {
	inFlags := map[string]flags.Flag{
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
	res := flags.HelpMessage(inFlags)
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
