package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Runner is the narrow process boundary consumed by application services.
type Runner interface {
	Quiet(name string, args ...string) (string, error)
	QuietIn(dir, name string, args ...string) (string, error)
	Run(id, name string, args ...string) error
	RunIn(id, dir, name string, args ...string) error
}

func (a *App) commandEnv() []string {
	environment := os.Environ()
	proxy := a.store.Settings().Proxy
	if proxy != "" {
		environment = append(environment, "HTTP_PROXY="+proxy, "HTTPS_PROXY="+proxy, "ALL_PROXY="+proxy)
	}
	return append(environment, "GIT_TERMINAL_PROMPT=0")
}

func (a *App) runQuiet(name string, args ...string) (string, error) {
	return a.runQuietIn("", name, args...)
}
func (a *App) runQuietIn(dir, name string, args ...string) (string, error) {
	if a.quietHook != nil {
		return a.quietHook(dir, name, args...)
	}
	resolved, err := findExecutable(name)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(a.commandContext(), resolved, args...)
	cmd.Dir, cmd.Env = dir, a.commandEnv()
	hideCommandWindow(cmd)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
func (a *App) runCommand(id, name string, args ...string) error {
	return a.runCommandIn(id, "", name, args...)
}
func (a *App) runCommandIn(id, dir, name string, args ...string) error {
	a.emitLog(id, "command", "$ "+name+" "+strings.Join(redactArgs(args), " "))
	if a.commandHook != nil {
		return a.commandHook(id, dir, name, args...)
	}
	resolved, err := findExecutable(name)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(a.commandContext(), resolved, args...)
	cmd.Dir, cmd.Env = dir, a.commandEnv()
	hideCommandWindow(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	var wg sync.WaitGroup
	stream := func(reader io.Reader, level string) {
		defer wg.Done()
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			a.emitLog(id, level, scanner.Text())
		}
	}
	wg.Add(2)
	go stream(stdout, "info")
	go stream(stderr, "stderr")
	wg.Wait()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("命令退出: %w", err)
	}
	return nil
}

func redactArgs(args []string) []string {
	result := append([]string(nil), args...)
	for i, argument := range result {
		if parsed, err := url.Parse(argument); err == nil && parsed.User != nil {
			parsed.User = url.User("***")
			result[i] = parsed.String()
		}
	}
	return result
}
func (a *App) commandContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}
