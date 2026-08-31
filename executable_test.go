package main

import "testing"

func TestFindGitExecutable(t *testing.T) {
	if _, err := findExecutable("git"); err != nil {
		t.Skip("Git is not installed in this environment")
	}
}

func TestFindExecutableRejectsUnknownTool(t *testing.T) {
	if _, err := findExecutable("unsupported-tool-name"); err == nil {
		t.Fatal("expected unsupported tool to fail")
	}
}
