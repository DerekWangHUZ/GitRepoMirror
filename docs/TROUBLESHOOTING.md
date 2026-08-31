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

Windows Credential Manager 中可能仍有过期的 HTTPS 凭据。重新运行 `glab auth login` 并选择让 GitLab CLI 配置 Git 凭据，然后再次检查目标仓库访问权限。

## 目标仓库已经存在

创建镜像时选择“关联已有仓库”，或选择“自动添加数字后缀”。只有确认目标仓库专门用于镜像时才应关联；完整镜像会覆盖和清理目标端分支、标签。

## 无法访问源仓库

- 确认 URL 正确，且当前账号对私有仓库有读取权限
- 检查代理设置和防火墙是否允许访问托管平台 443 端口
- 对组织仓库确认 OAuth token 已通过组织 SSO 授权

## Git LFS 同步失败

确认已安装 Git LFS，并验证源端允许下载、目标端允许上传 LFS 对象。普通 Git 引用成功并不代表所有 LFS 对象已成功传输。

## 日志中的命令退出

“命令退出”表示底层 CLI 返回非零状态。请查看它之前的标准错误行；常见原因包括权限不足、仓库保护规则、网络中断和凭据失效。

“命令探测超时或已取消”表示环境、认证或仓库查询在 30 秒内没有完成；请检查网络、代理和平台 CLI 状态后重试。“命令已取消”通常表示应用正在退出。
