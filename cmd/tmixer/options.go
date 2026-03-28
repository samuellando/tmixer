package main

import (
	"fmt"
	"strconv"
	"strings"

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
	"config": {
		ShortName:   "c",
		Description: "Provide an additional config file overriding global configs from ~/.tmixer.yml and ~/.config/tmixer/config.yml",
		Usage:       "--config config.yml or -c config.yml",
		Default:     "none",
		ParseInput: func(s string, c *config.Config) error {
			if s == "" {
				return fmt.Errorf("config file must be provided")
			}
			c.ConfigFiles = []string{"~/.config/tmixer/config.yml", "~/.tmixer.yml"}
			if s != "none" {
				c.ConfigFiles = append(c.ConfigFiles, s)
			}
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
		Description: "Whether projects from all config files should be combined or overridden, --config > ~/.tmixer.yml > ~/.config/tmixer/config.yml",
		Usage:       "--combineProjects false",
		Default:     "true",
		ParseInput: func(s string, c *config.Config) error {
			bool, err := strconv.ParseBool(s)
			if err != nil {
				return err
			}
			c.CombineProjects = ptr(bool)
			return nil
		},
		EnvironmentVariable: "TMIXER_COMBINE_PROJECTS",
	},
	"projectTtl": {
		Description: "The project time to live, after it's inactive for a certain time it will automatically be killed",
		Usage:       "--projectTtl 10h",
		ParseInput: func(s string, c *config.Config) error {
			if s == "" {
				return fmt.Errorf("REQUIRES TTL VALUE")
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
				return fmt.Errorf("REQUIRES SOCKET PATH")
			}
			c.TmuxSocketPath = &s
			return nil
		},
		EnvironmentVariable: "TMIXER_SOCKET_PATH",
	},
	"fzfFlags": {
		Description: "Flags to pass to fzf",
		Usage:       "comma or new line separated",
		Default: `--ansi
		--bind,ctrl-k:execute(tmixer kill {2})+reload(tmixer list)
		--bind,ctrl-r:execute(tmixer reset {2})+reload(tmixer list)
		--bind,ctrl-s:execute(tmixer start {2})+reload(tmixer list)
		`,
		ParseInput: func(s string, c *config.Config) error {
			s = strings.ReplaceAll(s, "\n", ",")
			flags := strings.Split(s, ",")
			c.FzfFlags = make([]string, 0, len(flags))
			for _, f := range flags {
				s = strings.TrimSpace(f)
				if s != "" {
					c.FzfFlags = append(c.FzfFlags, s)
				}
			}
			return nil
		},
		EnvironmentVariable: "TMIXER_FZF_FLAGS",
	},
}
