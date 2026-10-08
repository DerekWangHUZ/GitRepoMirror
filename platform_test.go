package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestCloneCommandRouting(t *testing.T) {
	github := RemoteSpec{Platform: PlatformGitHub, CloneURL: "https://github.com/org/repo.git"}
	gitlab := RemoteSpec{Platform: PlatformGitLab, CloneURL: "https://gitlab.com/org/repo.git"}
	generic := RemoteSpec{Platform: PlatformGeneric, CloneURL: "https://code.example/org/repo.git"}
	ready := EnvironmentStatus{
		GitHub: ToolStatus{Installed: true, Authenticated: true},
		GitLab: ToolStatus{Installed: true, Authenticated: true},
	}

	if command := cloneCommand(github, "target", "mirror", ready); command.Name != "gh" || command.Args[len(command.Args)-1] != "--mirror" {
		t.Fatalf("unexpected GitHub command: %#v", command)
	}
	if command := cloneCommand(gitlab, "target", "shallow", ready); command.Name != "glab" || command.Args[len(command.Args)-2] != "--depth=1" {
		t.Fatalf("unexpected GitLab command: %#v", command)
	}
	if command := cloneCommand(generic, "target", "mirror", ready); command.Name != "git" {
		t.Fatalf("unexpected generic command: %#v", command)
	}
	if command := cloneCommand(github, "target", "mirror", EnvironmentStatus{}); command.Name != "git" {
		t.Fatalf("expected git fallback: %#v", command)
	}
}

func TestClassifyPlatformAuthFailure(t *testing.T) {
	tests := []struct {
		name, platform, output, state string
	}{
		{"GitHub invalid token", PlatformGitHub, "The token in default is invalid.", "invalid"},
		{"GitLab missing token", PlatformGitLab, "No access or refresh token was found; No token found", "missing"},
		{"unknown auth failure", PlatformGitLab, "connection reset", "unauthenticated"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, message := classifyPlatformAuthFailure(test.platform, test.output)
			if state != test.state || message == "" {
				t.Fatalf("unexpected classification: %q %q", state, message)
			}
		})
	}
}

func TestPlatformLoginArgsUseWebHTTPS(t *testing.T) {
	tests := []struct {
		platform, host, flow string
	}{
		{PlatformGitHub, "github.com", "--web"},
		{PlatformGitLab, "gitlab.com", "--device"},
	}
	for _, test := range tests {
		args := platformLoginArgs(test.platform, test.host)
		expected := []string{"auth", "login", "--hostname", test.host, test.flow, "--git-protocol", "https"}
		if !reflect.DeepEqual(args, expected) {
			t.Fatalf("unexpected login args: %#v", args)
		}
	}
}

func TestRedactArgs(t *testing.T) {
	redacted := redactArgs([]string{"push", "https://user:token@example.com/team/repo.git"})
	if redacted[1] != "https://%2A%2A%2A@example.com/team/repo.git" {
		t.Fatalf("credential was not redacted: %q", redacted[1])
	}
}

func TestGitLabCredentialEnvironmentUsesGlabHelper(t *testing.T) {
	base := []string{
		"Path=C:\\Windows",
		"GCM_INTERACTIVE=Always",
		"GIT_CONFIG_COUNT=4",
		"GIT_CONFIG_KEY_0=credential.helper",
		"GIT_CONFIG_VALUE_0=manager",
		"GIT_CONFIG_KEY_3=credential.https://example.invalid.helper",
		"GIT_CONFIG_VALUE_3=!stale helper",
	}
	environment := withGitLabCredentialHelper(base, "C:\\Program Files\\glab", "C:\\Program Files\\GitHub CLI")
	if got := environmentValue(environment, "GIT_CONFIG_COUNT"); got != "3" {
		t.Fatalf("unexpected config count: %q", got)
	}
	if got := environmentValue(environment, "GIT_CONFIG_KEY_0"); got != "credential.helper" {
		t.Fatalf("unexpected reset helper key: %q", got)
	}
	if got := environmentValue(environment, "GIT_CONFIG_VALUE_0"); got != "" {
		t.Fatalf("unexpected reset helper value: %q", got)
	}
	if got := environmentValue(environment, "GIT_CONFIG_KEY_1"); got != "credential.https://gitlab.com.helper" {
		t.Fatalf("unexpected config key: %q", got)
	}
	if got := environmentValue(environment, "GIT_CONFIG_VALUE_1"); got != "!glab auth git-credential" {
		t.Fatalf("unexpected config helper: %q", got)
	}
	if got := environmentValue(environment, "GIT_CONFIG_KEY_2"); got != "credential.https://github.com.helper" {
		t.Fatalf("unexpected GitHub config key: %q", got)
	}
	if got := environmentValue(environment, "GIT_CONFIG_VALUE_2"); got != "!gh auth git-credential" {
		t.Fatalf("unexpected GitHub config helper: %q", got)
	}
	if got := environmentValue(environment, "GIT_CONFIG_KEY_3"); got != "" {
		t.Fatalf("stale config pair survived: %q", got)
	}
	if got := environmentValue(environment, "GIT_CONFIG_VALUE_3"); got != "" {
		t.Fatalf("stale config pair survived: %q", got)
	}
	if got := environmentValue(environment, "PATH"); got != "C:\\Program Files\\glab"+string(os.PathListSeparator)+"C:\\Program Files\\GitHub CLI"+string(os.PathListSeparator)+"C:\\Windows" {
		t.Fatalf("CLI directories were not added to PATH: %q", got)
	}
	if got := environmentValue(environment, "GCM_INTERACTIVE"); got != "Always" {
		t.Fatalf("unrelated environment value changed: %q", got)
	}
	assertGitConfigPairsMatchCount(t, environment)
}

