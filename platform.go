package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type commandSpec struct {
	Name string
	Args []string
}

type gitLabProject struct {
	Visibility    string `json:"visibility"`
	HTTPURLToRepo string `json:"http_url_to_repo"`
	WebURL        string `json:"web_url"`
}

type gitLabProtectedBranch struct {
	Name           string `json:"name"`
	AllowForcePush bool   `json:"allow_force_push"`
}

func platformCLI(platform string) string {
	if definition := platformByID(platform); definition != nil {
		return definition.CLI
	}
	return ""
}

func cloneCommand(remote RemoteSpec, destination, mode string, status EnvironmentStatus) commandSpec {
	flags := []string{"--mirror"}
	if mode == "shallow" {
		flags = []string{"--depth=1", "--single-branch"}
	}
	definition := platformByID(remote.Platform)
	if definition != nil && definition.CloneViaCLI {
		if tool := status.Tool(remote.Platform); tool.Installed && tool.Authenticated {
			args := []string{"repo", "clone", remote.CloneURL, destination, "--"}
			args = append(args, flags...)
			return commandSpec{Name: definition.CLI, Args: args}
		}
	}
	args := []string{"clone"}
	args = append(args, flags...)
	args = append(args, remote.CloneURL, destination)
	return commandSpec{Name: "git", Args: args}
}

func (a *App) ensureManagedRepository(id, platform, namespace, requested, visibility, policy string) (RemoteSpec, string, error) {
	name := requested
	for suffix := 0; suffix < 100; suffix++ {
		fullName := namespace + "/" + name
		existingVisibility, exists, err := a.managedRepositoryVisibility(platform, fullName)
		if err != nil {
			return RemoteSpec{}, "", err
		}
		if !exists {
			a.emitProgress(id, 1, "创建目标仓库", "running")
			if err := a.createManagedRepository(id, platform, fullName, visibility); err != nil {
				a.emitProgress(id, 1, "创建目标仓库", "failed")
				return RemoteSpec{}, "", err
			}
			a.emitProgress(id, 1, "创建目标仓库", "done")
			remote, err := a.fetchManagedRemote(platform, fullName)
			return remote, visibility, err
		}
		if policy == "associate" {
			a.emitLog(id, "info", "已关联现有目标仓库 "+fullName)
			remote, err := a.fetchManagedRemote(platform, fullName)
			return remote, strings.ToLower(existingVisibility), err
		}
		if policy != "rename" {
			return RemoteSpec{}, "", fmt.Errorf("目标仓库已存在，请选择关联已有仓库或自动重命名")
		}
		name = requested + "-" + strconv.Itoa(suffix+2)
	}
	return RemoteSpec{}, "", fmt.Errorf("自动重命名尝试次数过多")
}

func (a *App) managedRepositoryVisibility(platform, fullName string) (string, bool, error) {
	definition := platformByID(platform)
	if definition == nil || definition.RepoExists == nil {
		return "", false, fmt.Errorf("不支持的平台")
	}
	return definition.RepoExists(a, fullName)
}

// looksLikeMissing reports whether CLI output indicates the repository does
// not exist rather than a network or authentication problem.
func looksLikeMissing(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "404") ||
		strings.Contains(lower, "not found") ||
		strings.Contains(lower, "could not resolve to a repository")
}

func repoExistsViaGitHubAPI(app *App, fullName string) (string, bool, error) {
	output, err := app.runQuiet("gh", "repo", "view", fullName, "--json", "visibility", "--jq", ".visibility")
	if err != nil {
		if looksLikeMissing(output) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("目标仓库查询失败，请检查 CLI 登录状态和网络连接")
	}
	return strings.TrimSpace(output), true, nil
}

func repoExistsViaGitLabAPI(app *App, fullName string) (string, bool, error) {
	project, output, err := fetchGitLabProject(app, fullName)
	if err != nil {
		if looksLikeMissing(output) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("目标仓库查询失败，请检查 CLI 登录状态和网络连接")
	}
	return project.Visibility, true, nil
}

