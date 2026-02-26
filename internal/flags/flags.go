package flags

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
)

type Flag struct {
	ShortName           string
	Description         string
	Usage               string
	Default             string
	ParseInput          func(string, *config.Config) error
	EnvironmentVariable string
}

func HelpMessage(flags map[string]Flag) string {
	out := ""
	// Collect and sort keys
	keys := make([]string, 0, len(flags))
	for k := range flags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		flag := flags[name]
		out += fmt.Sprintf("--%s", name)
		if flag.ShortName != "" {
			out += fmt.Sprintf(" OR -%s", flag.ShortName)
		}
		out += "\n"
		if flag.Description != "" {
			out += fmt.Sprintf("\t%s\n", flag.Description)
		}
		if flag.Usage != "" {
			out += fmt.Sprintf("\tUsage: %s\n", flag.Usage)
		}
		if flag.Default != "" {
			out += fmt.Sprintf("\tDefault: %s\n", flag.Default)
		}
		if flag.EnvironmentVariable != "" {
			out += fmt.Sprintf("\tEnvVar: $%s\n", flag.EnvironmentVariable)
		}
		out += "\n"
	}
	return out
}

func ParseArgs(ctx context.Context, args []string, flags map[string]Flag, conf *config.Config) ([]string, error) {
	type flagParseEvent struct {
		InputArgs     []string       `json:"inputArgs"`
		ParsedFlags   [][]string     `json:"parsedFlags"`
		RemainingArgs []string       `json:"remainingArgs"`
		Result        *config.Config `json:"result"`
		Errors        []string       `json:"errors,omitempty"`
	}
	event := &flagParseEvent{
		InputArgs:     args,
		ParsedFlags:   make([][]string, 0),
		RemainingArgs: make([]string, 0),
		Errors:        make([]string, 0),
	}
	finish := log.Track(ctx, "flagParseEvent", event)
	defer finish()

	remaining := make([]string, 0)
	flagSet := make(map[string]bool)
	var err error
	for i := 0; i < len(args); i++ {
		matched := false
		for name, flag := range flags {
			if (isLongFlag(args[i]) && args[i][2:] == name) || (isShortFlag(args[i]) && args[i][1:] == flag.ShortName) {
				matched = true
				flagSet[name] = true
				var val string
				if i+1 < len(args) && !isFlag(args[i+1]) {
					val = args[i+1]
					i++
				} else {
					val = flag.Default
				}
				if parseErr := flag.ParseInput(val, conf); parseErr != nil {
					err = errors.Join(err, fmt.Errorf("while parsing flag %s: %w", name, parseErr))
					event.Errors = append(event.Errors, err.Error())
					return nil, err
				}
				event.ParsedFlags = append(event.ParsedFlags, []string{name, val})
				break
			}
		}
		if !matched {
			remaining = append(remaining, args[i])
		}
	}

	for name, flag := range flags {
		if !flagSet[name] && flag.EnvironmentVariable != "" {
			if val, set := os.LookupEnv(flag.EnvironmentVariable); set {
				if parseErr := flag.ParseInput(val, conf); parseErr != nil {
					err = errors.Join(err, fmt.Errorf("while parsing flag env var %s: %w", name, parseErr))
					event.Errors = append(event.Errors, err.Error())
					return nil, err
				}
			}
		}
	}

	event.RemainingArgs = remaining
	event.Result = conf
	return remaining, nil
}

func isFlag(arg string) bool {
	return isLongFlag(arg) || isShortFlag(arg)
}

func isLongFlag(arg string) bool {
	return strings.HasPrefix(arg, "--")
}

func isShortFlag(arg string) bool {
	return strings.HasPrefix(arg, "-")
}
