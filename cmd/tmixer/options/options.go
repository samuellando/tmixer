package options

import (
	"flag"
)

var FLAG_SET = flag.NewFlagSet("", flag.ContinueOnError)

var DISPLAY_HELP bool
var VERBOSE bool

func init() {
	FLAG_SET.BoolVar(&DISPLAY_HELP, "help", false, "Display the help message")
	FLAG_SET.BoolVar(&VERBOSE, "verbose", false, "Display debug logs")
}
