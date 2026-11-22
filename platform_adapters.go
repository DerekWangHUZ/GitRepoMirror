package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// giteePlatform adapts the official gitee CLI (npm @gitee/gitee-cli). The CLI
// login flow stores a personal access token that also authenticates git over
// HTTPS, so syncs inject a basic-auth extra header built from the token.
func giteePlatform() *platformDefinition {
	return &platformDefinition{
		ID:           PlatformGitee,
		Label:        "Gitee",
		Host:         "gitee.com",
		CLI:          "gitee",
		Managed:      true,
		CloneViaCLI:  false,
		Visibilities: []string{"private", "public"},
		AuthStatusArgs: func() []string {
			return []string{"api", "user"}
		},
		ParseLogin: parseGiteeLogin,
		LoginArgs: func(host string) []string {
			return []string{"auth", "login", "--hostname", host}
		},
		RepoExists: repoExistsViaGiteeAPI,
		CreateRepo: createRepoViaGiteeCLI,
		FetchRepo:  fetchRepoViaGiteeAPI,
		StatusField: func(status *EnvironmentStatus) *ToolStatus {
			return &status.Gitee
		},
	}
}

type giteeRepository struct {
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
	HTMLURL  string `json:"html_url"`
}

func repoExistsViaGiteeAPI(app *App, fullName string) (string, bool, error) {
	output, err := app.runQuiet("gitee", "api", "/repos/"+fullName)
	if err != nil {
		if looksLikeMissing(output) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("目标仓库查询失败，请检查 CLI 登录状态和网络连接")
	}
	var repo giteeRepository
	if err := json.Unmarshal(jsonFromOutput(output), &repo); err != nil {
		return "", false, fmt.Errorf("Gitee 返回的仓库信息格式无效: %w", err)
	}
	visibility := "public"
	if repo.Private {
		visibility = "private"
	}
	return visibility, true, nil
}

func createRepoViaGiteeCLI(app *App, id, fullName, visibility string) error {
	parts := strings.Split(fullName, "/")
	private := visibility == "private"
	switch {
	case len(parts) == 2:
		args := []string{"repo", "create", "-n", parts[1]}
		if private {
			args = append(args, "--private")
		}
		if err := app.runCommand(id, "gitee", args...); err != nil {
			return fmt.Errorf("Gitee 仓库创建失败: %w", err)
		}
	case len(parts) == 3:
		// Group or organization namespaces go through the Gitee v5 API; the
		// repo create command only targets the personal space.
		owner := parts[0] + "/" + parts[1]
		args := []string{"api", "/orgs/" + owner + "/repos", "--method", "POST",
			"-f", "name=" + parts[2], "-f", "private=" + fmt.Sprintf("%t", private)}
		if err := app.runCommand(id, "gitee", args...); err != nil {
			return fmt.Errorf("Gitee 仓库创建失败: %w", err)
		}
	default:
		return fmt.Errorf("Gitee 目标命名空间格式无效")
	}
	return nil
}

func fetchRepoViaGiteeAPI(app *App, fullName string) (string, string, error) {
	output, err := app.runQuiet("gitee", "api", "/repos/"+fullName)
	if err != nil {
		return "", "", err
	}
	var repo giteeRepository
	if err := json.Unmarshal(jsonFromOutput(output), &repo); err != nil {
		return "", "", fmt.Errorf("Gitee 返回的仓库信息格式无效: %w", err)
	}
	if repo.FullName == "" {
		return "", "", fmt.Errorf("Gitee 返回的仓库信息不完整")
	}
	cloneURL := "https://gitee.com/" + repo.FullName + ".git"
	return cloneURL, repo.HTMLURL, nil
}

// codebergPlatform adapts the forgejo CLI (fj). fj places the host flag before
// the subcommand and localizes its output; the login name is therefore parsed
// from the "<name>@<host>" tail of `fj whoami`.
func codebergPlatform() *platformDefinition {
	return &platformDefinition{
		ID:           PlatformCodeberg,
		Label:        "Codeberg",
		Host:         "codeberg.org",
		CLI:          "fj",
		Managed:      true,
		CloneViaCLI:  false,
		Visibilities: []string{"private", "public"},
		AuthStatusArgs: func() []string {
			return []string{"-H", "codeberg.org", "whoami"}
		},
		ParseLogin: parseForgejoLogin,
		LoginArgs: func(host string) []string {
			return []string{"-H", host, "auth", "login"}
		},
		RepoExists: repoExistsViaForgejoCLI,
		CreateRepo: createRepoViaForgejoCLI,
		FetchRepo:  fetchRepoViaForgejoCLI,
		StatusField: func(status *EnvironmentStatus) *ToolStatus {
			return &status.Codeberg
		},
	}
}

