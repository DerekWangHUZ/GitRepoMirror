package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

func (a *App) commandEnv() []string {
	// Inherited GIT_CONFIG_* pairs are dropped: stale or orphaned pairs (a
	// count without matching keys, for example) make every git invocation fail
	// with "unable to parse command-line config", and the app rebuilds the
	// pairs its credential strategy needs on its own.
	environment := dropGitConfigOverrides(os.Environ())
	proxy := a.store.Settings().Proxy
	if proxy != "" {
		environment = append(environment, "HTTP_PROXY="+proxy, "HTTPS_PROXY="+proxy, "ALL_PROXY="+proxy)
	}
	// The Azure CLI writes a per-command log file under %USERPROFILE%\.azure,
	// which fails in restricted-integrity process contexts and aborts the
	// whole command before it runs. The app already captures command output in
	// its own run log, so the CLI-side file log is redundant.
	environment = append(environment, "AZURE_LOGGING_ENABLE_LOG_FILE=false")
	return replaceCommandEnv(environment, map[string]string{
		"GIT_TERMINAL_PROMPT": "0",
		"GCM_INTERACTIVE":     "Never",
	})
}

// commandEnvForRemotes merges the credential strategies of every remote in the
// sync chain instead of taking only the first hit: a Gitee→GitHub or
// GitLab→Gitee mirror needs both platforms' overrides side by side. Duplicate
// config keys collapse so repeated remotes do not re-query the same CLI token.
func (a *App) commandEnvForRemotes(remotes ...RemoteSpec) []string {
	environment := a.commandEnv()
	var entries [][2]string
	var dirs []string
	for _, remote := range remotes {
		remoteEntries, remoteDirs, ok := a.credentialOverridesFor(remote)
		if !ok {
			continue
		}
		entries = append(entries, remoteEntries...)
		dirs = append(dirs, remoteDirs...)
	}
	if len(entries) == 0 {
		return environment
	}
	return withCredentialHelpers(environment, dedupeConfigEntries(entries), dirs...)
}

// dedupeConfigEntries keeps the first entry per config key so merged remote
// strategies cannot emit the same GIT_CONFIG_KEY_n twice.
func dedupeConfigEntries(entries [][2]string) [][2]string {
	seen := make(map[string]bool, len(entries))
	result := make([][2]string, 0, len(entries))
	for _, entry := range entries {
		if seen[entry[0]] {
			continue
		}
		seen[entry[0]] = true
		result = append(result, entry)
	}
	return result
}

// credentialOverridesFor resolves the git-credential environment strategy for
// one remote. GitLab HTTPS remotes reset all helpers and reinstall the glab
// helper; GitHub keeps its own helper so a follow-up push still authenticates.
// Gitee HTTPS remotes receive a basic-auth extra header built from the CLI
// token.
func (a *App) credentialOverridesFor(remote RemoteSpec) ([][2]string, []string, bool) {
	switch remote.Platform {
	case PlatformGitLab:
		if !usesGitLabHTTPS(remote) {
			return nil, nil, false
		}
		return a.gitLabCredentialOverrides()
	case PlatformGitee:
		if !usesGiteeHTTPS(remote) {
			return nil, nil, false
		}
		return a.giteeCredentialOverrides()
	}
	return nil, nil, false
}

func (a *App) gitLabCredentialOverrides() ([][2]string, []string, bool) {
	glabDir := a.executableDir("glab")
	if glabDir == "" {
		return nil, nil, false
	}
	entries := [][2]string{
		{"credential.helper", ""},
		{"credential.https://gitlab.com.helper", "!glab auth git-credential"},
	}
	dirs := []string{glabDir}
	if ghDir := a.executableDir("gh"); ghDir != "" {
		entries = append(entries, [2]string{"credential.https://github.com.helper", "!gh auth git-credential"})
		dirs = append(dirs, ghDir)
	}
	return entries, dirs, true
}

func usesGiteeHTTPS(remote RemoteSpec) bool {
	if remote.Platform != PlatformGitee {
		return false
	}
	parsed, err := url.Parse(remote.CloneURL)
	return err == nil && strings.EqualFold(parsed.Scheme, "https") && strings.EqualFold(parsed.Hostname(), "gitee.com")
}

