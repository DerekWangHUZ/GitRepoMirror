package main

import (
	"encoding/base64"
	"errors"
	"reflect"
	"strings"
	"testing"
)

var errTestNotFound = errors.New("未找到命令")

func TestCloneCommandFallsBackToGitForNewPlatforms(t *testing.T) {
	ready := EnvironmentStatus{
		GitHub:    ToolStatus{Installed: true, Authenticated: true},
		GitLab:    ToolStatus{Installed: true, Authenticated: true},
		Gitee:     ToolStatus{Installed: true, Authenticated: true},
		Codeberg:  ToolStatus{Installed: true, Authenticated: true},
		AzureDevOps: ToolStatus{Installed: true, Authenticated: true},
	}
	remote := RemoteSpec{Platform: PlatformGitee, CloneURL: "https://gitee.com/team/repo.git"}
	command := cloneCommand(remote, "target", "mirror", ready)
	if command.Name != "git" || !reflect.DeepEqual(command.Args[:2], []string{"clone", "--mirror"}) {
		t.Fatalf("unexpected gitee clone command: %#v", command)
	}
	azure := RemoteSpec{Platform: PlatformAzureDevOps, CloneURL: "https://dev.azure.com/org/proj/_git/repo"}
	if command := cloneCommand(azure, "target", "shallow", ready); command.Name != "git" {
		t.Fatalf("unexpected azure clone command: %#v", command)
	}
}

