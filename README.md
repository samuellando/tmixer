<div align="center">
    <h1>tmixer</h1>
    <p>v0.1.0-alpha</p>
</div>

> [!NOTE]
> Tmixer is currently in alpha phase. It's still not complete, and still mostly untested. I don't recommend using this just yet.

tmixer is a feature rich and very fast [tmux](https://github.com/tmux/tmux) sessionier written in go.

- tmixer indexes all your projects, and allows you to quickly switch between tmux sessions created for them
    - In the directory you want
    - With the windows you want
    - With any extra setup commands run every time you switch to the session
- tmixer is fast
    - It communicates with tmux using control mode, and communicates with it using stream instead
    of spwning a new process for each command.

## Usage

After installing tmixer just add the following to you tmux.config:
```
bind s display-popup -E "tmixer"
bind r run-shell "tmixer reset"
```

Now hitting `leader - s` will open a fzf window listing all your projects, both with active 
sessions and not.

Within this window, you can search for the project you want and press enter to switch to it.

### tmixer command

```
tmixer [flags] [command] [project_name]

Commands:

All commands will by default open fzf if no project_name is provided. Except for
the start command, which will start the configured default project in a new tmux client.

switch (default)
        switch the active tmux client to the project. It will eitehr switch to the
        existing session and run it's configured switch comands or it will start
        a session for the project, open it's configured startup windows and
        then switch to it.

        Note that a projects switch commands are automatically run any time you switch
        to the session in tmux, even without this command. For example with leader-b.

start
        Equivalent to starting tmux normally, but will open into a project.

stop
        kill the session/project.

reset
        stop and restart the project session.

notify-switch
        Internal command used to hook into tmux for when it switches session.
        tmixer automatically sets up the tmux hooks when it is run.

Flags:

--log OR -l
        Output logs to a file, for debug purposes
        Usage: --log out.log, or -l out.log

--help OR -h
        display a help message to stdout
        Usage: --help or -h

--config OR -c
        Provide an additional config file overriding global configs from ~/.tmixer.yml and ~/.config/tmixer/config.yml
        Usage: --config config.yml or -c config.yml

--defaultProject
        Set a default project for the start command, if none is passed
        Usage: --defaultProject projects--tmixer

--combineProjects
        Wheter projects from all config files should be combined or overridden, --config > ~/.tmixer.yml > ~/.config/tmixer/config.yml
        Usage: --combineProjects false
        Default: true
```

## Configuration

Here is an example configuration:

`~/.tmixeryml` or `~/.config/tmixer/config.yml`

```yml
defaultProject: projects--dots

projects:
  # A standard project
  bin:
    directory: "~/bin"
  # This includes all sub directories as their own projects
  projects:
    directory: "~/Projects"
    subDirectories: true

    startupWindows:
      - name: "nvim"
        command: "nvim ."

      - name: "shell"

    switchCommands:
      - "ln -sf /opy/core/usr $(pwd)"
```

## Roadmap

- Extra key bindings in the fzf window
- Different fzfs 
- Config panes
- project level configs
- TTL for sessions after which they are auto reset (based on the creation time)
- Github projects
- Github worktress
- Preview window??
- Fuzz test session names
