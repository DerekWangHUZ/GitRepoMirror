package main

import (
	"fmt"
	"strings"
)

func (a *App) CheckEnvironment() EnvironmentStatus {
	return EnvironmentStatus{
		Git: installedTool("git"), GitLFS: installedTool("git-lfs"),
		GitHub: a.checkPlatformTool(PlatformGitHub), GitLab: a.checkPlatformTool(PlatformGitLab),
	}
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
