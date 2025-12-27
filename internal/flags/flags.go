package flags

import (
	"fmt"
	"strings"

	"samuellando.com/tmixer/internal/config"
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

func ParseArgs(args []string, flags []Flag, config *config.Config) ([]string, error) {
	remaining := make([]string, 0)
	for i := 1; i < len(args); i++ {
		matched := false
		for _, flag := range flags {
			if (isLongFlag(args[i]) && args[i][2:] == flag.Name) || (isShortFlag(args[i]) && args[i][1:] == flag.ShortName) {
				matched = true
				var err error
				if i+1 < len(args) && !isFlag(args[i+1]) {
					err = flag.ParseInput(args[i+1], config)
					i++
				} else {
					err = flag.ParseInput(flag.Default, config)
				}
				if err != nil {
					return nil, fmt.Errorf("While parsing flag %s: %w", flag.Name, err)
				}
				break
			}
		}
		if !matched {
			remaining = append(remaining, args[i])
		}
	}
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
