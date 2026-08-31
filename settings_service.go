package main

import (
	"fmt"
	"net/url"
	"strings"
)

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
	return a.store.SaveSettings(settings)
}
