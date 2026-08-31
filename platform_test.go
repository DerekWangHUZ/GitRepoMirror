package main

import (
	"errors"
	"reflect"
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
