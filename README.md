<div align="center">
    <h1>tmixer</h1>
    <p>v0.3.0-alpha</p>
</div>

> [!NOTE]
> Tmixer is currently in alpha and there are no official releases yet. It's unstable and may change with any future releases.

## Overview

tmixer is a feature rich and speedy [tmux](https://github.com/tmux/tmux) sessionier written in go.

- tmixer allows you to configure your projects:
    - In specific directories or sub directories
    - With specific windows and panes to create at startup
    - With switching commands to run every time you switch to the project
- tmixer then indexes all your projects on your system, and allows you to quickly switch between tmux sessions created for them
- tmixer is fast
    - It communicates with tmux using control mode, using streams instead of spawning a new process for each command.

## Dependencies

- tmux
- fzf

## Installation

Since tmixer is still in early development, clone the repository and run `make install`

## Usage

After installing tmixer just add the following to you tmux.config:
```
bind s display-popup -E "tmixer"
```

Now hitting `leader - s` will open a fzf window listing all your projects, both with active 
sessions and without. 

Within this window, you can search for the project you want and press enter to switch to it.

There are also some extra keybindings in this window:

- `ctrl-k` will kill the active highlighted session with `tmixer kill`
- `crtl-r` will call `tmixer reset` on the highlighted project
- `crtl-s` will start the highlighted project with `tmixer start` but not switch to it


Another useful thing to add your tmux config is the following
```
bind r run-shell "tmixer reset"
````

This will reset the currently attached project when pressing `leader-r`

### tmixer command

While the primary interface with tmixer will be the popup window configured above,
there is also a CLI.

```
tmixer [flags] [command] [project_name]

Commands:

All commands will by default open fzf if no project_name is provided. Except for
the start command, which will start the configued default project in a new tmux client,
and the reset command, which will reset the currently attached session.

switch (default)
        switch the active tmux client to the project. It will eitehr switch to the
        existing session or start a session with the project configuration.

        After switching to a project/session, tmixer will run it's configured
        switch commands.

        Note that a project's switch commands are automatically run any time you switch
        to the session in tmux, even without this command. For example with leader-b.

start
        Equivalent to starting tmux normally, but will open into a project. By
        default it starts the default configured project.

kill
        kill the session/project.

reset
        kill and restart the project session. By default will reset the attached session.
```

## Configuration

The tmixer configuration is where you specify all your projects, as well as some 
other global options.

This can be done in `~/.tmixer.yml`, `~/.config/tmixer/config.yml`, any
config files passed to the command line, and finally a local `.tmixer.yml` file
in the project directory.

Projects from all the config files will be combined unless the `combineProjects` 
is set to `false`. Global options will be overriden in the following order 
`configs passed to command line` overrides `environment variables` overrides `~/.tmixer.yml` overrides `~/.config/tmixer/config.yml`

Here is an example configuration:

```yml
defaultProject: projects--dots

projects:
  home:
    directory: "~/"

  bin:
    directory: "~/bin"

  projects:
    directory: "~/Projects"
    subDirectories: true

    windows:
      - name: "nvim"
        command: "nvim ."

      - name: "shell"

      - name: logs 
        command: "echo hello"
        panes:
          - command: "tail -f ~/.tmixer.yml"
          - command: "ls ~/Documents"
            split: "vert"

    switchCommands:
      - "ls"
```

And here are all the command line flags and their equivalent environment variables:

```
--combineProjects
        Wheter projects from all config files should be combined or overridden, --config > ~/.tmixer.yml > ~/.config/tmixer/config.yml
        Usage: --combineProjects false
        Default: true
        EnvVar: $TMIXER_COMBINE_PROJECTS

--config OR -c
        Provide an additional config file overriding global configs from ~/.tmixer.yml and ~/.config/tmixer/config.yml
        Usage: --config config.yml or -c config.yml
        EnvVar: $TMIXER_CONFIG

--defaultProject
        Set a default project for the start command, if none is passed
        Usage: --defaultProject projects--tmixer
        EnvVar: $TMIXER_DEFAULT_PROJECT

--help OR -h
        display a help message to stdout
        Usage: --help or -h

--logFile
        Output logs to a file in addition to ~/.local/state/tmixer/logs .
        Usage: --logFile out.log
        EnvVar: $TMIXER_LOG_FILE

--logLevel
        logging level: info or debug
        Usage: --logLevel debug
        Default: info
        EnvVar: $TMIXER_LOG_LEVEL

--logRetentionDays
        how many previous days of logs to keep in ~/.local/state/tmixer/logs. Set to 0 to only keep current day and none to disable logging.
        Usage: --logRetentionDays 5
        Default: 1
        EnvVar: $TMIXER_LOG_RETENTION_DAYS

--projectTtl
        The project time to live, after it's inactive for a certain time it will automatically be killed
        Usage: --projectTtl 10h
        EnvVar: $TMIXER_PROJECT_TTL

--tmuxSocketPath OR -S
        The tmux socket path, the tmux -S flag
        Usage: --tmuxSocketPath /tmp/tmux/socket.sock
        EnvVar: $TMIXER_SOCKET_PATH
``` 
