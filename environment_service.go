package main

import (
	"fmt"
	"strings"
)

func (a *App) CheckEnvironment() EnvironmentStatus {
	status := EnvironmentStatus{
		Git: installedTool(a, "git"), GitLFS: installedTool(a, "git-lfs"),
	}
	// 注册表驱动：每个平台定义通过 StatusField 写入自己的状态槽位，
	// 新平台只需要提供定义，不再维护手写 switch。
	for _, definition := range managedPlatforms() {
		if definition.StatusField == nil {
			continue
		}
		*definition.StatusField(&status) = a.checkPlatformTool(definition)
	}
	return status
}

func installedTool(app *App, name string) ToolStatus {
	path, err := app.resolveExecutable(name)
	if err != nil {
		return ToolStatus{Message: "未安装"}
	}
	return ToolStatus{Installed: true, Path: path, Message: "可用"}
}

func (a *App) checkPlatformTool(definition *platformDefinition) ToolStatus {
	name := definition.CLI
	status := installedTool(a, name)
	if !status.Installed {
		status.Message = fmt.Sprintf("未安装 %s，可使用完整 URL 降级同步", name)
		return status
	}
	output, err := a.runQuiet(name, definition.AuthStatusArgs()...)
	if err != nil {
		status.AuthState, status.Message = classifyPlatformAuthFailure(definition, output)
		return status
	}
	status.Authenticated = true
	status.AuthState = "authenticated"
	if definition.LoginProbeArgs != nil {
		if login, err := a.runQuiet(name, definition.LoginProbeArgs()...); err == nil {
			status.Login = definition.ParseLogin(login)
		}
	} else {
		status.Login = definition.ParseLogin(output)
	}
	if status.Login == "" {
		a.emitLog("", "warning", definition.Label+" CLI 登录名解析失败，原始输出前 200 字符: "+truncateForLog(output))
	}
	status.Message = "已登录"
	return status
}

func truncateForLog(output string) string {
	output = strings.TrimSpace(output)
	if len(output) > 200 {
		return output[:200] + "..."
	}
	return output
}

func classifyPlatformAuthFailure(definition *platformDefinition, output string) (string, string) {
	label := definition.Label + " CLI"
	lower := strings.ToLower(output)
	if strings.Contains(lower, "no access or refresh token") || strings.Contains(lower, "no token found") {
		return "missing", label + " 缺少访问令牌，请重新登录"
	}
	if strings.Contains(lower, "token") && (strings.Contains(lower, "invalid") || strings.Contains(lower, "expired") || strings.Contains(lower, "already used")) {
		return "invalid", label + " 凭据已失效，请重新登录"
	}
	return "unauthenticated", label + " 尚未登录或认证检查失败"
}

func (a *App) StartLogin(platform string) error {
	definition := platformByID(platform)
	if definition == nil || definition.LoginArgs == nil {
		return fmt.Errorf("该平台不支持 CLI 登录")
	}
	path, err := a.resolveExecutable(definition.CLI)
	if err != nil {
		return fmt.Errorf("未找到 %s", definition.CLI)
	}
	return startLoginCommand(path, definition.LoginArgs(definition.Host)...)
}