func TestGitLabCredentialEnvironmentOmitsGitHubHelperWithoutGH(t *testing.T) {
	base := []string{"Path=C:\\Windows"}
	environment := withGitLabCredentialHelper(base, "C:\\Program Files\\glab", "")
	if got := environmentValue(environment, "GIT_CONFIG_COUNT"); got != "2" {
		t.Fatalf("unexpected config count: %q", got)
	}
	if got := environmentValue(environment, "GIT_CONFIG_KEY_2"); got != "" {
		t.Fatalf("GitHub helper was injected without gh: %q", got)
	}
	if got := environmentValue(environment, "GIT_CONFIG_VALUE_2"); got != "" {
		t.Fatalf("GitHub helper was injected without gh: %q", got)
	}
	if got := environmentValue(environment, "PATH"); got != "C:\\Program Files\\glab"+string(os.PathListSeparator)+"C:\\Windows" {
		t.Fatalf("glab directory was not added to PATH: %q", got)
	}
	assertGitConfigPairsMatchCount(t, environment)
}

func TestGitLabCredentialEnvironmentKeepsPathWhenGitHubCLIIsOnPath(t *testing.T) {
	separator := string(os.PathListSeparator)
	ghDir := "C:\\Program Files\\GitHub CLI"
	original := "C:\\Windows" + separator + ghDir
	environment := withGitLabCredentialHelper([]string{"Path=" + original}, "C:\\Users\\tester\\bin", ghDir)
	if got := environmentValue(environment, "GIT_CONFIG_COUNT"); got != "3" {
		t.Fatalf("unexpected config count: %q", got)
	}
	want := "C:\\Users\\tester\\bin" + separator + original
	if got := environmentValue(environment, "PATH"); got != want {
		t.Fatalf("unexpected PATH: %q, want %q", got, want)
	}
	if occurrences := strings.Count(environmentValue(environment, "PATH"), ghDir); occurrences != 1 {
		t.Fatalf("gh directory appears %d times on PATH", occurrences)
	}
	assertGitConfigPairsMatchCount(t, environment)
}

func TestCommandEnvForRemotesKeepsGitHubCredentialForGitLabSource(t *testing.T) {
	app := NewApp()
	app.executableHook = fakeExecutableLookup()
	environment := app.commandEnvForRemotes(
		RemoteSpec{Platform: PlatformGitLab, CloneURL: "https://gitlab.com/team/source.git"},
		RemoteSpec{Platform: PlatformGitHub, CloneURL: "https://github.com/team/target.git"},
	)
	if got := environmentValue(environment, "GIT_CONFIG_VALUE_2"); got != "!gh auth git-credential" {
		t.Fatalf("GitHub helper is missing for a GitLab to GitHub sync: %q", got)
	}
	if got := environmentValue(environment, "GIT_CONFIG_KEY_2"); got != "credential.https://github.com.helper" {
		t.Fatalf("GitHub helper key is missing: %q", got)
	}
	if got := environmentValue(environment, "GIT_CONFIG_VALUE_1"); got != "!glab auth git-credential" {
		t.Fatalf("GitLab helper was lost: %q", got)
	}
	if got := environmentValue(environment, "GIT_TERMINAL_PROMPT"); got != "0" {
		t.Fatalf("terminal prompt setting was lost: %q", got)
	}
	assertGitConfigPairsMatchCount(t, environment)
}

