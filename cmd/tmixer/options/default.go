package options

import (
	"samuellando.com/tmixer/internal/config"
)

var DEFAULT_CONFIG = config.Config{
	ConfigFiles: []string{
		"~/.config/tmixer/config.yml",
		"~/.tmixer.yml",
	},
	CombineProjects: ptr(true),
	FzfFlags: []string{
		"--ansi",
		"--bind", "ctrl-k:execute(tmixer kill {2})+reload(tmixer list)",
		"--bind", "ctrl-r:execute(tmixer reset {2})+reload(tmixer list)",
		"--bind", "ctrl-s:execute(tmixer start {2})+reload(tmixer list)",
	},
}
