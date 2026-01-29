package main

import (
	"testing"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
)

func TestHelpFlag(t *testing.T) {
	conf := config.New()
	flag := FLAGS["help"]

	err := flag.ParseInput("", conf)
	if err != nil {
		t.Error(err)
	}

	if !OPTION_DISPLAY_HELP {
		t.Error("Should set the display help option")
	}
}

func TestLogFileFlag(t *testing.T) {
	conf := config.New()
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

func TestLogLevelFlag(t *testing.T) {
	conf := config.New()
	flag := FLAGS["logLevel"]

	err := flag.ParseInput("info", conf)
	if err != nil {
		t.Error(err)
	}
	if conf.LogLevel != log.LEVEL_INFO {
		t.Error("Should set log level to info")
	}

	err = flag.ParseInput("DEBUG", conf)
	if err != nil {
		t.Error(err)
	}
	if conf.LogLevel != log.LEVEL_DEBUG {
		t.Error("Should set log level to debug")
	}

	err = flag.ParseInput("i", conf)
	if err != nil {
		t.Error(err)
	}
	if conf.LogLevel != log.LEVEL_INFO {
		t.Error("Should set log level to info")
	}

	err = flag.ParseInput("d", conf)
	if err != nil {
		t.Error(err)
	}
	if conf.LogLevel != log.LEVEL_DEBUG {
		t.Error("Should set log level to debug")
	}

	err = flag.ParseInput("verbose", conf)
	if err == nil {
		t.Error("Requires a recognized log level")
	}
}

func TestLogRetentionDaysFlag(t *testing.T) {
	conf := config.New()
	flag := FLAGS["logRetentionDays"]

	err := flag.ParseInput("", conf)
	if err == nil {
		t.Error("Requires input")
	}

	err = flag.ParseInput("none", conf)
	if err != nil {
		t.Error(err)
	}
	if conf.LogRetentionDays == nil || *conf.LogRetentionDays != -1 {
		t.Error("Log retention days should be set to -1")
	}

	err = flag.ParseInput("5", conf)
	if err != nil {
		t.Error(err)
	}
	if conf.LogRetentionDays == nil || *conf.LogRetentionDays != 5 {
		t.Error("Log retention days should be set to 5")
	}

	err = flag.ParseInput("abc", conf)
	if err == nil {
		t.Error("Requires a numeric log retention value")
	}
}

func TestConfigFlag(t *testing.T) {
	conf := config.New()
	flag := FLAGS["config"]
	startingConfigFiles := len(conf.ConfigFiles)

	err := flag.ParseInput("", conf)
	if err == nil {
		t.Error("Requires input")
	}

	err = flag.ParseInput("config.yml", conf)
	if err != nil {
		t.Error(err)
	}
	if len(conf.ConfigFiles) != startingConfigFiles+1 || conf.ConfigFiles[len(conf.ConfigFiles)-1] != "config.yml" {
		t.Error("Config file should be added to config files")
	}

	err = flag.ParseInput("another.yml", conf)
	if err != nil {
		t.Error(err)
	}
	if len(conf.ConfigFiles) != startingConfigFiles+2 || conf.ConfigFiles[len(conf.ConfigFiles)-1] != "another.yml" {
		t.Error("Config file should be appended to config files")
	}
}

func TestDefaultProjectFlag(t *testing.T) {
	conf := config.New()
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
	conf := config.New()
	flag := FLAGS["combineProjects"]

	err := flag.ParseInput("false", conf)
	if err != nil {
		t.Error(err)
	}
	if conf.CombineProjects {
		t.Error("Combine projects should be set to false")
	}

	err = flag.ParseInput("true", conf)
	if err != nil {
		t.Error(err)
	}
	if !conf.CombineProjects {
		t.Error("Combine projects should be set to true")
	}

	err = flag.ParseInput("maybe", conf)
	if err == nil {
		t.Error("Requires a valid boolean value")
	}
}

func TestProjectTtlFlag(t *testing.T) {
	conf := config.New()
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
