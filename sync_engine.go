package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (a *App) performSync(repo Repository, syncLFS bool, environment EnvironmentStatus) error {
	tempDir, err := os.MkdirTemp("", "git-repo-mirror-*")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			a.emitLog(repo.ID, "warning", "临时目录清理失败: "+err.Error())
		}
	}()
	cloneDir := filepath.Join(tempDir, "repository")
	clone := cloneCommand(repo.Source, cloneDir, repo.Mode, environment)
	commandEnvironment := a.commandEnvForRemotes(repo.Source, repo.Target)
	a.emitProgress(repo.ID, 2, "拉取源仓库", "running")
	if err := a.runCommandInWithEnv(repo.ID, "", commandEnvironment, clone.Name, clone.Args...); err != nil {
		a.emitProgress(repo.ID, 2, "拉取源仓库", "failed")
		return fmt.Errorf("拉取源仓库失败: %w", err)
	}
	a.emitProgress(repo.ID, 2, "拉取源仓库", "done")
	a.emitProgress(repo.ID, 3, "推送目标仓库", "running")
	branches := []string(nil)
	if repo.Mode == "shallow" {
		branch, err := a.runQuietInWithEnv(cloneDir, commandEnvironment, "git", "branch", "--show-current")
		if err != nil || strings.TrimSpace(branch) == "" {
			a.emitProgress(repo.ID, 3, "推送目标仓库", "failed")
			return fmt.Errorf("无法识别源仓库默认分支")
		}
		branches = []string{strings.TrimSpace(branch)}
	} else if repo.Target.Platform == PlatformGitLab && repo.ManagementMode == ManagementCLI {
		var err error
		branches, err = localBranches(cloneDir, commandEnvironment, a)
		if err != nil {
			a.emitProgress(repo.ID, 3, "推送目标仓库", "failed")
			return fmt.Errorf("无法读取源仓库分支: %w", err)
		}
	}
	restoreProtection, force, err := a.prepareGitLabForcePush(repo, branches)
	if err != nil {
		a.emitProgress(repo.ID, 3, "推送目标仓库", "failed")
		return err
	}
	var push commandSpec
	if repo.Mode == "shallow" {
		push = shallowPushCommand(repo.Target, branches[0], force)
	} else {
		push = mirrorPushCommand(repo.Target, force)
	}
	pushErr := a.runCommandInWithEnv(repo.ID, cloneDir, commandEnvironment, push.Name, push.Args...)
	restoreErr := restoreProtection()
	if pushErr != nil {
		a.emitProgress(repo.ID, 3, "推送目标仓库", "failed")
		if restoreErr != nil {
			return errors.Join(fmt.Errorf("推送目标仓库失败: %w", pushErr), fmt.Errorf("恢复 GitLab 分支保护失败: %w", restoreErr))
		}
		return fmt.Errorf("推送目标仓库失败: %w", pushErr)
	}
	if restoreErr != nil {
		a.emitProgress(repo.ID, 3, "推送目标仓库", "failed")
		return fmt.Errorf("恢复 GitLab 分支保护失败: %w", restoreErr)
	}
	if syncLFS {
		if _, err := findExecutable("git-lfs"); err != nil {
			a.emitLog(repo.ID, "warning", "未安装 Git LFS，已跳过大文件对象")
		} else {
			a.emitLog(repo.ID, "info", "正在同步 Git LFS 对象")
			if err := a.runCommandInWithEnv(repo.ID, cloneDir, commandEnvironment, "git", "lfs", "fetch", "--all", repo.Source.CloneURL); err != nil {
				return fmt.Errorf("Git LFS 拉取失败: %w", err)
			}
			if err := a.runCommandInWithEnv(repo.ID, cloneDir, commandEnvironment, "git", "lfs", "push", "--all", repo.Target.CloneURL); err != nil {
				return fmt.Errorf("Git LFS 推送失败: %w", err)
			}
		}
	}
	a.emitProgress(repo.ID, 3, "推送目标仓库", "done")
	a.emitProgress(repo.ID, 4, "清理临时文件", "done")
	return nil
}

