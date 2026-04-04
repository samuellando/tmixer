package options

import (
	"testing"

	"samuellando.com/tmixer/internal/config"
)

func TestHelpFlag(t *testing.T) {
	t.Parallel()
	conf := &config.Config{}
	flag := FLAGS["help"]

	err := flag.ParseInput("", conf)
	if err != nil {
		t.Error(err)
	}

	if !*conf.DisplayHelp {
		t.Error("Should set the display help option")
	}
}

func TestVerboseFlag(t *testing.T) {
	t.Parallel()
	conf := &config.Config{}
	flag := FLAGS["verbose"]

	err := flag.ParseInput("", conf)
	if err != nil {
		t.Error(err)
	}

	if !*conf.DisplayLog {
		t.Error("Should set the display help option")
	}
}

func TestLogFileFlag(t *testing.T) {
	t.Parallel()
	conf := &config.Config{}
	flag := FLAGS["logFile"]

	err := flag.ParseInput("", conf)
	if err == nil {
		t.Error("Requires input")
	}

	err = flag.ParseInput("out.txt", conf)
	if err != nil {
		t.Error(err)
	}
	if conf.LogFile == nil || *conf.LogFile != "out.txt" {
		t.Error("Log file should be set in conf")
	}
}

func TestConfigFlag(t *testing.T) {
	t.Parallel()
	conf := &config.Config{}
	flag := FLAGS["config"]

	err := flag.ParseInput("", conf)
	if err == nil {
		t.Error("Requires input")
	}

	err = flag.ParseInput("config.yml", conf)
	if err != nil {
		t.Error(err)
	}
	if len(conf.ConfigFiles) != 3 || conf.ConfigFiles[len(conf.ConfigFiles)-1] != "config.yml" {
		t.Error("Config file should be added to config files")
	}
}

func TestDefaultProjectFlag(t *testing.T) {
	t.Parallel()
	conf := &config.Config{}
	flag := FLAGS["defaultProject"]

	err := flag.ParseInput("", conf)
	if err == nil {
		t.Error("Requires input")
	}

	err = flag.ParseInput("projects--tmixer", conf)
	if err != nil {
		t.Error(err)
	}
	if conf.DefaultProject == nil || *conf.DefaultProject != "projects--tmixer" {
		t.Error("Default project should be set in conf")
	}
}

func TestCombineProjectsFlag(t *testing.T) {
	t.Parallel()
	conf := &config.Config{}
	flag := FLAGS["combineProjects"]

	err := flag.ParseInput("false", conf)
	if err != nil {
		t.Error(err)
	}
	if *conf.CombineProjects {
		t.Error("Combine projects should be set to false")
	}

	err = flag.ParseInput("true", conf)
	if err != nil {
		t.Error(err)
	}
	if !*conf.CombineProjects {
		t.Error("Combine projects should be set to true")
	}

	err = flag.ParseInput("maybe", conf)
	if err == nil {
		t.Error("Requires a valid boolean value")
	}
}

func TestProjectTtlFlag(t *testing.T) {
	t.Parallel()
	conf := &config.Config{}
	flag := FLAGS["projectTtl"]

	err := flag.ParseInput("", conf)
	if err == nil {
		t.Error("Requires input")
	}

	err = flag.ParseInput("10h", conf)
	if err != nil {
		t.Error(err)
	}
	if conf.Ttl == nil || *conf.Ttl != "10h" {
		t.Error("Project ttl should be set in conf")
	}
}

func TestTmuxSocketPathFlag(t *testing.T) {
	t.Parallel()
	conf := &config.Config{}
	flag := FLAGS["tmuxSocketPath"]

	err := flag.ParseInput("", conf)
	if err == nil {
		t.Error("Requires input")
	}

	err = flag.ParseInput("sock", conf)
	if err != nil {
		t.Error(err)
	}
	if conf.TmuxSocketPath == nil || *conf.TmuxSocketPath != "sock" {
		t.Error("tmux socket should be set in conf")
	}
}

func TestFzfFlagsFlag(t *testing.T) {
	t.Parallel()
	conf := &config.Config{}
	flag := FLAGS["fzfFlags"]

	err := flag.ParseInput("--ansi,--bind", conf)
	if err != nil {
		t.Error(err)
	}
	if conf.FzfFlags == nil || conf.FzfFlags[0] != "--ansi" {
		t.Error("fzf flags should be set in conf")
	}
}
