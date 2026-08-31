//go:build !windows

package main

import "os/exec"

func hideCommandWindow(_ *exec.Cmd) {}

func startLoginCommand(executable string, args ...string) error {
	return exec.Command(executable, args...).Start()
}