var giteeLoginPattern = regexp.MustCompile(`"login"\s*:\s*"([^"]+)"`)

// parseGiteeLogin extracts the login from the Gitee API user payload. The
// combined output may carry npm shim noise around the JSON, so a direct
// unmarshal is backed up by a targeted field regex.
func parseGiteeLogin(output string) string {
	var user struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(jsonFromOutput(output), &user); err == nil && user.Login != "" {
		return user.Login
	}
	if match := giteeLoginPattern.FindStringSubmatch(output); match != nil {
		return match[1]
	}
	return ""
}

var forgejoLoginPattern = regexp.MustCompile(`([A-Za-z0-9_.-]+)@([A-Za-z0-9.-]+)\s*$`)

func parseForgejoLogin(output string) string {
	match := forgejoLoginPattern.FindStringSubmatch(strings.TrimSpace(output))
	if match == nil {
		return ""
	}
	return match[1]
}

func repoExistsViaForgejoCLI(app *App, fullName string) (string, bool, error) {
	_, err := app.runQuiet("fj", "-H", "codeberg.org", "repo", "view", fullName)
	if err != nil {
		if looksLikeMissing(err.Error()) {
			return "", false, nil
		}
		// Forgejo reports missing repositories in localized text; a view
		// failure cannot be distinguished from a missing repository in every
		// case, so the error is surfaced for a real network/auth problem and
		// treated as missing for plain lookup failures.
		return "", false, fmt.Errorf("目标仓库查询失败，请检查 CLI 登录状态和网络连接")
	}
	return "", true, nil
}

func createRepoViaForgejoCLI(app *App, id, fullName, visibility string) error {
	parts := strings.Split(fullName, "/")
	if len(parts) < 2 {
		return fmt.Errorf("Codeberg 目标命名空间格式无效")
	}
	name := parts[len(parts)-1]
	args := []string{"-H", "codeberg.org"}
	if len(parts) == 2 {
		args = append(args, "repo", "create", name)
	} else {
		args = append(args, "org", "repo", "create", strings.Join(parts[:len(parts)-1], "/"), name)
	}
	if visibility == "private" {
		args = append(args, "--private")
	}
	if err := app.runCommand(id, "fj", args...); err != nil {
		return fmt.Errorf("Codeberg 仓库创建失败: %w", err)
	}
	return nil
}

func fetchRepoViaForgejoCLI(app *App, fullName string) (string, string, error) {
	cloneURL := "https://codeberg.org/" + fullName + ".git"
	webURL := "https://codeberg.org/" + fullName
	return cloneURL, webURL, nil
}

// azureDevOpsPlatform adapts the Azure CLI (az) with the azure-devops
// extension. The namespace is <organization>/<project>; repository visibility
// is governed by the project, so the app keeps a fixed nominal value.
func azureDevOpsPlatform() *platformDefinition {
	return &platformDefinition{
		ID:           PlatformAzureDevOps,
		Label:        "Azure DevOps",
		Host:         "dev.azure.com",
		CLI:          "az",
		Managed:      true,
		CloneViaCLI:  false,
		Visibilities: []string{"private", "public"},
		AuthStatusArgs: func() []string {
			return []string{"account", "show", "--query", "user.name", "--output", "tsv", "--only-show-errors"}
		},
		ParseLogin: func(output string) string {
			return strings.TrimSpace(firstNonEmptyLine(output))
		},
		LoginArgs: func(host string) []string {
			return []string{"login", "--allow-no-subscriptions"}
		},
		RepoExists:    repoExistsViaAzureCLI,
		CreateRepo:    createRepoViaAzureCLI,
		FetchRepo:     fetchRepoViaAzureCLI,
		ParseHostPath: parseAzureDevOpsPath,
		WebURL:        azureDevOpsWebURL,
		ValidateNamespace: func(namespace string) error {
			if len(strings.Split(namespace, "/")) != 2 {
				return fmt.Errorf("Azure DevOps 目标命名空间需使用 <组织>/<项目>")
			}
			return nil
		},
		StatusField: func(status *EnvironmentStatus) *ToolStatus {
			return &status.AzureDevOps
		},
	}
}

func azureOrganizationURL(fullName string) (string, string, string, error) {
	parts := strings.Split(fullName, "/")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("Azure DevOps 目标命名空间需使用 <组织>/<项目>，且仓库名有效")
	}
	return "https://dev.azure.com/" + parts[0], parts[1], parts[2], nil
}

type azureRepository struct {
	RemoteURL string `json:"remoteUrl"`
	WebURL    string `json:"webUrl"`
}