func (a *App) prepareGitLabForcePush(repo Repository, branches []string) (func() error, bool, error) {
	noop := func() error { return nil }
	if repo.Target.Platform != PlatformGitLab || repo.ManagementMode != ManagementCLI {
		return noop, repo.Target.Platform != PlatformGitLab, nil
	}
	fullName := strings.Trim(strings.TrimSpace(repo.Target.Namespace), "/") + "/" + strings.TrimSpace(repo.Target.Repository)
	rules, err := a.fetchGitLabProtectedBranches(fullName)
	if err != nil {
		a.emitLog(repo.ID, "warning", "无法读取 GitLab 分支保护规则，改用非强制推送: "+err.Error())
		return noop, false, nil
	}
	var changed []string
	for _, rule := range rules {
		if rule.AllowForcePush || !protectedBranchRuleMatchesAny(rule.Name, branches) {
			continue
		}
		if err := a.setGitLabProtectedBranchForcePush(fullName, rule.Name, true); err != nil {
			restoreErr := restoreGitLabForcePush(a, repo.ID, fullName, changed)
			if restoreErr != nil {
				return noop, false, errors.Join(
					fmt.Errorf("无法为 GitLab 保护分支 %q 临时允许强制推送: %w", rule.Name, err),
					fmt.Errorf("恢复 GitLab 分支保护失败: %w", restoreErr),
				)
			}
			return noop, false, fmt.Errorf("无法为 GitLab 保护分支 %q 临时允许强制推送，请确认当前账号有管理分支保护权限: %w", rule.Name, err)
		}
		changed = append(changed, rule.Name)
	}
	if len(changed) > 0 {
		a.emitLog(repo.ID, "info", "已临时允许 GitLab 保护分支强制推送，镜像完成后自动恢复")
	}
	return func() error {
		return restoreGitLabForcePush(a, repo.ID, fullName, changed)
	}, true, nil
}

func restoreGitLabForcePush(app *App, repositoryID, fullName string, branches []string) error {
	var restoreErr error
	for index := len(branches) - 1; index >= 0; index-- {
		branch := branches[index]
		if err := app.setGitLabProtectedBranchForcePush(fullName, branch, false); err != nil {
			restoreErr = errors.Join(restoreErr, fmt.Errorf("%s: %w", branch, err))
			continue
		}
	}
	if restoreErr == nil && len(branches) > 0 {
		app.emitLog(repositoryID, "info", "GitLab 分支保护已恢复")
	}
	return restoreErr
}

func localBranches(cloneDir string, environment []string, app *App) ([]string, error) {
	output, err := app.runQuietInWithEnv(cloneDir, environment, "git", "for-each-ref", "--format=%(refname:strip=2)", "refs/heads")
	if err != nil {
		return nil, err
	}
	var branches []string
	for _, branch := range strings.Split(output, "\n") {
		if branch = strings.TrimSpace(branch); branch != "" {
			branches = append(branches, branch)
		}
	}
	return branches, nil
}

func protectedBranchRuleMatchesAny(rule string, branches []string) bool {
	for _, branch := range branches {
		if protectedBranchRuleMatches(rule, branch) {
			return true
		}
	}
	return false
}

func protectedBranchRuleMatches(rule, branch string) bool {
	if !strings.Contains(rule, "*") {
		return rule == branch
	}
	parts := strings.Split(rule, "*")
	if !strings.HasPrefix(branch, parts[0]) {
		return false
	}
	branch = strings.TrimPrefix(branch, parts[0])
	for _, part := range parts[1:] {
		index := strings.Index(branch, part)
		if index < 0 {
			return false
		}
		branch = branch[index+len(part):]
	}
	return true
}

func mirrorPushCommand(target RemoteSpec, force bool) commandSpec {
	force = force || target.Platform != PlatformGitLab
	args := []string{"push"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, "--prune", target.CloneURL)
	heads, tags := "refs/heads/*:refs/heads/*", "refs/tags/*:refs/tags/*"
	if force {
		heads, tags = "+"+heads, "+"+tags
	}
	return commandSpec{Name: "git", Args: append(args, heads, tags)}
}

func shallowPushCommand(target RemoteSpec, branch string, force bool) commandSpec {
	force = force || target.Platform != PlatformGitLab
	args := []string{"push"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, target.CloneURL, "HEAD:refs/heads/"+branch)
	return commandSpec{Name: "git", Args: args}
}
