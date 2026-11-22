# GitRepoMirror

![GitRepoMirror project cover](docs/assets/gitrepomirror-cover.png)

GitRepoMirror 是一款面向 Windows 的桌面 Git 仓库镜像工具，用于在 GitHub、GitLab、Gitee、Codeberg、Azure DevOps 以及其他兼容 Git 的托管平台之间建立独立副本，并按需重复同步。

> 完整镜像会执行强制推送，在用于重要仓库前请确认目标仓库不包含需要保留的独立提交。

## 功能

- 在 GitHub、GitLab、Gitee、Codeberg、Azure DevOps 与通用 Git 远端之间同步
- 完整镜像所有普通分支和标签，或仅同步默认分支最新版本
- 默认保留目标端独有的分支和标签，可在设置中开启严格镜像清理
- 自动创建、关联或重命名 GitHub/GitLab/Gitee/Codeberg/Azure DevOps 目标仓库
- 已登录平台在环境状态中显示账户名
- 支持 Private、Internal（仅 GitLab）和 Public 可见性
- 支持单仓库同步和 1–4 路并发批量同步
- 支持 HTTP、HTTPS、SOCKS5 代理和可选 Git LFS 对象同步
- 提供实时命令输出、分步进度、失败记录和系统深浅色主题
- 使用本机 `git`、`gh`、`glab`、`gitee`、`fj`、`az`，应用自身不保存平台 token
- GitLab HTTPS 操作复用本机 `glab` 登录状态，不额外要求 Credential Manager 登录

完整镜像仅推送 `refs/heads/*` 和 `refs/tags/*`，不会复制平台维护的 Pull Request、Merge Request 等只读内部引用。

## 下载与运行

从 [Releases](https://codeberg.org/DerekWangHUZ/GitRepoMirror/releases) 下载 `GitRepoMirror-v2.1-windows-amd64.exe`。

运行要求：

- Windows 10/11 x64
- Git
- Microsoft Edge WebView2 Runtime
- 自动管理 GitHub 仓库需要 GitHub CLI `gh`
- 自动管理 GitLab 仓库需要 GitLab CLI `glab`
- 自动管理 Gitee 仓库需要 Gitee CLI `gitee`
- 自动管理 Codeberg 仓库需要 forgejo CLI `fj`
- 自动管理 Azure DevOps 仓库需要 Azure CLI `az`（含 azure-devops 扩展）
- 同步 Git LFS 对象时需要 Git LFS

应用以单文件便携形式发布，无需安装。下载后可直接运行；Windows 可能显示来源确认提示，确认运行即可。

## 快速开始

1. 安装 Git，并按需要安装 `gh`、`glab`、`gitee`、`fj`、`az` 和 Git LFS。
2. 在应用左下角检查环境状态；若显示凭据失效或缺少令牌，点击对应的登录按钮。
3. 点击“新建镜像”，输入源仓库地址，例如 `owner/repository` 或完整 Git URL。
4. 选择目标平台、命名空间、目标名称、可见性和冲突处理方式。
5. 选择完整镜像或浅克隆模式，然后开始备份。
6. 在实时输出中确认拉取、推送和清理步骤均成功。

平台 CLI 不可用时，可先在目标平台创建空仓库，再在应用中填写完整的 HTTPS、SSH 或 SCP 风格推送地址。

## 同步模式

| 模式 | 行为 | 适用场景 |
| --- | --- | --- |
| 完整镜像 | 强制同步所有普通分支和标签；默认保留目标端多余的普通分支和标签，可在设置中开启清理（严格镜像）；GitHub/普通 Git 使用强制推送，GitLab CLI 管理的目标会临时允许强制推送并在完成后恢复保护 | 迁移、备份、保持目标副本一致 |
| 浅克隆 | 仅同步源仓库默认分支的最新状态 | 只关心当前代码、希望减少流量 |

GitHub 和普通 Git 的完整镜像会使用强制推送。GitLab CLI 管理的目标会在同步期间临时打开匹配保护规则的强制推送权限，推送完成后立即恢复原设置；如果当前账号没有管理分支保护的权限，同步会失败并提示原因。GitLab 仅使用完整 Git URL 管理的目标不会修改平台保护设置，只能进行非强制更新。

清理设置（严格镜像）开启后，目标端存在、但源端没有的普通分支和标签会被删除；关闭时它们会被保留，因此手动推送到目标仓库的分支不会丢失。无论开关状态，同名分支都会按源端内容强制覆盖。

## 数据与安全

设置和镜像映射保存在 `%AppData%\GitRepoMirror\data.json`。配置文件使用临时文件和原子替换写入。

- 平台凭据由官方 CLI 和系统凭据存储管理
- GitLab HTTPS 的 Git 凭据由本机 `glab auth git-credential` 临时提供；镜像到 GitHub 时 GitHub 侧同时复用本机 `gh auth git-credential`；Gitee HTTPS 由本机 `gitee` CLI 的令牌临时生成授权头，均不写入全局 Git 配置，也不落盘
- 命令参数不经过 Shell 字符串拼接
- 日志会隐藏 URL 中的用户名和密码
- 通用 Git 模式不会创建、修改或删除平台项目
- 删除仓库条目仅解除本地绑定，应用永远不会删除远端仓库

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

Windows 产物位于 `build\bin\GitRepoMirror.exe`。
