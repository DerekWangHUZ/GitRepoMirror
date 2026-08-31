//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

func hideCommandWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}

func startLoginCommand(executable string, args ...string) error {
	commandArgs := append([]string{"/c", "start", "", "/wait", executable}, args...)
	return exec.Command("cmd.exe", commandArgs...).Run()
}