func azReposShow(app *App, fullName string) (azureRepository, string, error) {
	orgURL, project, name, err := azureOrganizationURL(fullName)
	if err != nil {
		return azureRepository{}, "", err
	}
	args := []string{"repos", "show", "--org", orgURL, "--project", project, "--name", name,
		"--detect", "false", "--only-show-errors", "--output", "json"}
	output, err := app.runQuiet("az", args...)
	if err != nil {
		return azureRepository{}, output, err
	}
	var repo azureRepository
	if err := json.Unmarshal(jsonFromOutput(output), &repo); err != nil {
		return azureRepository{}, output, fmt.Errorf("Azure DevOps 返回的仓库信息格式无效: %w", err)
	}
	return repo, output, nil
}

func repoExistsViaAzureCLI(app *App, fullName string) (string, bool, error) {
	_, output, err := azReposShow(app, fullName)
	if err != nil {
		lower := strings.ToLower(output)
		if strings.Contains(lower, "resourcenotfound") || strings.Contains(lower, "does not exist") || looksLikeMissing(lower) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("目标仓库查询失败，请检查 CLI 登录状态和网络连接")
	}
	return "private", true, nil
}

func createRepoViaAzureCLI(app *App, id, fullName, visibility string) error {
	orgURL, project, name, err := azureOrganizationURL(fullName)
	if err != nil {
		return err
	}
	args := []string{"repos", "create", "--name", name, "--org", orgURL, "--project", project,
		"--detect", "false", "--only-show-errors", "--output", "json"}
	if err := app.runCommand(id, "az", args...); err != nil {
		return fmt.Errorf("Azure DevOps 仓库创建失败: %w", err)
	}
	return nil
}

func fetchRepoViaAzureCLI(app *App, fullName string) (string, string, error) {
	repo, _, err := azReposShow(app, fullName)
	if err != nil {
		return "", "", err
	}
	orgURL, project, name, err := azureOrganizationURL(fullName)
	if err != nil {
		return "", "", err
	}
	cloneURL := repo.RemoteURL
	if cloneURL == "" {
		cloneURL = orgURL + "/" + project + "/_git/" + name
	}
	webURL := repo.WebURL
	if webURL == "" {
		webURL = orgURL + "/" + project + "/_git/" + name
	}
	return cloneURL, webURL, nil
}

// parseAzureDevOpsPath normalizes Azure DevOps URL and SCP-style paths:
// <org>/<project>/_git/<repo> and v3/<org>/<project>/<repo> both resolve to
// [<org>, <project>, <repo>].
func parseAzureDevOpsPath(parts []string) ([]string, error) {
	if len(parts) > 0 && parts[0] == "v3" {
		parts = parts[1:]
	}
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "_git" {
			continue
		}
		filtered = append(filtered, part)
	}
	if len(filtered) != 3 {
		return nil, fmt.Errorf("Azure DevOps 仓库地址需使用 <组织>/<项目>/<仓库名> 格式")
	}
	return filtered, nil
}

func azureDevOpsWebURL(namespace, repository string) string {
	parts := strings.Split(namespace, "/")
	if len(parts) != 2 {
		return ""
	}
	return "https://dev.azure.com/" + parts[0] + "/" + parts[1] + "/_git/" + repository
}

// jsonFromOutput extracts the first JSON document from CLI output that may
// carry warnings, update notices, or telemetry noise on stdout or stderr.
func jsonFromOutput(output string) []byte {
	index := strings.Index(output, "{")
	if index < 0 {
		return []byte("{}")
	}
	return []byte(output[index:])
}

func firstNonEmptyLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}

// giteeCredentialOverrides builds a basic-auth extra header for gitee.com from
// the CLI-stored personal access token. The token travels only through the
// child process environment, never through a command line or persistent store.
// A GitHub remote on the same sync chain keeps receiving the gh helper — the
// GitLab strategy applied symmetrically — so a Gitee→GitHub push also
// authenticates when no system credential helper is configured.
func (a *App) giteeCredentialOverrides() ([][2]string, []string, bool) {
	tokenOutput, err := a.runQuietStdout("gitee", "auth", "token")
	if err != nil {
		return nil, nil, false
	}
	token := firstNonEmptyLine(tokenOutput)
	if token == "" {
		return nil, nil, false
	}
	userOutput, err := a.runQuiet("gitee", "api", "user")
	if err != nil {
		return nil, nil, false
	}
	login := giteePlatform().ParseLogin(userOutput)
	if login == "" {
		return nil, nil, false
	}
	header := "Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte(login+":"+token))
	entries := [][2]string{{"http.https://gitee.com/.extraheader", header}}
	var dirs []string
	if ghDir := a.executableDir("gh"); ghDir != "" {
		entries = append(entries, [2]string{"credential.https://github.com.helper", "!gh auth git-credential"})
		dirs = append(dirs, ghDir)
	}
	return entries, dirs, true
}
