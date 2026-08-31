package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx         context.Context
	mu          sync.RWMutex
	data        StoreData
	quietHook   func(dir, name string, args ...string) (string, error)
	commandHook func(id, dir, name string, args ...string) error
}

func NewApp() *App {
	return &App{data: defaultData()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	data, err := loadData()
	if err != nil {
		a.emitLog("", "error", err.Error())
		return
	}
	a.mu.Lock()
	a.data = data
	a.mu.Unlock()
}

func (a *App) shutdown(_ context.Context) {}

func (a *App) GetData() StoreData {
	a.mu.RLock()
	result := a.data
	result.Repositories = append([]Repository(nil), a.data.Repositories...)
	a.mu.RUnlock()
	sort.SliceStable(result.Repositories, func(i, j int) bool {
		return result.Repositories[i].CreatedAt.After(result.Repositories[j].CreatedAt)
	})
	return result
}

func (a *App) SaveSettings(settings Settings) error {
	settings.Proxy = strings.TrimSpace(settings.Proxy)
	settings.Prefix = strings.TrimSpace(settings.Prefix)
	settings.Suffix = strings.TrimSpace(settings.Suffix)
	if settings.Proxy != "" {
		u, err := url.Parse(settings.Proxy)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "socks5") {
			return fmt.Errorf("代理地址需使用 http://、https:// 或 socks5://")
		}
	}
	if settings.Concurrency < 1 || settings.Concurrency > 4 {
		return fmt.Errorf("并发数必须为 1 到 4")
	}
	a.mu.Lock()
	a.data.Settings = settings
	err := saveData(a.data)
	a.mu.Unlock()
	return err
}

func (a *App) CheckEnvironment() EnvironmentStatus {
	status := EnvironmentStatus{}
	status.Git = installedTool("git")
	status.GitLFS = installedTool("git-lfs")
	status.GitHub = a.checkPlatformTool(PlatformGitHub)
	status.GitLab = a.checkPlatformTool(PlatformGitLab)
	return status
}

func installedTool(name string) ToolStatus {
	path, err := findExecutable(name)
	if err != nil {
		return ToolStatus{Message: "未安装"}
	}
	return ToolStatus{Installed: true, Path: path, Message: "可用"}
}

func (a *App) checkPlatformTool(platform string) ToolStatus {
	name := platformCLI(platform)
	status := installedTool(name)
	if !status.Installed {
		if platform == PlatformGitHub {
			status.Message = "未安装 gh，可使用完整 URL 降级同步"
		} else {
			status.Message = "未安装 glab，可使用完整 URL 降级同步"
		}
		return status
	}
	var authArgs, loginArgs []string
	if platform == PlatformGitHub {
		authArgs = []string{"auth", "status", "--hostname", "github.com"}
		loginArgs = []string{"api", "user", "--hostname", "github.com", "--jq", ".login"}
	} else {
		authArgs = []string{"auth", "status", "--hostname", "gitlab.com"}
		loginArgs = []string{"api", "user", "--hostname", "gitlab.com", "--jq", ".username"}
	}
	output, err := a.runQuiet(name, authArgs...)
	if err != nil {
		status.AuthState, status.Message = classifyPlatformAuthFailure(platform, output)
		return status
	}
	status.Authenticated = true
	status.AuthState = "authenticated"
	if login, err := a.runQuiet(name, loginArgs...); err == nil {
		status.Login = strings.TrimSpace(login)
	}
	status.Message = "已登录"
	return status
}

func classifyPlatformAuthFailure(platform, output string) (string, string) {
	label := "平台 CLI"
	if platform == PlatformGitHub {
		label = "GitHub CLI"
	} else if platform == PlatformGitLab {
		label = "GitLab CLI"
	}
	lower := strings.ToLower(output)
	if strings.Contains(lower, "no access or refresh token") || strings.Contains(lower, "no token found") {
		return "missing", label + " 缺少访问令牌，请重新登录"
	}
	if strings.Contains(lower, "token") && (strings.Contains(lower, "invalid") || strings.Contains(lower, "expired")) {
		return "invalid", label + " 凭据已失效，请重新登录"
	}
	return "unauthenticated", label + " 尚未登录或认证检查失败"
}

func platformLoginArgs(platform, host string) []string {
	flow := "--web"
	if platform == PlatformGitLab {
		flow = "--device"
	}
	return []string{"auth", "login", "--hostname", host, flow, "--git-protocol", "https"}
}

func (a *App) StartLogin(platform string) error {
	name := platformCLI(platform)
	if name == "" {
		return fmt.Errorf("该平台不支持 CLI 登录")
	}
	path, err := findExecutable(name)
	if err != nil {
		return fmt.Errorf("未找到 %s", name)
	}
	host := "github.com"
	if platform == PlatformGitLab {
		host = "gitlab.com"
	}
	return startLoginCommand(path, platformLoginArgs(platform, host)...)
}

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
	repo := Repository{
		ID: id, Source: source, Target: target, ManagementMode: management,
		Visibility: visibility, Mode: req.Mode, Status: "syncing", CreatedAt: time.Now(),
	}
	if err := a.performSync(repo, req.SyncLFS); err != nil {
		repo.Status = "failed"
		repo.LastError = err.Error()
		if saveErr := a.upsertRepository(repo); saveErr != nil {
			return repo, errors.Join(err, saveErr)
		}
		return repo, err
	}
	repo.Status = "healthy"
	repo.LastSync = time.Now()
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
	a.mu.RLock()
	syncLFS := a.data.Settings.SyncLFS
	a.mu.RUnlock()
	repo.Status = "syncing"
	repo.LastError = ""
	if err := a.upsertRepository(repo); err != nil {
		return err
	}
	err := a.performSync(repo, syncLFS)
	if err != nil {
		repo.Status = "failed"
		repo.LastError = err.Error()
	} else {
		repo.Status = "healthy"
		repo.LastSync = time.Now()
	}
	if saveErr := a.upsertRepository(repo); saveErr != nil {
		return errors.Join(err, saveErr)
	}
	return err
}