func TestCommandEnvForRemotesSkipsHelpersWithoutGlab(t *testing.T) {
	app := NewApp()
	app.executableHook = func(name string) (string, error) {
		if name == "gh" {
			return "C:\\Program Files\\GitHub CLI\\gh.exe", nil
		}
		return "", errors.New("未找到命令: glab")
	}
	environment := app.commandEnvForRemotes(RemoteSpec{Platform: PlatformGitLab, CloneURL: "https://gitlab.com/team/source.git"})
	for _, entry := range environment {
		if key, _, ok := strings.Cut(entry, "="); ok && isGitConfigEntry(key) {
			t.Fatalf("credential helpers were injected without glab: %q", entry)
		}
	}
}

func TestGitConfigOverridesAreAcceptedByGit(t *testing.T) {
	git, err := findExecutable("git")
	if err != nil {
		t.Skip("Git is not installed in this environment")
	}
	app := NewApp()
	app.executableHook = fakeExecutableLookup()
	environment := app.commandEnvForRemotes(RemoteSpec{Platform: PlatformGitLab, CloneURL: "https://gitlab.com/team/source.git"})
	for _, key := range []string{"credential.https://gitlab.com.helper", "credential.https://github.com.helper"} {
		command := exec.Command(git, "config", "--get-all", key)
		command.Env = environment
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("git rejected the environment for %q: %v\n%s", key, err, output)
		}
		if strings.TrimSpace(string(output)) == "" {
			t.Fatalf("git resolved no helper for %q", key)
		}
	}
}

func fakeExecutableLookup() func(string) (string, error) {
	paths := map[string]string{
		"glab": "C:\\Program Files\\glab\\glab.exe",
		"gh":   "C:\\Program Files\\GitHub CLI\\gh.exe",
	}
	return func(name string) (string, error) {
		if path, ok := paths[name]; ok {
			return path, nil
		}
		return "", errors.New("未找到命令: " + name)
	}
}

// assertGitConfigPairsMatchCount fails when GIT_CONFIG_COUNT no longer matches
// the GIT_CONFIG_KEY_n/GIT_CONFIG_VALUE_n pairs Git would actually read.
func assertGitConfigPairsMatchCount(t *testing.T, environment []string) {
	t.Helper()
	count := 0
	if value := environmentValue(environment, "GIT_CONFIG_COUNT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			t.Fatalf("GIT_CONFIG_COUNT is not a number: %q", value)
		}
		count = parsed
	}
	keys, values := gitConfigPairs(environment)
	if len(keys) != count || len(values) != count {
		t.Fatalf("GIT_CONFIG_COUNT=%d does not match %d keys and %d values", count, len(keys), len(values))
	}
	for index := 0; index < count; index++ {
		if !slices.Contains(keys, fmt.Sprintf("GIT_CONFIG_KEY_%d", index)) {
			t.Fatalf("GIT_CONFIG_KEY_%d is missing: %#v", index, keys)
		}
		if !slices.Contains(values, fmt.Sprintf("GIT_CONFIG_VALUE_%d", index)) {
			t.Fatalf("GIT_CONFIG_VALUE_%d is missing: %#v", index, values)
		}
	}
}

func gitConfigPairs(environment []string) ([]string, []string) {
	var keys, values []string
	for _, entry := range environment {
		name, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		switch {
		case strings.HasPrefix(strings.ToUpper(name), "GIT_CONFIG_KEY_"):
			keys = append(keys, name)
		case strings.HasPrefix(strings.ToUpper(name), "GIT_CONFIG_VALUE_"):
			values = append(values, name)
		}
	}
	return keys, values
}

