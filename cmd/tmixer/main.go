package main

import (
	"errors"
	"log"
	"os"

	"samuellando.com/tmixer/cmd/tmixer/client"
	"samuellando.com/tmixer/cmd/tmixer/server"
)

var ErrNoSelection = errors.New("NO SELECTION MADE")
var ErrProjectNotFound = errors.New("PROJECT NOT FOUND")
var ErrCommandNotRecognized = errors.New("COMMAND NOT RECOGNIZED")

func main() {
	args := os.Args[1:]
	var err error
	if len(args) >= 1 && args[0] == "server" {
		err = server.Run(args...)
	} else {
		err = client.Run(args...)
	}
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
