<div align="center">
    <h1>tmixer</h1>
    <p>v0.4.0-alpha</p>
</div>

> [!NOTE]
> Tmixer is currently in alpha and there are no official releases yet. It's unstable and may change with any future releases.

## Overview

tmixer is a feature rich and fast [tmux](https://github.com/tmux/tmux) session manager written in go.

- tmixer indexes all the projects on your system, and allows you to quickly switch between tmux sessions created for them
- tmixer has powerful per-project configurations for:
    - Automatically spawning tmux window and pane layouts at startup, with specific commands to execute on each.
    - Hooks that should be automatically executed every time the project is switched to.
- In addition to basing projects on specific directories, tmixer can also import projects from:
    - Sub directories of a specific parent directory, for example ~/Projects will import all the projects in that directory
    - Importers such as git-worktrees, github, gitlab, etc. (Coming soon)
- tmixer is fast. It communicates with tmux using control mode instead of spawning a new process for each command.

## Dependencies

- tmux
- fzf

## Installation

Since tmixer is still in early development, clone the repository and run `make install`.

Once the project has reached an official stable release some more installation
options will be made available.

## Quickstart

1. Add the following to your tmux.config:
    ```
    bind s display-popup -E "tmixer" # Open an fzf picker of all the projects
    bind r run-shell "tmixer reset"  # Reset the currently attached project
    ```

2. Add something like the following to `~/.tmixer.yml` or `~/.config/tmixer/config.yml`
    ```yml
    defaultProject: home

    projects:
      home:
        directory: "~/"

      bin:
        directory: "~/bin"

      projects:
        directory: "~/Projects"
        subDirectories: true # Will list all projects within this directory

        # See the configuration section on how to add windows and panes
    ```

3. Run `tmixer start` This will open tmux in your default project.

4. Hitting `leader - s` will open a fzf window listing all your projects, both with active 
sessions and without. 
    - Within this window, you can search for the project you want and press enter to start/switch to it.

    - There are also some extra keybindings in this window:

        - `ctrl-k` will kill the active highlighted session with `tmixer kill`
        - `ctrl-r` will call `tmixer reset` on the highlighted project
        - `ctrl-s` will start the highlighted project with `tmixer start` but not switch to it

5. Hitting `leader - r` will reset the currently attached project. Killing the session,
and recreating it will all it's configured windows and panes.

## tmixer command

While the primary interface with tmixer will be the popup window configured above,
there is also a CLI.

```
tmixer [flags] [command] [project_name]

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

Flags:
```

## Configuration

Tmixer can be configured with config files, environment variables and command line flags.

The tmixer configuration is where you specify all your projects, as well as some other global options. 

This can be done in:
- `~/.tmixer.yml`
- `~/.config/tmixer/config.yml`
- a local `.tmixer.yml` file in the project directory. (Project level configs only)

Projects from all the config files will be combined unless the `combineProjects` 
is set to `false`.

Configurations will be overridden in the following order:
- `configs passed to command line` overrides
- `environment variables` overrides
- `~/.tmixer.yml` overrides
- `~/.config/tmixer/config.yml`


### Summary of all configuration options

#### Top level/global configurations

| yaml option      | type              | description                                                                                          | default value  |
| ---              | ---               | ---                                                                                                  | ---            |
| defaultProject   | `string`          | The default project to open at startup                                                               |                |
| ttl              | `string`          | How long before a project with no activity is automatically killed (example `24h`)                   |                |
| combineProjects  | `bool`            | Whether the list of projects should be overridden or combined when multiple config files are present | `true`         |
| logFile          | `string`          | Additional log file in addition to `~/.local/state/tmixer/logs/`                                     |                |
| tmuxSocketPath   | `string`          | What tmux socket to use, equivalent to the tmux `-S` flag                                            | `tmux default` |
| projects         | `[]projectConfig` | A list of project configurations                                                                     |                |

#### Project configurations

| yaml option    | type             | description                                                                                             | default value |
| ---            | ---              | ---                                                                                                     | ---           |
| name*          | `string`         | The name of the project                                                                                 |               |
| directory*     | `string`         | The directory of the project                                                                            |               |
| subDirectories | `bool`           | If true, the subdirectories will be loaded as individual projects with names `[parent_name]--[sub_dir]` | `false`       |
| windows        | `[]windowConfig` | A list of window configurations                                                                         |               |
| switchCommands | `[][]string`     | A list of commands to run every time this project is switched to                                        |               |

* Required

#### Window configurations

| yaml option | type           | description                                                                                  | default value                |
| ---         | ---            | ---                                                                                          | ---                          |
| name        | `string`       | The name of the window, will display in tmux status bar                                      | Displays the current program |
| command     | `[]string`     | The startup command to execute on the first pane of this window, shorthand for adding a pane |                              |
| panes       | `[]paneConfig` | A list of pane configurations                                                                |                              |

#### Pane configurations

| yaml option | type       | description                                                                                     | default value |
| ---         | ---        | ---                                                                                             | ---           |
| command     | `[]string` | The startup command for this pane                                                               |               |
| split       | `string`   | How to split the previously defined pane when splitting this pane  (`vertical` or `horizontal`) | `horizontal`  |


### Example configuration

```yml
defaultProject: projects--dots
ttl: 24h

projects:
  - name: home
    directory: "~/"

  - name: bin
    directory: "~/bin"

  - name: projects
    directory: "~/Projects"
    subDirectories: true

    windows:
      - command: ["nvim", "."]

      - name: "shell"

      - name: logs 
        panes:
          - command: ["tail", "-f", "./logs.out"]
          - command: ["tail", "-f", "./err.out"]
            split: "vert"

    switchCommands:
      - "ln -sfn /opt/corp/sys/usr ."
```

### Command line flags and environment variables

```
--combineProjects
        Whether projects from all config files should be combined or overridden, --config > ~/.tmixer.yml > ~/.config/tmixer/config.yml
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

--projectTtl
        The project time to live, after it's inactive for a certain time it will automatically be killed
        Usage: --projectTtl 10h
        EnvVar: $TMIXER_PROJECT_TTL

--tmuxSocketPath OR -S
        The tmux socket path, the tmux -S flag
        Usage: --tmuxSocketPath /tmp/tmux/socket.sock
        EnvVar: $TMIXER_SOCKET_PATH
``` 
