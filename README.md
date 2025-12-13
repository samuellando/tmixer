## Roadmap

- TTL for sessions after which they are auto refreshed (based on the creation time)
- Picking up config from the local dir, projects parent dir
    ```yaml
    StartWindows:
        - nvim:
            Command: ["nvim", "."]

    Projects:
        projects:
            Directory: "~/Projects"
            SubDirectories: true
    ``` 

    Will auto apply the global options to all in the file.
    If ~/Projects has a .tmixer.yaml, then that is picked up as well.
    If nay sub dir has a .tmixer.yaml, then that is applied on top as well.
- Capturing keybindings, or hooking
    - So we can run swap commands when tmux swaps with B-L
- Preview window
- win-dings (icons)
