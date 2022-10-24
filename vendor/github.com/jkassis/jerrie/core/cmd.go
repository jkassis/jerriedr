package core

import (
	"os"
	"os/exec"
	"syscall"
)

// SubCmd runs a command and attaches its Stdout and Stdin to the root Stdin and Stdout
func SubCmd(cliCmd string, args ...string) (*exec.Cmd, error) {
	cmd := exec.Command(cliCmd, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	return cmd, err
}
