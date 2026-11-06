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
	_, finish, err := a.operations.begin(a.commandContext(), id, false)
	if err != nil {
		return Repository{}, err
	}
	defer finish()
	target, management, visibility, err := a.resolveTarget(id, req, source, environment)
	if err != nil {
		return Repository{}, err
	}
	repo := Repository{ID: id, Source: source, Target: target, ManagementMode: management, Visibility: visibility, Mode: req.Mode, Status: "syncing", CreatedAt: time.Now()}
	if err := a.performSync(repo, req.SyncLFS, environment); err != nil {
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
	definition := platformByID(platform)
	ready := definition != nil && definition.Managed
	if ready {
		tool := environment.Tool(platform)
		ready = tool.Installed && tool.Authenticated
	}
	if ready {
		tool := environment.Tool(platform)
		name := strings.TrimSpace(req.TargetName)
		if name == "" {
			name = strings.TrimSpace(req.Prefix) + source.Repository + strings.TrimSpace(req.Suffix)
		}
		namespace := strings.Trim(strings.TrimSpace(req.TargetNamespace), "/")
		if namespace == "" {
			namespace = tool.Login
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
	if definition != nil && target.Platform != platform {
		return RemoteSpec{}, "", "", fmt.Errorf("%s 降级模式需要 %s 的完整目标 URL", definition.Label, definition.Host)
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
	return a.syncRepository(id, false)
}

func (a *App) syncRepository(id string, batchMember bool) error {
	repo, ok := a.repositoryByID(id)
	if !ok {
		return fmt.Errorf("未找到仓库记录")
	}
	_, finish, err := a.operations.begin(a.commandContext(), id, batchMember)
	if err != nil {
		return err
	}
	defer finish()
	syncLFS := a.store.Settings().SyncLFS
	environment := a.CheckEnvironment()
	repo.Status, repo.LastError = "syncing", ""
	if err := a.upsertRepository(repo); err != nil {
		return err
	}
	err = a.performSync(repo, syncLFS, environment)
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
	finishBatch, err := a.operations.beginBatch()
	if err != nil {
		return err
	}
	defer finishBatch()
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
			if err := a.syncRepository(repository.ID, true); err != nil {
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

func (a *App) RemoveRepository(id string) error {
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
