package flags

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
)

type Flag struct {
	Name        string
	ShortName   string
	Description string
	Usage       string
	Default     string
	ParseInput  func(string, *config.Config) error
}

func HelpMessage(flags []Flag) string {
	out := ""
	for _, flag := range flags {
		out += fmt.Sprintf("--%s", flag.Name)
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
		out += "\n"
	}
	return out
}

func ParseArgs(ctx context.Context, args []string, flags []Flag, conf *config.Config) ([]string, error) {
	type flagParseEvent struct {
		InputArgs     []string       `json:"inputArgs"`
		ParsedFlags   [][]string     `json:"parsedFlags"`
		RemainingArgs []string       `json:"remainingArgs"`
		Result        *config.Config `json:"result"`
		Errors        []string       `json:"errors"`
	}
	event := &flagParseEvent{
		InputArgs:     args,
		ParsedFlags:   make([][]string, 0),
		RemainingArgs: nil,
		Errors:        make([]string, 0),
	}
	finish := log.Track(ctx, "flagParseEvent", event)
	defer finish()

	remaining := make([]string, 0)
	var err error
	for i := 1; i < len(args); i++ {
		matched := false
		for _, flag := range flags {
			if (isLongFlag(args[i]) && args[i][2:] == flag.Name) || (isShortFlag(args[i]) && args[i][1:] == flag.ShortName) {
				matched = true
				var val string
				if i+1 < len(args) && !isFlag(args[i+1]) {
					val = args[i+1]
					i++
				} else {
					val = flag.Default
				}
				if parseerr := flag.ParseInput(val, conf); parseerr != nil {
					err = errors.Join(err, fmt.Errorf("While parsing flag %s: %w", flag.Name, parseerr))
					event.Errors = append(event.Errors, err.Error())
					return nil, err
				}
				event.ParsedFlags = append(event.ParsedFlags, []string{flag.Name, val})
				break
			}
		}
		if !matched {
			remaining = append(remaining, args[i])
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
