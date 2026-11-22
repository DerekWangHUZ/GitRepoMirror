package main

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestMirrorPushCommandForceModes(t *testing.T) {
	tests := []struct {
		name   string
		target RemoteSpec
		force  bool
		prune  bool
		want   []string
	}{
		{
			name:   "GitLab fallback without API permission",
			target: RemoteSpec{Platform: PlatformGitLab, CloneURL: "https://gitlab.com/team/backup.git"},
			force:  false,
			want:   []string{"push", "https://gitlab.com/team/backup.git", "refs/heads/*:refs/heads/*", "refs/tags/*:refs/tags/*"},
		},
		{
			name:   "GitLab full mirror after temporary permission",
			target: RemoteSpec{Platform: PlatformGitLab, CloneURL: "https://gitlab.com/team/backup.git"},
			force:  true,
			want:   []string{"push", "--force", "https://gitlab.com/team/backup.git", "+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*"},
		},
		{
			name:   "GitHub keeps force mirror",
			target: RemoteSpec{Platform: PlatformGitHub, CloneURL: "https://github.com/team/backup.git"},
			force:  false,
			want:   []string{"push", "--force", "https://github.com/team/backup.git", "+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*"},
		},
		{
			name:   "GitLab strict mirror prunes extra refs",
			target: RemoteSpec{Platform: PlatformGitLab, CloneURL: "https://gitlab.com/team/backup.git"},
			force:  false,
			prune:  true,
			want:   []string{"push", "--prune", "https://gitlab.com/team/backup.git", "refs/heads/*:refs/heads/*", "refs/tags/*:refs/tags/*"},
		},
		{
			name:   "GitHub strict mirror forces and prunes",
			target: RemoteSpec{Platform: PlatformGitHub, CloneURL: "https://github.com/team/backup.git"},
			force:  false,
			prune:  true,
			want:   []string{"push", "--force", "--prune", "https://github.com/team/backup.git", "+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := mirrorPushCommand(test.target, test.force, test.prune)
			if got.Name != "git" || !reflect.DeepEqual(got.Args, test.want) {
				t.Fatalf("mirrorPushCommand(%#v) = %#v, want %#v", test.target, got, test.want)
			}
		})
	}
}

func TestShallowPushCommandForceModes(t *testing.T) {
	gitlab := shallowPushCommand(RemoteSpec{Platform: PlatformGitLab, CloneURL: "https://gitlab.com/team/backup.git"}, "main", false)
	if !reflect.DeepEqual(gitlab.Args, []string{"push", "https://gitlab.com/team/backup.git", "HEAD:refs/heads/main"}) {
		t.Fatalf("unexpected GitLab shallow push: %#v", gitlab)
	}
	gitlabForce := shallowPushCommand(RemoteSpec{Platform: PlatformGitLab, CloneURL: "https://gitlab.com/team/backup.git"}, "main", true)
	if !reflect.DeepEqual(gitlabForce.Args, []string{"push", "--force", "https://gitlab.com/team/backup.git", "HEAD:refs/heads/main"}) {
		t.Fatalf("unexpected forced GitLab shallow push: %#v", gitlabForce)
	}
	github := shallowPushCommand(RemoteSpec{Platform: PlatformGitHub, CloneURL: "https://github.com/team/backup.git"}, "main", false)
	if !reflect.DeepEqual(github.Args, []string{"push", "--force", "https://github.com/team/backup.git", "HEAD:refs/heads/main"}) {
		t.Fatalf("unexpected GitHub shallow push: %#v", github)
	}
}

func TestProtectedBranchRuleMatchesGitLabWildcards(t *testing.T) {
	tests := []struct {
		rule, branch string
		want         bool
	}{
		{rule: "main", branch: "main", want: true},
		{rule: "main", branch: "develop", want: false},
		{rule: "release/*", branch: "release/v1", want: true},
		{rule: "release/*", branch: "feature/release/v1", want: false},
		{rule: "*", branch: "feature/release/v1", want: true},
	}
	for _, test := range tests {
		if got := protectedBranchRuleMatches(test.rule, test.branch); got != test.want {
			t.Fatalf("protectedBranchRuleMatches(%q, %q) = %v, want %v", test.rule, test.branch, got, test.want)
		}
	}
}

func TestPrepareGitLabForcePushRestoresProtection(t *testing.T) {
	app := NewApp()
	var commands [][]string
	app.quietHook = func(_ string, name string, args ...string) (string, error) {
		commands = append(commands, append([]string{name}, args...))
		if reflect.DeepEqual(args, []string{"api", "projects/team%2Frepo/protected_branches?per_page=100"}) {
			return `[{"name":"main","allow_force_push":false}]`, nil
		}
		return "", nil
	}
	repo := Repository{
		ID:             "gitlab",
		ManagementMode: ManagementCLI,
		Target: RemoteSpec{
			Platform:   PlatformGitLab,
			Namespace:  "team",
			Repository: "repo",
		},
	}
	restore, force, err := app.prepareForcePush(repo, []string{"main"}, true)
	if err != nil || !force {
		t.Fatalf("prepareGitLabForcePush() = restore=%v force=%v err=%v", restore != nil, force, err)
	}
	if err := restore(); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"glab", "api", "projects/team%2Frepo/protected_branches?per_page=100"},
		{"glab", "api", "--method", "PATCH", "projects/team%2Frepo/protected_branches/main?allow_force_push=true"},
		{"glab", "api", "--method", "PATCH", "projects/team%2Frepo/protected_branches/main?allow_force_push=false"},
	}
	if !reflect.DeepEqual(commands, want) {
		t.Fatalf("unexpected GitLab API calls: %#v", commands)
	}
}

