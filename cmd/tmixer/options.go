package main

import (
	"fmt"
	"strconv"
	"strings"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/flags"
	"samuellando.com/tmixer/internal/log"
)

var FLAGS = map[string]flags.Flag{
	"help": {
		ShortName:   "h",
		Description: "display a help message to stdout",
		Usage:       "--help or -h",
		ParseInput: func(s string, c *config.Config) error {
			c.DisplayHelp = true
			return nil
		},
	},
	"logFile": {
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
	"logLevel": {
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
	"logRetentionDays": {
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
	"defaultProject": {
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
	"combineProjects": {
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
	"projectTtl": {
		Description: "The project time to live, after it's inactive for a certain time it will automatically be killed",
		Usage:       "--projectTtl 10h",
		ParseInput: func(s string, c *config.Config) error {
			if s == "" {
				return fmt.Errorf("Requires ttl value")
			}
			c.Ttl = &s
			return nil
		},
		EnvironmentVariable: "TMIXER_PROJECT_TTL",
	},
	"tmuxSocketPath": {
		Description: "The tmux socket path, the tmux -S flag",
		ShortName:   "S",
		Usage:       "--tmuxSocketPath /tmp/tmux/socket.sock",
		ParseInput: func(s string, c *config.Config) error {
			if s == "" {
				return fmt.Errorf("Requires socket path")
			}
			c.TmuxSocketPath = &s
			return nil
		},
		EnvironmentVariable: "TMIXER_SOCKET_PATH",
	},
}
