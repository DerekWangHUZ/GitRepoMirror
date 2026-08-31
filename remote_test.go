package main

import "testing"

func TestParseRemote(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		platform   string
		host       string
		namespace  string
		repository string
	}{
		{"github shorthand", "octo/example", PlatformGitHub, "github.com", "octo", "example"},
		{"github https", "https://github.com/octo/example.git", PlatformGitHub, "github.com", "octo", "example"},
		{"github scp", "git@github.com:octo/example.git", PlatformGitHub, "github.com", "octo", "example"},
		{"gitlab subgroup", "https://gitlab.com/team/subgroup/example.git", PlatformGitLab, "gitlab.com", "team/subgroup", "example"},
		{"gitlab ssh", "ssh://git@gitlab.com/team/example.git", PlatformGitLab, "gitlab.com", "team", "example"},
		{"generic https", "https://codeberg.org/team/example.git", PlatformGeneric, "codeberg.org", "team", "example"},
		{"generic scp", "git@code.example:team/example.git", PlatformGeneric, "code.example", "team", "example"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			remote, err := parseRemote(test.input)
			if err != nil {
				t.Fatalf("parseRemote(%q): %v", test.input, err)
			}
			if remote.Platform != test.platform || remote.Host != test.host || remote.Namespace != test.namespace || remote.Repository != test.repository {
				t.Fatalf("unexpected remote: %#v", remote)
			}
		})
	}
}

func TestParseRemoteRejectsUnsafeInput(t *testing.T) {
	inputs := []string{
		"", "owner", "owner/repo/extra", "file:///tmp/repo", "http://host/team/repo",
		"https://example.com/team/repo?token=secret", "git@example.com:../repo", "owner/repo;calc", "owner/repo\nnext",
	}
	for _, input := range inputs {
		if _, err := parseRemote(input); err == nil {
			t.Errorf("parseRemote(%q) should fail", input)
		}
	}
}

func TestValidateManagedTarget(t *testing.T) {
	valid := []struct{ platform, namespace, name, visibility string }{
		{PlatformGitHub, "company", "backup_repo", "private"},
		{PlatformGitLab, "company/platform", "backup.repo", "internal"},
	}
	for _, item := range valid {
		if err := validateManagedTarget(item.platform, item.namespace, item.name, item.visibility); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	invalid := []struct{ platform, namespace, name, visibility string }{
		{PlatformGeneric, "company", "repo", "private"},
		{PlatformGitHub, "company/..", "repo", "private"},
		{PlatformGitLab, "company", "bad/name", "private"},
		{PlatformGitHub, "company", "repo", "secret"},
	}
	for _, item := range invalid {
		if err := validateManagedTarget(item.platform, item.namespace, item.name, item.visibility); err == nil {
			t.Errorf("expected error for %#v", item)
		}
	}
}