func TestUsesGitLabHTTPSOnlyForGitLabHTTPSRemotes(t *testing.T) {
	tests := []struct {
		name   string
		remote RemoteSpec
		want   bool
	}{
		{
			name:   "gitlab https",
			remote: RemoteSpec{Platform: PlatformGitLab, CloneURL: "https://gitlab.com/team/repo.git"},
			want:   true,
		},
		{
			name:   "gitlab ssh",
			remote: RemoteSpec{Platform: PlatformGitLab, CloneURL: "ssh://git@gitlab.com/team/repo.git"},
			want:   false,
		},
		{
			name:   "github https",
			remote: RemoteSpec{Platform: PlatformGitHub, CloneURL: "https://github.com/team/repo.git"},
			want:   false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := usesGitLabHTTPS(test.remote); got != test.want {
				t.Fatalf("usesGitLabHTTPS(%#v) = %v, want %v", test.remote, got, test.want)
			}
		})
	}
}

func TestCommandEnvDisablesInteractiveCredentialPrompts(t *testing.T) {
	app := NewApp()
	if got := environmentValue(app.commandEnv(), "GIT_TERMINAL_PROMPT"); got != "0" {
		t.Fatalf("unexpected terminal prompt setting: %q", got)
	}
	if got := environmentValue(app.commandEnv(), "GCM_INTERACTIVE"); got != "Never" {
		t.Fatalf("unexpected credential manager setting: %q", got)
	}
}

func TestEnsureManagedRepositoryCreatesWithPlatformCLI(t *testing.T) {
	app := NewApp()
	var commands [][]string
	app.quietHook = func(_ string, name string, args ...string) (string, error) {
		commands = append(commands, append([]string{name}, args...))
		switch {
		case name == "gh" && len(args) > 1 && args[0] == "repo" && args[1] == "view":
			return "GraphQL: Could not resolve to a Repository", errors.New("exit status 1")
		case name == "gh" && reflect.DeepEqual(args[len(args)-2:], []string{"--jq", ".clone_url"}):
			return "https://github.com/team/repo.git\n", nil
		case name == "gh" && reflect.DeepEqual(args[len(args)-2:], []string{"--jq", ".html_url"}):
			return "https://github.com/team/repo\n", nil
		default:
			return "", nil
		}
	}
	app.commandHook = func(_ string, _ string, name string, args ...string) error {
		commands = append(commands, append([]string{name}, args...))
		return nil
	}
	remote, visibility, err := app.ensureManagedRepository("id", PlatformGitHub, "team", "repo", "private", "error")
	if err != nil {
		t.Fatal(err)
	}
	if visibility != "private" || remote.CloneURL != "https://github.com/team/repo.git" {
		t.Fatalf("unexpected result: %#v %s", remote, visibility)
	}
	foundCreate := false
	for _, command := range commands {
		if reflect.DeepEqual(command, []string{"gh", "repo", "create", "team/repo", "--private"}) {
			foundCreate = true
		}
	}
	if !foundCreate {
		t.Fatalf("create command not recorded: %#v", commands)
	}
}

func TestEnsureManagedRepositoryAssociatesWithoutChangingVisibility(t *testing.T) {
	app := NewApp()
	app.quietHook = func(_ string, _ string, args ...string) (string, error) {
		if reflect.DeepEqual(args, []string{"api", "projects/team%2Frepo"}) {
			return `{"visibility":"public","http_url_to_repo":"https://gitlab.com/team/repo.git","web_url":"https://gitlab.com/team/repo"}`, nil
		}
		return "", nil
	}
	app.commandHook = func(_ string, _ string, _ string, _ ...string) error {
		t.Fatal("associate must not create a repository")
		return nil
	}
	remote, visibility, err := app.ensureManagedRepository("id", PlatformGitLab, "team", "repo", "private", "associate")
	if err != nil {
		t.Fatal(err)
	}
	if remote.Platform != PlatformGitLab || visibility != "public" {
		t.Fatalf("unexpected association: %#v %s", remote, visibility)
	}
}

func TestManagedRepositoryLookupDoesNotTreatNetworkErrorAsMissing(t *testing.T) {
	app := NewApp()
	app.quietHook = func(_ string, _ string, _ ...string) (string, error) {
		return "connection reset", errors.New("exit status 1")
	}
	if _, _, err := app.managedRepositoryVisibility(PlatformGitHub, "team/repo"); err == nil {
		t.Fatal("expected lookup error")
	}
}

func TestGitOnlyTargetCannotBeDeletedRemotely(t *testing.T) {
	app := NewApp()
	data := app.store.Snapshot()
	data.Repositories = []Repository{{ID: "one", ManagementMode: ManagementGitOnly}}
	app.store.Replace(data)
	if err := app.RemoveRepository("one", true); err == nil {
		t.Fatal("expected remote deletion to be rejected")
	}
}
