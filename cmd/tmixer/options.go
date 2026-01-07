package main

import (
	"fmt"
	"strconv"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/flags"
)

var (
	OPTION_DISPLAY_HELP = false
)

var FLAGS = []flags.Flag{
	{
		Name:        "log",
		ShortName:   "l",
		Description: "Output logs to a file, for debug purposes",
		Usage:       "--log out.log, or -l out.log",
		ParseInput: func(s string, c *config.Config) error {
			if s == "" {
				return fmt.Errorf("log file must be provided")
			}
			c.LogFile = &s
			return nil
		},
	},
	{
		Name:        "help",
		ShortName:   "h",
		Description: "display a help message to stdout",
		Usage:       "--help or -h",
		ParseInput: func(s string, c *config.Config) error {
			OPTION_DISPLAY_HELP = true
			return nil
		},
	},
	{
		Name:        "logRetentionDays",
		Description: "how many days of logs to keep in ~/.local/state/tmixer/logs",
		Usage:       "--logRetentionDays 5",
		Default:     "1",
		ParseInput: func(s string, c *config.Config) error {
			if s == "" {
				return fmt.Errorf("log file must be provided")
			}
			v, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("while parsing log retentionn days: %s", err)
			}
			c.LogRetentionDays = &v
			return nil
		},
	},
	{
		Name:        "config",
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
	},
	{
		Name:        "defaultProject",
		Description: "Set a default project for the start command, if none is passed",
		Usage:       "--defaultProject projects--tmixer",
		ParseInput: func(s string, c *config.Config) error {
			if s == "" {
				return fmt.Errorf("default project must be provided")
			}
			c.DefaultProject = &s
			return nil
		},
	},
	{
		Name:        "combineProjects",
		Description: "Wheter projects from all config files should be combined or overridden, --config > ~/.tmixer.yml > ~/.config/tmixer/config.yml",
		Usage:       "--combineProjects false",
		Default:     "true",
		ParseInput: func(s string, c *config.Config) error {
			bool, err := strconv.ParseBool(s)
			if err != nil {
				return err
			}
			c.CombineProjects = bool
			return nil
		},
	},
}
