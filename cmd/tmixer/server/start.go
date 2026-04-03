package server

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func Start() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find the server executable: %v", err)
	}
	cmd := exec.Command(exe, "server")
	// Detach process: put in new process group so it keeps running after we exit.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// Silence child stdio (runs as a daemon). Change to a logfile if you want server logs.
	devNull, _ := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	cmd.Stdin = devNull
	if startErr := cmd.Start(); startErr != nil {
		return fmt.Errorf("failed to start server process: %v", err)
	}
	return nil
}
