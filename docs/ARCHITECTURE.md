# 架构说明

GitRepoMirror 是一个 Wails v2 桌面应用。Go 后端负责状态、验证、进程执行和文件持久化，原生 JavaScript 前端负责界面与事件展示。

## 主要组件

- `app.go`：稳定的 Wails API 门面和窗口生命周期
- `mirror_service.go`、`sync_engine.go`：镜像用例编排与 Git/LFS 同步流程
- `environment_service.go`、`platform.go`：本机工具认证和平台仓库管理
- `command_runner.go`：命令查找、环境、输出流和参数脱敏边界
- `repository_store.go`、`store.go`：并发内存快照和 JSON 原子持久化
- `events.go`：可替换的运行时日志、进度和数据变更事件边界
- `frontend/src/ui`：状态、模板、渲染、操作控制和静态窗口外壳

Wails 公共方法、JSON DTO 和运行时事件只由 `App` 暴露。内部模块可重构，但这些边界以及 `%AppData%\GitRepoMirror\data.json` 的格式必须保持向后兼容。

## 完整镜像流程

1. 解析并验证源端和目标端。
2. 通过 `gh` 或 `glab` 查询、创建或关联托管平台项目。
3. 在系统临时目录执行裸镜像克隆。
4. 强制推送并清理目标端的 `refs/heads/*` 与 `refs/tags/*`。
5. 按需拉取和推送全部 Git LFS 对象。
6. 保存成功或失败状态，并清理临时目录。

所有命令使用参数数组直接启动，不通过 Shell 拼接。任务环境设置 `GIT_TERMINAL_PROMPT=0`，网络代理由应用设置按任务注入。

## 数据边界

应用只保存设置、远端元数据、同步状态和错误摘要，不保存平台 token。批量同步使用有界并发，默认并发数为 2，允许范围为 1–4。

## 回归保障

Go 测试使用命令 hook 和 `EventSink` fake 验证命令、状态与事件序列。前端使用 Vitest 验证纯模板和状态契约，并使用 Playwright 在固定视口下比较浅色、深色、主要页面、弹窗和响应式视觉基线。
