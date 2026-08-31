package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (a *App) performSync(repo Repository, syncLFS bool) error {
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
	clone := cloneCommand(repo.Source, cloneDir, repo.Mode, a.CheckEnvironment())
	a.emitProgress(repo.ID, 2, "拉取源仓库", "running")
	if err := a.runCommand(repo.ID, clone.Name, clone.Args...); err != nil {
		a.emitProgress(repo.ID, 2, "拉取源仓库", "failed")
		return fmt.Errorf("拉取源仓库失败: %w", err)
	}
	a.emitProgress(repo.ID, 2, "拉取源仓库", "done")
	a.emitProgress(repo.ID, 3, "推送目标仓库", "running")
	if repo.Mode == "shallow" {
		branch, err := a.runQuietIn(cloneDir, "git", "branch", "--show-current")
		if err != nil || strings.TrimSpace(branch) == "" {
			a.emitProgress(repo.ID, 3, "推送目标仓库", "failed")
			return fmt.Errorf("无法识别源仓库默认分支")
		}
		if err = a.runCommandIn(repo.ID, cloneDir, "git", "push", "--force", repo.Target.CloneURL, "HEAD:refs/heads/"+strings.TrimSpace(branch)); err != nil {
			a.emitProgress(repo.ID, 3, "推送目标仓库", "failed")
			return fmt.Errorf("推送目标仓库失败: %w", err)
		}
	} else if err := a.runCommandIn(repo.ID, cloneDir, "git", "push", "--force", "--prune", repo.Target.CloneURL, "+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*"); err != nil {
		a.emitProgress(repo.ID, 3, "推送目标仓库", "failed")
		return fmt.Errorf("推送目标仓库失败: %w", err)
	}
	if syncLFS {
		if _, err := findExecutable("git-lfs"); err != nil {
			a.emitLog(repo.ID, "warning", "未安装 Git LFS，已跳过大文件对象")
		} else {
			a.emitLog(repo.ID, "info", "正在同步 Git LFS 对象")
			if err := a.runCommandIn(repo.ID, cloneDir, "git", "lfs", "fetch", "--all", repo.Source.CloneURL); err != nil {
				return fmt.Errorf("Git LFS 拉取失败: %w", err)
			}
			if err := a.runCommandIn(repo.ID, cloneDir, "git", "lfs", "push", "--all", repo.Target.CloneURL); err != nil {
				return fmt.Errorf("Git LFS 推送失败: %w", err)
			}
		}
	}
	a.emitProgress(repo.ID, 3, "推送目标仓库", "done")
	a.emitProgress(repo.ID, 4, "清理临时文件", "done")
	return nil
}
