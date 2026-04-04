package options

import (
	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/flags"
)

func ptr[T any](v T) *T {
	return &v
}

var FLAGS = map[string]flags.Flag{
	"help": {
		ShortName:   "h",
		Description: "display a help message to stdout",
		Usage:       "--help or -h",
		Bare:        true,
		ParseInput: func(s string, c *config.Config) error {
			c.DisplayHelp = ptr(true)
			return nil
		},
	},
	"verbose": {
		ShortName:   "v",
		Description: "display information about the run",
		Usage:       "--verbose or -v",
		Bare:        true,
		ParseInput: func(s string, c *config.Config) error {
			c.DisplayLog = ptr(true)
			return nil
		},
	},
}
