package flags

import (
	"samuellando.com/tmixer/internal/config"
)

type Flag struct {
	ShortName           string
	Description         string
	Usage               string
	Default             string
	Bare                bool
	ParseInput          func(string, *config.Config) error
	EnvironmentVariable string
	isSet               bool
}

func (flag *Flag) matches(name string, arg string) bool {
	return (isLongFlag(arg) && arg[2:] == name) || (isShortFlag(arg) && arg[1:] == flag.ShortName)
}
