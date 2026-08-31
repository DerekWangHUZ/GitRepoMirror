package main

import (
	"os"
	"strings"
	"testing"
)

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

func TestWindowsManifestEnablesPerMonitorV2DPI(t *testing.T) {
	payload, err := os.ReadFile("build/windows/wails.exe.manifest")
	if err != nil {
		t.Fatal(err)
	}
	manifest := string(payload)
	for _, expected := range []string{"true/pm", "permonitorv2,permonitor"} {
		if !strings.Contains(manifest, expected) {
			t.Fatalf("Windows manifest is missing %q DPI awareness", expected)
		}
	}
}
