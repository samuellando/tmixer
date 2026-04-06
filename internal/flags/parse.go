package flags

import (
	"context"
	"fmt"
	"os"
	"strings"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
)

// Parse the args for the flags specified in inFlags
// The following order of precedence is used.
// 1. If a flags is present in args:
//   - It will be set to the next arguments value
//   - If it's a bare flag it's function will be called on an empty string
//
// 2. If the flag's environment variable is set, it will be set to that value
// 3. FInally, if the flag has a default value, it will be set to that value
func ParseArgs(ctx context.Context, args []string, inFlags map[string]Flag, conf config.Config) (*config.Config, []string, error) {
	logEvent := log.Track(ctx, "flagParseEvent")
	defer logEvent.Done()
	logEvent.Log("inputArgs", args)

	// Create a copy of the input, using pointers so we can keep track of what is set
	flags := make(map[string]*Flag)
	for name, flag := range inFlags {
		cpFlag := flag
		flags[name] = &cpFlag
	}

	remaining, err := setFlagsFromArgs(&conf, args, flags)
	if err != nil {
		logEvent.Error(err)
		return nil, nil, err
	}
	err = setFlagsFromEnvVars(&conf, flags)
	if err != nil {
		logEvent.Error(err)
		return nil, nil, err
	}
	err = setFlagsToDefault(&conf, flags)
	if err != nil {
		logEvent.Error(err)
		return nil, nil, err
	}

	logEvent.Log("remainingArgs", remaining)
	logEvent.Log("result", conf)
	return &conf, remaining, nil
}

func setFlagsFromArgs(conf *config.Config, args []string, flags map[string]*Flag) ([]string, error) {
	remaining := make([]string, 0)
	for i := 0; i < len(args); i++ {
		matched := false
		for name, flag := range flags {
			if flag.matches(name, args[i]) {
				matched = true
				flag.isSet = true
				var val string
				if !flag.Bare {
					if i+1 >= len(args) {
						return nil, fmt.Errorf("Flag %s requires a value", name)
					}
					val = args[i+1]
					i++
				}
				if parseErr := flag.ParseInput(val, conf); parseErr != nil {
					err := fmt.Errorf("while parsing flag %s: %w", name, parseErr)
					return nil, err
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

func setFlagsFromEnvVars(conf *config.Config, flags map[string]*Flag) error {
	for name, flag := range flags {
		if !flag.isSet && flag.EnvironmentVariable != "" {
			if val, set := os.LookupEnv(flag.EnvironmentVariable); set {
				flag.isSet = true
				if parseErr := flag.ParseInput(val, conf); parseErr != nil {
					return fmt.Errorf("while parsing flag env var %s: %w", name, parseErr)
				}
			}
		}
	}
	return nil
}

func setFlagsToDefault(conf *config.Config, flags map[string]*Flag) error {
	for name, flag := range flags {
		if !flag.isSet && flag.Default != "" {
			flag.isSet = true
			if parseErr := flag.ParseInput(flag.Default, conf); parseErr != nil {
				return fmt.Errorf("while parsing flag env var %s: %w", name, parseErr)
			}
		}
	}
	return nil
}

func isLongFlag(arg string) bool {
	return strings.HasPrefix(arg, "--")
}

func isShortFlag(arg string) bool {
	return strings.HasPrefix(arg, "-")
}
