package main

import (
	"reflect"
	"testing"
)

func TestMirrorPushCommandAvoidsForceForGitLab(t *testing.T) {
	tests := []struct {
		name   string
		target RemoteSpec
		want   []string
	}{
		{
			name:   "GitLab protected branches",
			target: RemoteSpec{Platform: PlatformGitLab, CloneURL: "https://gitlab.com/team/backup.git"},
			want:   []string{"push", "--prune", "https://gitlab.com/team/backup.git", "refs/heads/*:refs/heads/*", "refs/tags/*:refs/tags/*"},
		},
		{
			name:   "GitHub keeps force mirror",
			target: RemoteSpec{Platform: PlatformGitHub, CloneURL: "https://github.com/team/backup.git"},
			want:   []string{"push", "--force", "--prune", "https://github.com/team/backup.git", "+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := mirrorPushCommand(test.target)
			if got.Name != "git" || !reflect.DeepEqual(got.Args, test.want) {
				t.Fatalf("mirrorPushCommand(%#v) = %#v, want %#v", test.target, got, test.want)
			}
		})
	}
}

func TestShallowPushCommandAvoidsForceForGitLab(t *testing.T) {
	gitlab := shallowPushCommand(RemoteSpec{Platform: PlatformGitLab, CloneURL: "https://gitlab.com/team/backup.git"}, "main")
	if !reflect.DeepEqual(gitlab.Args, []string{"push", "https://gitlab.com/team/backup.git", "HEAD:refs/heads/main"}) {
		t.Fatalf("unexpected GitLab shallow push: %#v", gitlab)
	}
	github := shallowPushCommand(RemoteSpec{Platform: PlatformGitHub, CloneURL: "https://github.com/team/backup.git"}, "main")
	if !reflect.DeepEqual(github.Args, []string{"push", "--force", "https://github.com/team/backup.git", "HEAD:refs/heads/main"}) {
		t.Fatalf("unexpected GitHub shallow push: %#v", github)
	}
}
