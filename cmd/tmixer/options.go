package main

import (
	"fmt"
	"strconv"
	"strings"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/flags"
	"samuellando.com/tmixer/internal/log"
)

var (
	OPTION_DISPLAY_HELP = false
)

var FLAGS = []flags.Flag{
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
		Name:        "logFile",
		Description: "Output logs to a file in addition to ~/.local/state/tmixer/logs .",
		Usage:       "--logFile out.log",
		ParseInput: func(s string, c *config.Config) error {
			if s == "" {
				return fmt.Errorf("log file must be provided")
			}
			c.LogFile = &s
			return nil
		},
		EnvironmentVariable: "TMIXER_LOG_FILE",
	},
	{
		Name:        "logLevel",
		Description: "logging level: info or debug",
		Default:     "info",
		Usage:       "--logLevel debug",
		ParseInput: func(s string, c *config.Config) error {
			s = strings.ToLower(s)
			if strings.HasPrefix(s, "i") {
				c.LogLevel = log.LEVEL_INFO
			} else if strings.HasPrefix(s, "d") {
				c.LogLevel = log.LEVEL_DEBUG
			} else {
				return fmt.Errorf("Unrecognized log level")
			}
			return nil
		},
		EnvironmentVariable: "TMIXER_LOG_LEVEL",
	},
	{
		Name:        "logRetentionDays",
		Description: "how many previous days of logs to keep in ~/.local/state/tmixer/logs. Set to 0 to only keep current day and none to disable logging.",
		Usage:       "--logRetentionDays 5",
		Default:     "1",
		ParseInput: func(s string, c *config.Config) error {
			if s == "" {
				return fmt.Errorf("log file must be provided")
			}
			var v int
			var err error
			if strings.ToLower(s) == "none" {
				v = -1
			} else {
				v, err = strconv.Atoi(s)
				if err != nil {
					return fmt.Errorf("while parsing log retentionn days: %s", err)
				}
			}
			c.LogRetentionDays = &v
			return nil
		},
		EnvironmentVariable: "TMIXER_LOG_RETENTION_DAYS",
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
		EnvironmentVariable: "TMIXER_CONFIG",
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
		EnvironmentVariable: "TMIXER_DEFAULT_PROJECT",
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
		EnvironmentVariable: "TMIXER_COMBINE_PROJECTS",
	},
	{
		Name:        "projectTtl",
		Description: "The project time to live, after it's inactive for a certain time it will automatically be killed",
		Usage:       "--projectTtl 10h",
		ParseInput: func(s string, c *config.Config) error {
			c.Ttl = &s
			return nil
		},
		EnvironmentVariable: "TMIXER_PROJECT_TTL",
	},
}
