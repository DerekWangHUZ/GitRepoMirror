# GitRepoMirror

![GitRepoMirror project cover](docs/assets/gitrepomirror-cover.png)

GitRepoMirror 是一款面向 Windows 的桌面 Git 仓库镜像工具，用于在 GitHub、GitLab 以及其他兼容 Git 的托管平台之间建立独立副本，并按需重复同步。

> 完整镜像会执行强制推送，首次用于重要仓库前请确认目标仓库不包含需要保留的独立提交。

## 功能

- 在 GitHub、GitLab 与通用 Git 远端之间同步
- 完整镜像所有普通分支和标签，或仅同步默认分支最新版本
- 自动创建、关联或重命名 GitHub/GitLab 目标仓库
- 支持 Private、Internal 和 Public 可见性
- 支持单仓库同步和 1–4 路并发批量同步
- 支持 HTTP、HTTPS、SOCKS5 代理和可选 Git LFS 对象同步
- 提供实时命令输出、分步进度、失败记录和系统深浅色主题
- 使用本机 `git`、`gh`、`glab`，应用自身不保存平台 token

完整镜像仅推送 `refs/heads/*` 和 `refs/tags/*`，不会复制平台维护的 Pull Request、Merge Request 等只读内部引用。

## 下载与运行

从 [Releases](https://gitlab.com/DerekWangHUZ/GitRepoMirror/-/releases) 下载最新的 `GitRepoMirror.exe`。

运行要求：

- Windows 10/11 x64
- Git
- Microsoft Edge WebView2 Runtime
- 自动管理 GitHub 仓库时需要 GitHub CLI `gh`
- 自动管理 GitLab 仓库时需要 GitLab CLI `glab`
- 同步 Git LFS 对象时需要 Git LFS

本项目当前不提供安装程序。下载后可直接运行；Windows 首次启动时可能显示来源确认提示。

## 快速开始

1. 安装 Git，并按需要安装 `gh`、`glab` 和 Git LFS。
2. 在应用左下角检查环境状态；若显示凭据失效或缺少令牌，点击对应的登录按钮。
3. 点击“新建镜像”，输入源仓库地址，例如 `owner/repository` 或完整 Git URL。
4. 选择目标平台、命名空间、目标名称、可见性和冲突处理方式。
5. 选择完整镜像或浅克隆模式，然后开始备份。
6. 在实时输出中确认拉取、推送和清理步骤均成功。

平台 CLI 不可用时，可先在目标平台创建空仓库，再在应用中填写完整的 HTTPS、SSH 或 SCP 风格推送地址。

## 同步模式

| 模式 | 行为 | 适用场景 |
| --- | --- | --- |
| 完整镜像 | 强制同步并清理目标端多余的普通分支和标签 | 迁移、备份、保持目标副本一致 |
| 浅克隆 | 仅同步源仓库默认分支的最新状态 | 只关心当前代码、希望减少流量 |

完整镜像会使用强制推送。不要把包含目标端独立提交的仓库配置为镜像目标。

## 数据与安全

设置和镜像映射保存在 `%AppData%\GitRepoMirror\data.json`。配置文件使用临时文件和原子替换写入。

- 平台凭据由官方 CLI 和系统凭据存储管理
- 命令参数不经过 Shell 字符串拼接
- 日志会隐藏 URL 中的用户名和密码
- 通用 Git 模式不会创建、修改或删除平台项目
- 只有由平台 CLI 管理的目标仓库才允许从应用内删除远端

更多信息见 [安全策略](SECURITY.md) 和 [故障排查](docs/TROUBLESHOOTING.md)。

## 本地开发

需要 Go 1.24+、Node.js 20+ 和 Wails CLI v2.11+。

```powershell
cd frontend
npm install
npm run build
cd ..

go test ./...
go vet ./...
wails build -clean -platform windows/amd64 -o GitRepoMirror.exe
```

Windows 产物位于 `build\bin\GitRepoMirror.exe`。项目结构和同步过程见 [架构说明](docs/ARCHITECTURE.md)，参与开发前请阅读 [贡献指南](CONTRIBUTING.md)。

## 项目状态

项目持续迭代中。已知限制与各版本变更记录见 [CHANGELOG.md](CHANGELOG.md)。
