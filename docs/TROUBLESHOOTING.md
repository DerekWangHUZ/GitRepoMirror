# 故障排查

## 配置损坏后以默认设置启动

如果 `%AppData%\GitRepoMirror\data.json` 不是有效 JSON，应用不会覆盖原文件，而会将其改名为同目录下的 `data.corrupt-日期时间.json` 并使用默认设置启动。具体备份路径会显示在运行日志中。确认备份内容后，可重新保存设置并重新建立镜像记录。

## 窗口文字和图标模糊

请使用最新构建。GitRepoMirror 已声明 Windows Per-Monitor V2 DPI 感知，可在 125%、150% 等缩放比例以及多显示器切换时由 WebView2 直接按目标 DPI 绘制，而不是让 Windows 拉伸整张窗口位图。

升级后需要完全退出旧进程再启动新 EXE。如果仍然模糊，请检查是否曾在 EXE“兼容性”属性中手动覆盖高 DPI 缩放行为；这类外部覆盖会优先于应用声明。

## 显示“凭据失效”或“缺少令牌”

在应用中点击对应平台的重新登录按钮，或在终端执行：

```powershell
gh auth login --hostname github.com --web --git-protocol https
glab auth login --hostname gitlab.com --device --git-protocol https
```

然后使用 `gh auth status` 和 `glab auth status` 验证。不要把输出中的 token 分享到 Issue。

## GitLab 返回 HTTP Basic: Access denied

最新版应用会在 GitLab HTTPS 操作中复用 `glab auth git-credential`，并临时屏蔽系统中可能保存旧凭据的其他 Git helper，不会要求再登录一个独立的 GitLab 连接。先确认：

```powershell
glab auth status --hostname gitlab.com
```

如果仍返回 401/403，请检查目标仓库是否属于当前账号、是否被安排删除，以及分支保护规则。不要把 token 写入仓库 URL 或提交到配置文件。

## GitLab 推送时显示 `You are not allowed to push code to this project`

这通常不是 CLI 登录失败。若目标项目的 API 信息包含 `marked_for_deletion_at` 或项目名称带有 `deletion_scheduled`，GitLab 会拒绝推送，即使当前账号仍是项目所有者。请在 GitLab 页面撤销删除计划，或换用一个新的目标仓库名称。

应用日志中的 `Cannot prompt because user interactivity has been disabled` 是后台任务拒绝弹出凭据窗口的提示；应结合前面的 HTTP 状态和远端错误判断真正原因。

## 目标仓库已经存在

创建镜像时选择“关联已有仓库”，或选择“自动添加数字后缀”。只有确认目标仓库专门用于镜像时才应关联；完整镜像会强制覆盖目标端同名分支和标签。默认设置下，目标端独有的分支和标签会被保留；若在设置中开启了“清理目标端多余分支和标签”，它们会在同步时被删除。

## GitLab 受保护分支拒绝强制推送

GitLab CLI 管理的目标会在完整镜像期间临时允许匹配保护规则的强制推送，推送完成后自动恢复原设置。若提示没有权限修改保护分支，当前账号需要具备项目的分支保护管理权限；也可以在 GitLab 中手动允许强制推送后重试。仅使用完整 Git URL 的 GitLab 目标不会修改平台设置，只支持非强制更新，因此目标存在独立提交或源端发生历史重写时，需要先清理目标或手动调整保护策略。

## 无法访问源仓库

- 确认 URL 正确，且当前账号对私有仓库有读取权限
- 检查代理设置和防火墙是否允许访问托管平台 443 端口
- 对组织仓库确认 OAuth token 已通过组织 SSO 授权

## Git LFS 同步失败

确认已安装 Git LFS，并验证源端允许下载、目标端允许上传 LFS 对象。普通 Git 引用成功并不代表所有 LFS 对象已成功传输。

## 日志中的命令退出

“命令退出”表示底层 CLI 返回非零状态。请查看它之前的标准错误行；常见原因包括权限不足、仓库保护规则、网络中断和凭据失效。

“命令探测超时或已取消”表示环境、认证或仓库查询在 30 秒内没有完成；请检查网络、代理和平台 CLI 状态后重试。“命令已取消”通常表示应用正在退出。
