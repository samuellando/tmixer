package config

type PaneConfig struct {
	Command []string `yaml:"command"`
	Split   *string  `yaml:"split"`
}
