package options

import (
	"context"
	"embed"
	"flag"
	"os"
	"path/filepath"

	"samuellando.com/tmixer/internal/config"
)

//go:embed default_config.yml
var f embed.FS

var FLAG_SET = flag.NewFlagSet("", flag.ContinueOnError)

var DISPLAY_HELP bool
var VERBOSE bool
var configs []config.FSref

func init() {
	FLAG_SET.BoolVar(&DISPLAY_HELP, "help", false, "Display the help message")
	FLAG_SET.BoolVar(&VERBOSE, "verbose", false, "Display debug logs")

	configDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	configFS := os.DirFS(filepath.Join(configDir, "tmixer"))
	homeFS := os.DirFS(homeDir)
	configs = []config.FSref{
		{FS: f, Name: "default_config.yml"},
		{FS: configFS, Name: "config.yml"},
		{FS: homeFS, Name: ".tmixer.yml"},
	}

}

func Config(ctx context.Context) (*config.Config, error) {
	return config.LoadFiles(ctx, configs)
}
