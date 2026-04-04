package options

import (
	"fmt"

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
	"config": {
		ShortName:   "c",
		Description: "Provide an additional config file overriding global configs from ~/.tmixer.yml and ~/.config/tmixer/config.yml",
		Usage:       "--config config.yml or -c config.yml",
		ParseInput: func(s string, c *config.Config) error {
			if s == "" {
				return fmt.Errorf("config file must be provided")
			}
			c.ConfigFiles = append(c.ConfigFiles, s)
			return nil
		},
		EnvironmentVariable: "TMIXER_CONFIG",
	},
}
