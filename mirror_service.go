package main

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// MirrorService methods form the application use-case layer exposed by App.
func (a *App) CreateMirror(req MirrorRequest) (Repository, error) {
	source, err := parseRemote(req.Source)
	if err != nil {
		return Repository{}, err
	}
	if req.Mode != "shallow" {
		req.Mode = "mirror"
	}
	if req.Visibility == "" {
		req.Visibility = "private"
	}
	if req.ConflictPolicy == "" {
		req.ConflictPolicy = "error"
	}
	environment := a.CheckEnvironment()
	if !environment.Git.Installed {
		return Repository{}, fmt.Errorf("未找到 Git，无法执行同步")
	}
	id := newID()
	target, management, visibility, err := a.resolveTarget(id, req, source, environment)
	if err != nil {
		return Repository{}, err
	}
	repo := Repository{ID: id, Source: source, Target: target, ManagementMode: management, Visibility: visibility, Mode: req.Mode, Status: "syncing", CreatedAt: time.Now()}
	if err := a.performSync(repo, req.SyncLFS); err != nil {
		repo.Status, repo.LastError = "failed", err.Error()
		if saveErr := a.upsertRepository(repo); saveErr != nil {
			return repo, errors.Join(err, saveErr)
		}
		return repo, err
	}
	repo.Status, repo.LastSync = "healthy", time.Now()
	if err := a.upsertRepository(repo); err != nil {
		return repo, err
	}
	return repo, nil
}

func (a *App) resolveTarget(id string, req MirrorRequest, source RemoteSpec, environment EnvironmentStatus) (RemoteSpec, string, string, error) {
	platform := req.TargetPlatform
	if platform == "" {
		platform = PlatformGeneric
	}
	ready := platform == PlatformGitHub && environment.GitHub.Installed && environment.GitHub.Authenticated
	ready = ready || platform == PlatformGitLab && environment.GitLab.Installed && environment.GitLab.Authenticated
	if ready {
		name := strings.TrimSpace(req.TargetName)
		if name == "" {
			name = strings.TrimSpace(req.Prefix) + source.Repository + strings.TrimSpace(req.Suffix)
		}
		namespace := strings.Trim(strings.TrimSpace(req.TargetNamespace), "/")
		if namespace == "" {
			if platform == PlatformGitHub {
				namespace = environment.GitHub.Login
			} else {
				namespace = environment.GitLab.Login
			}
		}
		if err := validateManagedTarget(platform, namespace, name, req.Visibility); err != nil {
			return RemoteSpec{}, "", "", err
		}
		remote, visibility, err := a.ensureManagedRepository(id, platform, namespace, name, req.Visibility, req.ConflictPolicy)
		return remote, ManagementCLI, visibility, err
	}
	target, err := parseRemote(req.TargetURL)
	if err != nil {
		return RemoteSpec{}, "", "", fmt.Errorf("目标仓库: %w", err)
	}
	if platform == PlatformGitHub && target.Platform != PlatformGitHub {
		return RemoteSpec{}, "", "", fmt.Errorf("GitHub 降级模式需要 github.com 的完整目标 URL")
	}
	if platform == PlatformGitLab && target.Platform != PlatformGitLab {
		return RemoteSpec{}, "", "", fmt.Errorf("GitLab 降级模式需要 gitlab.com 的完整目标 URL")
	}
	a.emitProgress(id, 1, "检查目标仓库", "running")
	if _, err := a.runQuiet("git", "ls-remote", target.CloneURL); err != nil {
		a.emitProgress(id, 1, "检查目标仓库", "failed")
		return RemoteSpec{}, "", "", fmt.Errorf("目标仓库不可访问，请确认仓库已创建且凭据有效")
	}
	a.emitProgress(id, 1, "检查目标仓库", "done")
	return target, ManagementGitOnly, "", nil
}

func (a *App) SyncRepository(id string) error {
	repo, ok := a.repositoryByID(id)
	if !ok {
		return fmt.Errorf("未找到仓库记录")
	}
	syncLFS := a.store.Settings().SyncLFS
	repo.Status, repo.LastError = "syncing", ""
	if err := a.upsertRepository(repo); err != nil {
		return err
	}
	err := a.performSync(repo, syncLFS)
	if err != nil {
		repo.Status, repo.LastError = "failed", err.Error()
	} else {
		repo.Status, repo.LastSync = "healthy", time.Now()
	}
	if saveErr := a.upsertRepository(repo); saveErr != nil {
		return errors.Join(err, saveErr)
	}
	return err
}

func (a *App) SyncAll() error {
	snapshot := a.store.Snapshot()
	repositories := snapshot.Repositories
	limit := snapshot.Settings.Concurrency
	if limit < 1 || limit > 4 {
		limit = 1
	}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	var errorsMu sync.Mutex
	var messages []string
	for _, repository := range repositories {
		repository := repository
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := a.SyncRepository(repository.ID); err != nil {
				errorsMu.Lock()
				messages = append(messages, repository.Source.DisplayName+": "+err.Error())
				errorsMu.Unlock()
			}
		}()
	}
	wg.Wait()
	if len(messages) > 0 {
		sort.Strings(messages)
		return errors.New(strings.Join(messages, "\n"))
	}
	return nil
}

func (a *App) RemoveRepository(id string, deleteRemote bool) error {
	repo, ok := a.repositoryByID(id)
	if !ok {
		return fmt.Errorf("未找到仓库记录")
	}
	if deleteRemote {
		if repo.ManagementMode != ManagementCLI {
			return fmt.Errorf("该目标由普通 Git 管理，只能解除本地记录")
		}
		environment := a.CheckEnvironment()
		ready := repo.Target.Platform == PlatformGitHub && environment.GitHub.Authenticated
		ready = ready || repo.Target.Platform == PlatformGitLab && environment.GitLab.Authenticated
		if !ready {
			return fmt.Errorf("对应平台 CLI 未登录，不能删除远端仓库")
		}
		fullName := repo.Target.Namespace + "/" + repo.Target.Repository
		var err error
		if repo.Target.Platform == PlatformGitHub {
			err = a.runCommand(id, "gh", "repo", "delete", fullName, "--yes")
		} else {
			err = a.runCommand(id, "glab", "repo", "delete", fullName, "--yes")
		}
		if err != nil {
			return fmt.Errorf("远端仓库删除失败: %w", err)
		}
	}
	err := a.store.Remove(id)
	if err == nil {
		a.emitChanged()
	}
	return err
}

func (a *App) OpenURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("拒绝打开不安全的地址")
	}
	runtime.BrowserOpenURL(a.ctx, u.String())
	return nil
}