func (a *App) SyncAll() error {
	a.mu.RLock()
	repositories := append([]Repository(nil), a.data.Repositories...)
	limit := a.data.Settings.Concurrency
	a.mu.RUnlock()
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
	a.mu.Lock()
	filtered := make([]Repository, 0, len(a.data.Repositories)-1)
	for _, item := range a.data.Repositories {
		if item.ID != id {
			filtered = append(filtered, item)
		}
	}
	a.data.Repositories = filtered
	err := saveData(a.data)
	a.mu.Unlock()
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

func (a *App) Minimise() { runtime.WindowMinimise(a.ctx) }

func (a *App) ToggleMaximise() {
	if runtime.WindowIsMaximised(a.ctx) {
		runtime.WindowUnmaximise(a.ctx)
	} else {
		runtime.WindowMaximise(a.ctx)
	}
}

func (a *App) Close() { runtime.Quit(a.ctx) }

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
	environment := a.CheckEnvironment()
	clone := cloneCommand(repo.Source, cloneDir, repo.Mode, environment)
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
		err = a.runCommandIn(repo.ID, cloneDir, "git", "push", "--force", repo.Target.CloneURL, "HEAD:refs/heads/"+strings.TrimSpace(branch))
		if err != nil {
			a.emitProgress(repo.ID, 3, "推送目标仓库", "failed")
			return fmt.Errorf("推送目标仓库失败: %w", err)
		}
	} else {
		err := a.runCommandIn(repo.ID, cloneDir, "git", "push", "--force", "--prune", repo.Target.CloneURL,
			"+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*")
		if err != nil {
			a.emitProgress(repo.ID, 3, "推送目标仓库", "failed")
			return fmt.Errorf("推送目标仓库失败: %w", err)
		}
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

func (a *App) repositoryByID(id string) (Repository, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, repository := range a.data.Repositories {
		if repository.ID == id {
			return repository, true
		}
	}
	return Repository{}, false
}

func (a *App) upsertRepository(repository Repository) error {
	a.mu.Lock()
	found := false
	for i := range a.data.Repositories {
		if a.data.Repositories[i].ID == repository.ID {
			a.data.Repositories[i] = repository
			found = true
			break
		}
	}
	if !found {
		a.data.Repositories = append(a.data.Repositories, repository)
	}
	err := saveData(a.data)
	a.mu.Unlock()
	if err == nil {
		a.emitChanged()
	}
	return err
}

func (a *App) commandEnv() []string {
	environment := os.Environ()
	a.mu.RLock()
	proxy := a.data.Settings.Proxy
	a.mu.RUnlock()
	if proxy != "" {
		environment = append(environment, "HTTP_PROXY="+proxy, "HTTPS_PROXY="+proxy, "ALL_PROXY="+proxy)
	}
	return append(environment, "GIT_TERMINAL_PROMPT=0")
}

func (a *App) runQuiet(name string, args ...string) (string, error) {
	return a.runQuietIn("", name, args...)
}

func (a *App) runQuietIn(dir, name string, args ...string) (string, error) {
	if a.quietHook != nil {
		return a.quietHook(dir, name, args...)
	}
	resolved, err := findExecutable(name)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(a.commandContext(), resolved, args...)
	cmd.Dir = dir
	cmd.Env = a.commandEnv()
	hideCommandWindow(cmd)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (a *App) runCommand(id, name string, args ...string) error {
	return a.runCommandIn(id, "", name, args...)
}

func (a *App) runCommandIn(id, dir, name string, args ...string) error {
	a.emitLog(id, "command", "$ "+name+" "+strings.Join(redactArgs(args), " "))
	if a.commandHook != nil {
		return a.commandHook(id, dir, name, args...)
	}
	resolved, err := findExecutable(name)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(a.commandContext(), resolved, args...)
	cmd.Dir = dir
	cmd.Env = a.commandEnv()
	hideCommandWindow(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	var wg sync.WaitGroup
	stream := func(reader io.Reader, level string) {
		defer wg.Done()
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			a.emitLog(id, level, scanner.Text())
		}
	}
	wg.Add(2)
	go stream(stdout, "info")
	go stream(stderr, "stderr")
	wg.Wait()
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("命令退出: %w", err)
	}
	return nil
}

func redactArgs(args []string) []string {
	result := append([]string(nil), args...)
	for i, argument := range result {
		if parsed, err := url.Parse(argument); err == nil && parsed.User != nil {
			parsed.User = url.User("***")
			result[i] = parsed.String()
		}
	}
	return result
}

func (a *App) commandContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

func (a *App) emitLog(id, level, message string) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "log", LogEvent{RepositoryID: id, Level: level, Message: message, Time: time.Now().Format("15:04:05")})
	}
}

func (a *App) emitProgress(id string, step int, label, status string) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "progress", ProgressEvent{RepositoryID: id, Step: step, Label: label, Status: status})
	}
}

func (a *App) emitChanged() {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "repositories-changed")
	}
}

func newID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(bytes)
}
