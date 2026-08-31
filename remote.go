package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	pathPartPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	scpPattern      = regexp.MustCompile(`^([^@\s]+)@([^:\s]+):([^\s]+)$`)
)

func parseRemote(raw string) (RemoteSpec, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return RemoteSpec{}, fmt.Errorf("请输入仓库地址")
	}
	if strings.ContainsAny(raw, "\x00\r\n\t ") {
		return RemoteSpec{}, fmt.Errorf("仓库地址不能包含空白或控制字符")
	}

	if !strings.Contains(raw, "://") && !scpPattern.MatchString(raw) {
		parts := strings.Split(strings.Trim(raw, "/"), "/")
		if len(parts) != 2 || !validPathParts(parts) {
			return RemoteSpec{}, fmt.Errorf("简写地址必须使用 GitHub 的 owner/repository 格式")
		}
		return makeRemote(PlatformGitHub, "github.com", parts, "https://github.com/"+strings.Join(parts, "/")+".git"), nil
	}

	if match := scpPattern.FindStringSubmatch(raw); match != nil {
		host := strings.ToLower(match[2])
		parts, err := splitRepoPath(match[3])
		if err != nil {
			return RemoteSpec{}, err
		}
		return makeRemote(platformForHost(host), host, parts, raw), nil
	}

	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return RemoteSpec{}, fmt.Errorf("仓库 URL 格式无效")
	}
	switch strings.ToLower(u.Scheme) {
	case "https", "ssh":
	default:
		return RemoteSpec{}, fmt.Errorf("仅支持 HTTPS、SSH 或 SCP 风格的仓库地址")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return RemoteSpec{}, fmt.Errorf("仓库 URL 不能包含查询参数或片段")
	}
	parts, err := splitRepoPath(u.EscapedPath())
	if err != nil {
		return RemoteSpec{}, err
	}
	host := strings.ToLower(u.Hostname())
	return makeRemote(platformForHost(host), host, parts, raw), nil
}

func splitRepoPath(value string) ([]string, error) {
	decoded, err := url.PathUnescape(strings.Trim(value, "/"))
	if err != nil {
		return nil, fmt.Errorf("仓库路径编码无效")
	}
	decoded = strings.TrimSuffix(decoded, ".git")
	parts := strings.Split(decoded, "/")
	if len(parts) < 2 || !validPathParts(parts) {
		return nil, fmt.Errorf("仓库地址必须包含命名空间和仓库名")
	}
	return parts, nil
}

func validPathParts(parts []string) bool {
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || !pathPartPattern.MatchString(part) {
			return false
		}
	}
	return true
}

func platformForHost(host string) string {
	switch strings.ToLower(host) {
	case "github.com":
		return PlatformGitHub
	case "gitlab.com":
		return PlatformGitLab
	default:
		return PlatformGeneric
	}
}

func makeRemote(platform, host string, parts []string, cloneURL string) RemoteSpec {
	namespace := strings.Join(parts[:len(parts)-1], "/")
	repository := parts[len(parts)-1]
	path := namespace + "/" + repository
	webURL := "https://" + host + "/" + path
	if platform == PlatformGeneric && (host == "" || strings.Contains(host, "@")) {
		webURL = ""
	}
	return RemoteSpec{
		Platform: platform, Host: host, Namespace: namespace, Repository: repository,
		CloneURL: cloneURL, WebURL: webURL, DisplayName: host + "/" + path,
	}
}

func validateManagedTarget(platform, namespace, name, visibility string) error {
	if platform != PlatformGitHub && platform != PlatformGitLab {
		return fmt.Errorf("目标平台不支持 CLI 管理")
	}
	parts := strings.Split(strings.Trim(namespace, "/"), "/")
	if namespace == "" || !validPathParts(parts) {
		return fmt.Errorf("目标命名空间格式无效")
	}
	if !pathPartPattern.MatchString(name) || name == "." || name == ".." || len(name) > 100 {
		return fmt.Errorf("目标仓库名只能包含字母、数字、点、连字符和下划线，且不超过 100 个字符")
	}
	switch visibility {
	case "private", "internal", "public":
		return nil
	default:
		return fmt.Errorf("目标仓库可见性无效")
	}
}
