package main

import (
	"strings"
)

// platformDefinition describes one Git hosting platform and how the app
// integrates with its CLI. Registry entries replace the scattered per-platform
// switches; github and gitlab keep their existing behaviour exactly.
type platformDefinition struct {
	ID           string
	Label        string
	Host         string
	CLI          string
	Managed      bool
	CloneViaCLI  bool
	Visibilities []string
	// ForceNegotiation marks targets whose protected branches must be
	// temporarily relaxed before a forced mirror push (GitLab today).
	ForceNegotiation bool
	// AuthStatusArgs builds the quiet command that succeeds only when the CLI
	// is authenticated. Its output also feeds login parsing when
	// LoginProbeArgs is nil.
	AuthStatusArgs func() []string
	// LoginProbeArgs optionally runs a second command that prints the login
	// name; nil reuses the auth status output.
	LoginProbeArgs func() []string
	ParseLogin     func(output string) string
	// LoginArgs builds the interactive CLI login command for the host.
	LoginArgs func(host string) []string
	// RepoExists reports the visibility and existence of a managed repository.
	RepoExists func(app *App, fullName string) (string, bool, error)
	// ValidateNamespace optionally enforces platform-specific namespace rules
	// beyond the shared path-part validation.
	ValidateNamespace func(namespace string) error
	// CreateRepo creates fullName with the requested visibility.
	CreateRepo func(app *App, id, fullName, visibility string) error
	// FetchRepo returns the clone and web URLs of fullName.
	FetchRepo func(app *App, fullName string) (string, string, error)
	// ParseHostPath overrides default owner/repo parsing for hosts whose URL
	// layout is not <namespace>/<repository> (Azure DevOps). It receives the
	// already decoded, .git-stripped path segments.
	ParseHostPath func(parts []string) ([]string, error)
	// WebURL builds the browse URL for a managed namespace/repository. nil
	// falls back to https://<host>/<namespace>/<repository>.
	WebURL func(namespace, repository string) string
}

var platformRegistry = map[string]*platformDefinition{
	PlatformGitHub:      githubPlatform(),
	PlatformGitLab:      gitlabPlatform(),
	PlatformGitee:       giteePlatform(),
	PlatformCodeberg:    codebergPlatform(),
	PlatformAzureDevOps: azureDevOpsPlatform(),
}

// platformOrder keeps a deterministic iteration order for environment checks.
var platformOrder = []string{
	PlatformGitHub, PlatformGitLab, PlatformGitee, PlatformCodeberg, PlatformAzureDevOps,
}

// platformByID returns the registry entry for managed platforms and nil for
// generic or unknown remotes.
func platformByID(id string) *platformDefinition { return platformRegistry[id] }

// platformByHost maps a git host onto its registry entry; unknown hosts have
// no entry and fall back to the generic platform.
func platformByHost(host string) *platformDefinition {
	normalized := strings.ToLower(host)
	for _, id := range platformOrder {
		if platformRegistry[id].Host == normalized {
			return platformRegistry[id]
		}
	}
	return nil
}

// managedPlatforms lists the registry entries in deterministic order.
func managedPlatforms() []*platformDefinition {
	definitions := make([]*platformDefinition, 0, len(platformOrder))
	for _, id := range platformOrder {
		definitions = append(definitions, platformRegistry[id])
	}
	return definitions
}

// forceNegotiationRequired reports whether forced pushes to the platform need
// protected-branch negotiation first.
func forceNegotiationRequired(platform string) bool {
	definition := platformByID(platform)
	return definition != nil && definition.ForceNegotiation
}

// Tool selects the EnvironmentStatus field of a platform. The JSON contract
// keeps dedicated fields, so the mapping stays explicit.
func (status EnvironmentStatus) Tool(platform string) ToolStatus {
	switch platform {
	case PlatformGitHub:
		return status.GitHub
	case PlatformGitLab:
		return status.GitLab
	case PlatformGitee:
		return status.Gitee
	case PlatformCodeberg:
		return status.Codeberg
	case PlatformAzureDevOps:
		return status.AzureDevOps
	}
	return ToolStatus{}
}

func githubPlatform() *platformDefinition {
	return &platformDefinition{
		ID:           PlatformGitHub,
		Label:        "GitHub",
		Host:         "github.com",
		CLI:          "gh",
		Managed:      true,
		CloneViaCLI:  true,
		Visibilities: []string{"private", "public"},
		AuthStatusArgs: func() []string {
			return []string{"auth", "status", "--hostname", "github.com"}
		},
		LoginProbeArgs: func() []string {
			return []string{"api", "user", "--hostname", "github.com", "--jq", ".login"}
		},
		ParseLogin: func(output string) string { return strings.TrimSpace(output) },
		LoginArgs: func(host string) []string {
			return []string{"auth", "login", "--hostname", host, "--web", "--git-protocol", "https"}
		},
		RepoExists: repoExistsViaGitHubAPI,
		CreateRepo: createRepoViaGitHubCLI,
		FetchRepo:  fetchRepoViaGitHubAPI,
	}
}

func gitlabPlatform() *platformDefinition {
	return &platformDefinition{
		ID:               PlatformGitLab,
		Label:            "GitLab",
		Host:             "gitlab.com",
		CLI:              "glab",
		Managed:          true,
		CloneViaCLI:      true,
		Visibilities:     []string{"private", "internal", "public"},
		ForceNegotiation: true,
		AuthStatusArgs: func() []string {
			return []string{"auth", "status", "--hostname", "gitlab.com"}
		},
		LoginProbeArgs: func() []string {
			return []string{"api", "user", "--hostname", "gitlab.com", "--jq", ".username"}
		},
		ParseLogin: func(output string) string { return strings.TrimSpace(output) },
		LoginArgs: func(host string) []string {
			return []string{"auth", "login", "--hostname", host, "--device", "--git-protocol", "https"}
		},
		RepoExists: repoExistsViaGitLabAPI,
		CreateRepo: createRepoViaGitLabCLI,
		FetchRepo:  fetchRepoViaGitLabAPI,
		WebURL:     nil,
	}
}
