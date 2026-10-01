package main

import (
	"reflect"
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
	restore, force, err := app.prepareGitLabForcePush(repo, []string{"main"})
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
