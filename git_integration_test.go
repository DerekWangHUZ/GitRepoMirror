package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMirrorSyncWithLocalGitRepositories(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	source := filepath.Join(root, "source")
	target := filepath.Join(root, "target.git")
	runGitTestCommand(t, git, root, "init", source)
	runGitTestCommand(t, git, root, "init", "--bare", target)
	runGitTestCommand(t, git, source, "config", "user.name", "GitRepoMirror Test")
	runGitTestCommand(t, git, source, "config", "user.email", "test@example.invalid")
	runGitTestCommand(t, git, source, "commit", "--allow-empty", "-m", "initial")

	app := NewApp()
	app.events = &recordingEventSink{}
	repository := Repository{
		ID: "integration", Mode: "mirror",
		Source: RemoteSpec{Platform: PlatformGeneric, CloneURL: source},
		Target: RemoteSpec{Platform: PlatformGeneric, CloneURL: target},
	}
	if err := app.performSync(repository, false, EnvironmentStatus{}); err != nil {
		t.Fatal(err)
	}
	output := runGitTestCommand(t, git, root, "--git-dir", target, "show-ref", "--heads")
	if !strings.Contains(output, "refs/heads/") {
		t.Fatalf("target repository has no mirrored branch: %s", output)
	}
}

func runGitTestCommand(t *testing.T, git, dir string, args ...string) string {
	t.Helper()
	command := exec.Command(git, args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
	return string(output)
}