func TestCheckPlatformToolGiteeUsesAPIUser(t *testing.T) {
	app := NewApp()
	app.executableHook = func(name string) (string, error) {
		if name == "gitee" {
			return "C:\\gitee\\gitee.cmd", nil
		}
		return "", errTestNotFound
	}
	app.quietHook = func(_ string, name string, args ...string) (string, error) {
		if name != "gitee" {
			t.Fatalf("unexpected command: %s %v", name, args)
		}
		if reflect.DeepEqual(args, []string{"api", "user"}) {
			return `{"login":"gitee-user","name":"n"}`, nil
		}
		return "", errTestNotFound
	}
	status := app.checkPlatformTool(platformByID(PlatformGitee))
	if !status.Installed || !status.Authenticated || status.Login != "gitee-user" {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestCheckPlatformToolCodebergParsesWhoami(t *testing.T) {
	app := NewApp()
	app.executableHook = func(name string) (string, error) {
		if name == "fj" {
			return "C:\\fj\\fj.exe", nil
		}
		return "", errTestNotFound
	}
	app.quietHook = func(_ string, name string, args ...string) (string, error) {
		if name != "fj" || !reflect.DeepEqual(args, []string{"-H", "codeberg.org", "whoami"}) {
			t.Fatalf("unexpected command: %s %v", name, args)
		}
		return "currently signed into cb-user@codeberg.org", nil
	}
	status := app.checkPlatformTool(platformByID(PlatformCodeberg))
	if !status.Authenticated || status.Login != "cb-user" {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestCheckPlatformToolAzureReadsAccountName(t *testing.T) {
	app := NewApp()
	app.executableHook = func(name string) (string, error) {
		if name == "az" {
			return "C:\\az\\az.cmd", nil
		}
		return "", errTestNotFound
	}
	app.quietHook = func(_ string, name string, args ...string) (string, error) {
		if name != "az" || !reflect.DeepEqual(args, []string{"account", "show", "--query", "user.name", "--output", "tsv", "--only-show-errors"}) {
			t.Fatalf("unexpected command: %s %v", name, args)
		}
		return "user@example.com\n", nil
	}
	status := app.checkPlatformTool(platformByID(PlatformAzureDevOps))
	if !status.Authenticated || status.Login != "user@example.com" {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestParseForgejoLoginHandlesLocalizedOutput(t *testing.T) {
	cases := map[string]string{
		"currently signed into cb-user@codeberg.org":        "cb-user",
		"当前已登录 cb-user@codeberg.org":                        "cb-user",
		"Momentan angemeldet als: cb-user@codeberg.org":     "cb-user",
		"Error: api error: token was already used":          "",
	}
	for output, want := range cases {
		if got := parseForgejoLogin(output); got != want {
			t.Fatalf("parseForgejoLogin(%q) = %q, want %q", output, got, want)
		}
	}
}

func TestResolveTargetGiteeCreatesWithCLI(t *testing.T) {
	app := NewApp()
	var commands [][]string
	lookupCount := 0
	app.quietHook = func(_ string, name string, args ...string) (string, error) {
		commands = append(commands, append([]string{name}, args...))
		if reflect.DeepEqual(args, []string{"api", "/repos/team/repo"}) {
			lookupCount++
			if lookupCount == 1 {
				return "404 Not Found", errors.New("exit status 1")
			}
			return `{"full_name":"team/repo","private":true,"html_url":"https://gitee.com/team/repo"}`, nil
		}
		return "", nil
	}
	app.commandHook = func(_ string, _ string, name string, args ...string) error {
		commands = append(commands, append([]string{name}, args...))
		return nil
	}
	remote, management, visibility, err := app.resolveTarget("id", MirrorRequest{
		TargetPlatform: PlatformGitee,
		TargetName:     "repo",
		TargetNamespace: "team",
		Visibility:     "private",
		ConflictPolicy: "error",
	}, RemoteSpec{}, EnvironmentStatus{Gitee: ToolStatus{Installed: true, Authenticated: true, Login: "gitee-user"}})
	if err != nil {
		t.Fatal(err)
	}
	if management != ManagementCLI || visibility != "private" || remote.CloneURL != "https://gitee.com/team/repo.git" {
		t.Fatalf("unexpected result: %#v %s %s", remote, management, visibility)
	}
	found := false
	for _, command := range commands {
		if reflect.DeepEqual(command, []string{"gitee", "repo", "create", "-n", "repo", "--private"}) {
			found = true
		}
	}
	if !found {
		t.Fatalf("gitee create command missing: %#v", commands)
	}
}

func TestGiteeCredentialOverridesInjectExtraHeader(t *testing.T) {
	app := NewApp()
	app.executableHook = func(name string) (string, error) {
		if name == "gitee" {
			return "C:\\gitee\\gitee.cmd", nil
		}
		return "", errTestNotFound
	}
	app.quietHook = func(_ string, name string, args ...string) (string, error) {
		switch {
		case name == "gitee" && reflect.DeepEqual(args, []string{"auth", "token"}):
			return "token-value-123\n", nil
		case name == "gitee" && reflect.DeepEqual(args, []string{"api", "user"}):
			return `{"login":"gitee-user"}`, nil
		}
		return "", errTestNotFound
	}
	entries, dirs, ok := app.credentialOverridesFor(RemoteSpec{
		Platform: PlatformGitee, CloneURL: "https://gitee.com/team/repo.git",
	})
	if !ok || len(entries) != 1 || len(dirs) != 0 {
		t.Fatalf("unexpected overrides: %#v %#v %v", entries, dirs, ok)
	}
	if !strings.HasPrefix(entries[0][0], "http.https://gitee.com/.extraheader") {
		t.Fatalf("unexpected config key: %q", entries[0][0])
	}
	if !strings.Contains(entries[0][1], "Basic ") {
		t.Fatalf("unexpected auth header: %q", entries[0][1])
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(entries[0][1], "Authorization: Basic "))
	if err != nil {
		t.Fatalf("header is not valid basic auth: %v", err)
	}
	if string(decoded) != "gitee-user:token-value-123" {
		t.Fatalf("unexpected basic auth payload: %q", decoded)
	}
	if _, _, ok := app.credentialOverridesFor(RemoteSpec{
		Platform: PlatformGitee, CloneURL: "git@gitee.com:team/repo.git",
	}); ok {
		t.Fatal("SSH remotes must not receive the HTTPS extra header")
	}
}

func TestValidateManagedTargetAcceptsNewPlatforms(t *testing.T) {
	valid := []struct{ platform, namespace, name, visibility string }{
		{PlatformGitee, "team", "repo", "private"},
		{PlatformCodeberg, "team", "repo", "public"},
		{PlatformAzureDevOps, "org/project", "repo", "private"},
	}
	for _, item := range valid {
		if err := validateManagedTarget(item.platform, item.namespace, item.name, item.visibility); err != nil {
			t.Fatalf("unexpected error for %#v: %v", item, err)
		}
	}
	if err := validateManagedTarget(PlatformAzureDevOps, "org", "repo", "private"); err == nil {
		t.Fatal("expected azure namespace without project to be rejected")
	}
	if err := validateManagedTarget(PlatformGitee, "team", "repo", "internal"); err == nil {
		t.Fatal("expected gitee internal visibility to be rejected")
	}
}

func TestParseAzureDevOpsPathNormalizesLayouts(t *testing.T) {
	url, err := parseAzureDevOpsPath([]string{"org", "project", "_git", "repo"})
	if err != nil || !reflect.DeepEqual(url, []string{"org", "project", "repo"}) {
		t.Fatalf("unexpected URL path: %#v %v", url, err)
	}
	scp, err := parseAzureDevOpsPath([]string{"v3", "org", "project", "repo"})
	if err != nil || !reflect.DeepEqual(scp, []string{"org", "project", "repo"}) {
		t.Fatalf("unexpected scp path: %#v %v", scp, err)
	}
	if _, err := parseAzureDevOpsPath([]string{"org", "repo"}); err == nil {
		t.Fatal("expected two-segment path to be rejected")
	}
}

func TestAzureRemoteParsing(t *testing.T) {
	remote, err := parseRemote("https://dev.azure.com/org/project/_git/example")
	if err != nil {
		t.Fatal(err)
	}
	if remote.Namespace != "org/project" || remote.Repository != "example" {
		t.Fatalf("unexpected remote: %#v", remote)
	}
	if remote.WebURL != "https://dev.azure.com/org/project/_git/example" {
		t.Fatalf("unexpected web URL: %q", remote.WebURL)
	}
}
