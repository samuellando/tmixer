package config

type WindowConfig struct {
	Name    *string      `yaml:"name"`
	Command []string     `yaml:"command"`
	Panes   []PaneConfig `yaml:"panes"`
}
