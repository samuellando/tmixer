package config

type ProjectConfig struct {
	Name           string         `yaml:"name"`
	Directory      string         `yaml:"directory"`
	SubDirectories bool           `yaml:"subDirectories"`
	Windows        []WindowConfig `yaml:"windows"`
	SwitchCommands [][]string     `yaml:"switchCommands"`
}
