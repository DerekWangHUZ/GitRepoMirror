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

func platformCLI(platform string) string {
	switch platform {
	case PlatformGitHub:
		return "gh"
	case PlatformGitLab:
		return "glab"
	default:
		return ""
	}
}

func cloneCommand(remote RemoteSpec, destination, mode string, status EnvironmentStatus) commandSpec {
	flags := []string{"--mirror"}
	if mode == "shallow" {
		flags = []string{"--depth=1", "--single-branch"}
	}
	ready := remote.Platform == PlatformGitHub && status.GitHub.Installed && status.GitHub.Authenticated
	ready = ready || remote.Platform == PlatformGitLab && status.GitLab.Installed && status.GitLab.Authenticated
	if ready {
		args := []string{"repo", "clone", remote.CloneURL, destination, "--"}
		args = append(args, flags...)
		return commandSpec{Name: platformCLI(remote.Platform), Args: args}
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
	var output string
	var err error
	switch platform {
	case PlatformGitHub:
		output, err = a.runQuiet("gh", "repo", "view", fullName, "--json", "visibility", "--jq", ".visibility")
	case PlatformGitLab:
		var project gitLabProject
		project, output, err = a.fetchGitLabProject(fullName)
		if err == nil {
			return project.Visibility, true, nil
		}
	default:
		return "", false, fmt.Errorf("不支持的平台")
	}
	if err != nil {
		lower := strings.ToLower(output)
		if strings.Contains(lower, "404") || strings.Contains(lower, "not found") || strings.Contains(lower, "could not resolve to a repository") {
			return "", false, nil
		}
		return "", false, fmt.Errorf("目标仓库查询失败，请检查 CLI 登录状态和网络连接")
	}
	return strings.TrimSpace(output), true, nil
}

func (a *App) fetchGitLabProject(fullName string) (gitLabProject, string, error) {
	output, err := a.runQuiet("glab", "api", "projects/"+url.PathEscape(fullName))
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

func (a *App) createManagedRepository(id, platform, fullName, visibility string) error {
	visibilityFlag := "--" + visibility
	switch platform {
	case PlatformGitHub:
		if err := a.runCommand(id, "gh", "repo", "create", fullName, visibilityFlag); err != nil {
			return fmt.Errorf("GitHub 仓库创建失败: %w", err)
		}
	case PlatformGitLab:
		if err := a.runCommand(id, "glab", "repo", "create", fullName, visibilityFlag, "--skipGitInit"); err != nil {
			return fmt.Errorf("GitLab 仓库创建失败: %w", err)
		}
	default:
		return fmt.Errorf("不支持的平台")
	}
	return nil
}

func (a *App) fetchManagedRemote(platform, fullName string) (RemoteSpec, error) {
	parts := strings.Split(fullName, "/")
	var cloneURL, webURL string
	var err error
	switch platform {
	case PlatformGitHub:
		cloneURL, err = a.runQuiet("gh", "api", "repos/"+fullName, "--jq", ".clone_url")
		if err == nil {
			webURL, err = a.runQuiet("gh", "api", "repos/"+fullName, "--jq", ".html_url")
		}
	case PlatformGitLab:
		var project gitLabProject
		project, _, err = a.fetchGitLabProject(fullName)
		cloneURL = project.HTTPURLToRepo
		webURL = project.WebURL
	}
	if err != nil {
		return RemoteSpec{}, fmt.Errorf("读取目标仓库信息失败: %w", err)
	}
	host := "github.com"
	if platform == PlatformGitLab {
		host = "gitlab.com"
	}
	remote := makeRemote(platform, host, parts, strings.TrimSpace(cloneURL))
	remote.WebURL = strings.TrimSpace(webURL)
	return remote, nil
}
