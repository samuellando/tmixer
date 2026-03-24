package flags

import (
	"fmt"
	"sort"
)

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
