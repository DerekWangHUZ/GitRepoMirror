package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
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
	return replaceCommandEnv(environment, map[string]string{
		"GIT_TERMINAL_PROMPT": "0",
		"GCM_INTERACTIVE":     "Never",
	})
}

func (a *App) commandEnvForRemotes(remotes ...RemoteSpec) []string {
	environment := a.commandEnv()
	for _, remote := range remotes {
		if !usesGitLabHTTPS(remote) {
			continue
		}
		glabPath, err := findExecutable("glab")
		if err != nil {
			return environment
		}
		return withGitLabCredentialHelper(environment, filepath.Dir(glabPath))
	}
	return environment
}

func (a *App) runQuiet(name string, args ...string) (string, error) {
	return a.runQuietIn("", name, args...)
}
func (a *App) runQuietIn(dir, name string, args ...string) (string, error) {
	return a.runQuietInWithEnv(dir, a.commandEnv(), name, args...)
}
func (a *App) runQuietInWithEnv(dir string, environment []string, name string, args ...string) (string, error) {
	if a.quietHook != nil {
		return a.quietHook(dir, name, args...)
	}
	resolved, err := findExecutable(name)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(a.commandContext(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, resolved, args...)
	cmd.Dir, cmd.Env = dir, environment
	hideCommandWindow(cmd)
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return string(output), fmt.Errorf("命令探测超时或已取消: %w", ctx.Err())
	}
	return string(output), err
}
func (a *App) runCommand(id, name string, args ...string) error {
	return a.runCommandIn(id, "", name, args...)
}
func (a *App) runCommandIn(id, dir, name string, args ...string) error {
	return a.runCommandInWithEnv(id, dir, a.commandEnv(), name, args...)
}
func (a *App) runCommandInWithEnv(id, dir string, environment []string, name string, args ...string) error {
	a.emitLog(id, "command", "$ "+name+" "+strings.Join(redactArgs(args), " "))
	if a.commandHook != nil {
		return a.commandHook(id, dir, name, args...)
	}
	resolved, err := findExecutable(name)
	if err != nil {
		return err
	}
	ctx := a.operations.context(id, a.commandContext())
	cmd := exec.CommandContext(ctx, resolved, args...)
	cmd.Dir, cmd.Env = dir, environment
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
		if ctx.Err() != nil {
			return fmt.Errorf("命令已取消: %w", ctx.Err())
		}
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

func usesGitLabHTTPS(remote RemoteSpec) bool {
	if remote.Platform != PlatformGitLab {
		return false
	}
	u, err := url.Parse(remote.CloneURL)
	return err == nil && strings.EqualFold(u.Scheme, "https") && strings.EqualFold(u.Hostname(), "gitlab.com")
}

func withGitLabCredentialHelper(environment []string, glabDir string) []string {
	overrides := map[string]string{
		"GIT_CONFIG_COUNT":   "1",
		"GIT_CONFIG_KEY_0":   "credential.https://gitlab.com.helper",
		"GIT_CONFIG_VALUE_0": "!glab auth git-credential",
	}
	if glabDir != "" && glabDir != "." {
		path := environmentValue(environment, "PATH")
		if path == "" {
			overrides["PATH"] = glabDir
		} else {
			overrides["PATH"] = glabDir + string(os.PathListSeparator) + path
		}
	}
	return replaceCommandEnv(environment, overrides)
}

func replaceCommandEnv(environment []string, overrides map[string]string) []string {
	result := make([]string, 0, len(environment)+len(overrides))
	for _, entry := range environment {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			if _, replace := findEnvOverride(overrides, key); replace {
				continue
			}
		}
		result = append(result, entry)
	}
	for key, value := range overrides {
		result = append(result, key+"="+value)
	}
	return result
}

func environmentValue(environment []string, key string) string {
	for _, entry := range environment {
		name, value, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(name, key) {
			return value
		}
	}
	return ""
}

func findEnvOverride(overrides map[string]string, key string) (string, bool) {
	for name, value := range overrides {
		if strings.EqualFold(name, key) {
			return value, true
		}
	}
	return "", false
}