// restoreWarningHooks wires an app whose GitLab protection restore fails while
// everything else succeeds: for-each-ref reports "main", the relax PATCH works
// and the restore PATCH is rejected.
func restoreWarningHooks(app *App) {
	app.quietHook = func(_ string, name string, args ...string) (string, error) {
		switch {
		case name == "git" && len(args) > 0 && args[0] == "for-each-ref":
			return "main\n", nil
		case name == "glab" && reflect.DeepEqual(args, []string{"api", "projects/team%2Frepo/protected_branches?per_page=100"}):
			return `[{"name":"main","allow_force_push":false}]`, nil
		case name == "glab" && len(args) == 4 && strings.HasSuffix(args[3], "allow_force_push=true"):
			return "", nil
		case name == "glab" && len(args) == 4 && strings.HasSuffix(args[3], "allow_force_push=false"):
			return "forbidden", errors.New("exit status 1")
		}
		return "", nil
	}
	app.commandHook = func(_ string, _ string, _ string, _ ...string) error { return nil }
}

func restoreWarningRepository(id string) Repository {
	return Repository{
		ID:             id,
		Mode:           "mirror",
		ManagementMode: ManagementCLI,
		Source:         RemoteSpec{Platform: PlatformGeneric, CloneURL: "https://code.example/source.git"},
		Target:         RemoteSpec{Platform: PlatformGitLab, Namespace: "team", Repository: "repo", CloneURL: "https://gitlab.com/team/repo.git"},
	}
}

func TestPerformSyncMarksPushDoneWhenRestoreFails(t *testing.T) {
	app := NewApp()
	events := &recordingEventSink{}
	app.events = events
	restoreWarningHooks(app)
	err := app.performSync(restoreWarningRepository("restore"), false, EnvironmentStatus{})
	if err == nil || !strings.Contains(err.Error(), "镜像已推送") {
		t.Fatalf("expected restore warning error, got %v", err)
	}
	if !errors.Is(err, errRestoreWarning) {
		t.Fatalf("error must carry the restore warning sentinel: %v", err)
	}
	for _, event := range events.progress {
		if event.Step == 3 && event.Status == "failed" {
			t.Fatalf("restore failure must not mark the push failed: %#v", events.progress)
		}
	}
	found := false
	for _, event := range events.progress {
		if event.Step == 3 && event.Status == "done" {
			found = true
		}
	}
	if !found {
		t.Fatalf("successful push must emit step-3 done: %#v", events.progress)
	}
}

func TestSyncRepositoryKeepsHealthyWhenRestoreFails(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	app := NewApp()
	restoreWarningHooks(app)
	data := app.store.Snapshot()
	data.Repositories = []Repository{restoreWarningRepository("restore")}
	app.store.Replace(data)
	err := app.SyncRepository("restore")
	if err == nil || !errors.Is(err, errRestoreWarning) {
		t.Fatalf("expected restore warning, got %v", err)
	}
	repository, _ := app.repositoryByID("restore")
	if repository.Status != "healthy" || repository.LastError != "" || repository.LastSync.IsZero() {
		t.Fatalf("restore warning must keep the record healthy: %#v", repository)
	}
}

func TestGitOnlyGitLabTargetSkipsProtectionNegotiation(t *testing.T) {
	app := NewApp()
	events := &recordingEventSink{}
	app.events = events
	var quietCommands [][]string
	app.quietHook = func(_ string, name string, args ...string) (string, error) {
		quietCommands = append(quietCommands, append([]string{name}, args...))
		return "", nil
	}
	var commands [][]string
	app.commandHook = func(_ string, _ string, name string, args ...string) error {
		commands = append(commands, append([]string{name}, args...))
		return nil
	}
	repo := Repository{
		ID:             "gitonly",
		Mode:           "mirror",
		ManagementMode: ManagementGitOnly,
		Source:         RemoteSpec{Platform: PlatformGeneric, CloneURL: "https://code.example/source.git"},
		Target:         RemoteSpec{Platform: PlatformGitLab, CloneURL: "https://gitlab.com/team/repo.git"},
	}
	if err := app.performSync(repo, false, EnvironmentStatus{}); err != nil {
		t.Fatal(err)
	}
	for _, command := range quietCommands {
		if command[0] == "glab" {
			t.Fatalf("git-only target must not query the GitLab API: %#v", quietCommands)
		}
	}
	push := pushCommandArgs(commands)
	if push == nil || !slices.Contains(push, "--force") {
		t.Fatalf("git-only target must fall back to a forced push: %#v", push)
	}
}