func fetchGitLabProject(app *App, fullName string) (gitLabProject, string, error) {
	output, err := app.runQuiet("glab", "api", "projects/"+url.PathEscape(fullName))
	if err != nil {
		return gitLabProject{}, output, err
	}
	var project gitLabProject
	if err := json.Unmarshal([]byte(output), &project); err != nil {
		return gitLabProject{}, output, fmt.Errorf("GitLab 返回的项目信息格式无效: %w", err)
	}
	if project.Visibility == "" || project.HTTPURLToRepo == "" || project.WebURL == "" {
		return gitLabProject{}, output, fmt.Errorf("GitLab 返回的项目信息不完整")
	}
	return project, output, nil
}

func fetchGitLabProtectedBranches(app *App, fullName string) ([]gitLabProtectedBranch, error) {
	output, err := app.runQuiet("glab", "api", "projects/"+url.PathEscape(fullName)+"/protected_branches?per_page=100")
	if err != nil {
		return nil, err
	}
	var branches []gitLabProtectedBranch
	if err := json.Unmarshal([]byte(output), &branches); err != nil {
		return nil, fmt.Errorf("GitLab 返回的保护分支信息格式无效: %w", err)
	}
	return branches, nil
}

func (a *App) setGitLabProtectedBranchForcePush(fullName, branch string, allowed bool) error {
	endpoint := "projects/" + url.PathEscape(fullName) + "/protected_branches/" + url.PathEscape(branch) + "?allow_force_push=" + strconv.FormatBool(allowed)
	if _, err := a.runQuiet("glab", "api", "--method", "PATCH", endpoint); err != nil {
		return err
	}
	return nil
}

func createRepoViaGitHubCLI(app *App, id, fullName, visibility string) error {
	if err := app.runCommand(id, "gh", "repo", "create", fullName, "--"+visibility); err != nil {
		return fmt.Errorf("GitHub 仓库创建失败: %w", err)
	}
	return nil
}

func createRepoViaGitLabCLI(app *App, id, fullName, visibility string) error {
	if err := app.runCommand(id, "glab", "repo", "create", fullName, "--"+visibility, "--skipGitInit"); err != nil {
		return fmt.Errorf("GitLab 仓库创建失败: %w", err)
	}
	return nil
}

func fetchRepoViaGitHubAPI(app *App, fullName string) (string, string, error) {
	cloneURL, err := app.runQuiet("gh", "api", "repos/"+fullName, "--jq", ".clone_url")
	if err == nil {
		var webURL string
		webURL, err = app.runQuiet("gh", "api", "repos/"+fullName, "--jq", ".html_url")
		return strings.TrimSpace(cloneURL), strings.TrimSpace(webURL), err
	}
	return "", "", err
}

func fetchRepoViaGitLabAPI(app *App, fullName string) (string, string, error) {
	project, _, err := fetchGitLabProject(app, fullName)
	return project.HTTPURLToRepo, project.WebURL, err
}

func (a *App) createManagedRepository(id, platform, fullName, visibility string) error {
	definition := platformByID(platform)
	if definition == nil || definition.CreateRepo == nil {
		return fmt.Errorf("不支持的平台")
	}
	return definition.CreateRepo(a, id, fullName, visibility)
}

func (a *App) fetchManagedRemote(platform, fullName string) (RemoteSpec, error) {
	definition := platformByID(platform)
	if definition == nil || definition.FetchRepo == nil {
		return RemoteSpec{}, fmt.Errorf("不支持的平台")
	}
	parts := strings.Split(fullName, "/")
	cloneURL, webURL, err := definition.FetchRepo(a, fullName)
	if err != nil {
		return RemoteSpec{}, fmt.Errorf("读取目标仓库信息失败: %w", err)
	}
	remote := makeRemote(platform, definition.Host, parts, strings.TrimSpace(cloneURL))
	remote.WebURL = strings.TrimSpace(webURL)
	return remote, nil
}
