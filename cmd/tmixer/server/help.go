package server

import (
	"strings"

	"samuellando.com/tmixer/cmd/tmixer/options"
	"samuellando.com/tmixer/internal/protocol"
)

const helpMessage = `tmixer [flags] [command] [project_name]

Commands:

All commands will by default open fzf if no project_name is provided. Except for 
the start command, which will start the configured default project in a new tmux client, 
and the reset command, which will reset the attached session.

switch (default)
	switch the active tmux client to the project. It will either switch to the 
	existing session or start a session for the project.

	After switching to a project/session, tmixer will run it's configured 
	switch commands.

	Note that a project's switch commands are automatically run any time you switch 
	to the session in tmux. For example even with leader-b.

start
	Equivalent to starting tmux normally, but will open into a project. By
	default it starts the default configured project.

kill
	kill the project.

reset
	reset the state of the project's session to it's configured state.

Flags: `

func createHelpResponse() *protocol.Response {
	b := strings.Builder{}
	options.FLAG_SET.SetOutput(&b)
	b.WriteString(helpMessage)
	b.WriteRune('\n')
	b.WriteRune('\n')
	options.FLAG_SET.PrintDefaults()
	return outputResponse(&protocol.Output{
		Output: ptr(b.String()),
	})
}