// resolveExecutable honours the test hook and otherwise falls back to the
// package lookup that also probes the well-known Windows install directories.
func (a *App) resolveExecutable(name string) (string, error) {
	if a.executableHook != nil {
		return a.executableHook(name)
	}
	return findExecutable(name)
}

// executableDir returns the directory holding the tool, or "" when the tool is
// not installed so callers can keep the behaviour of a missing CLI.
func (a *App) executableDir(name string) string {
	path, err := a.resolveExecutable(name)
	if err != nil {
		return ""
	}
	return filepath.Dir(path)
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

// runQuietStdout captures stdout separately from stderr so commands whose
// stderr carries update notices (gitee CLI) do not pollute parsed output.
func (a *App) runQuietStdout(name string, args ...string) (string, error) {
	if a.quietHook != nil {
		return a.quietHook("", name, args...)
	}
	resolved, err := findExecutable(name)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(a.commandContext(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, resolved, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	cmd.Env = a.commandEnv()
	hideCommandWindow(cmd)
	err = cmd.Run()
	if ctx.Err() != nil {
		return stdout.String(), fmt.Errorf("命令探测超时或已取消: %w", ctx.Err())
	}
	if err != nil {
		return stdout.String(), fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
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

// withGitLabCredentialHelper builds the environment shared by the clone and the
// push of one sync. The empty credential.helper entry resets every helper the
// system and the global git config provide, so GitHub needs its own helper as
// well: without it the follow-up push finds no helper at all while
// GIT_TERMINAL_PROMPT=0 has disabled the interactive prompt.
func withGitLabCredentialHelper(environment []string, glabDir, ghDir string) []string {
	entries := [][2]string{
		{"credential.helper", ""},
		{"credential.https://gitlab.com.helper", "!glab auth git-credential"},
	}
	dirs := []string{glabDir}
	if ghDir != "" {
		entries = append(entries, [2]string{"credential.https://github.com.helper", "!gh auth git-credential"})
		dirs = append(dirs, ghDir)
	}
	return withCredentialHelpers(environment, entries, dirs...)
}

// withCredentialHelpers installs GIT_CONFIG credential entries and prepends the
// helper executable directories to PATH.
func withCredentialHelpers(environment []string, entries [][2]string, dirs ...string) []string {
	environment = dropGitConfigOverrides(environment)
	overrides := make(map[string]string, 2*len(entries)+1)
	overrides["GIT_CONFIG_COUNT"] = strconv.Itoa(len(entries))
	for index, entry := range entries {
		overrides[fmt.Sprintf("GIT_CONFIG_KEY_%d", index)] = entry[0]
		overrides[fmt.Sprintf("GIT_CONFIG_VALUE_%d", index)] = entry[1]
	}
	if path, added := pathWithLeadingDirectories(environment, dirs...); added {
		overrides["PATH"] = path
	}
	return replaceCommandEnv(environment, overrides)
}

// dropGitConfigOverrides removes GIT_CONFIG_COUNT together with any inherited
// GIT_CONFIG_KEY_n/GIT_CONFIG_VALUE_n pair, so the count always describes the
// pairs written by withGitLabCredentialHelper.
func dropGitConfigOverrides(environment []string) []string {
	result := make([]string, 0, len(environment))
	for _, entry := range environment {
		if key, _, ok := strings.Cut(entry, "="); ok && isGitConfigEntry(key) {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func isGitConfigEntry(key string) bool {
	name := strings.ToUpper(key)
	return name == "GIT_CONFIG_COUNT" ||
		strings.HasPrefix(name, "GIT_CONFIG_KEY_") ||
		strings.HasPrefix(name, "GIT_CONFIG_VALUE_")
}

// pathWithLeadingDirectories returns PATH with the given directories in front
// and reports whether anything was prepended. Directories already on PATH are
// skipped so repeated syncs do not keep growing the variable.
func pathWithLeadingDirectories(environment []string, dirs ...string) (string, bool) {
	separator := string(os.PathListSeparator)
	existing := environmentValue(environment, "PATH")
	present := strings.Split(existing, separator)
	var leading []string
	for _, dir := range dirs {
		if dir == "" || dir == "." || slices.Contains(present, dir) {
			continue
		}
		leading = append(leading, dir)
		present = append(present, dir)
	}
	if len(leading) == 0 {
		return existing, false
	}
	if existing == "" {
		return strings.Join(leading, separator), true
	}
	return strings.Join(leading, separator) + separator + existing, true
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
