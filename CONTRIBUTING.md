# 贡献指南

感谢你参与 GitRepoMirror。提交代码前，请先搜索现有 Issue，避免重复工作。

## 开发环境

- Go 1.24+
- Node.js 20+
- Wails CLI 2.11+
- Windows 10/11 与 WebView2 Runtime（运行桌面端时）

安装前端依赖后，可分别运行前端开发服务器和 Wails 开发模式。提交前必须执行：

```powershell
go test ./...
go vet ./...

cd frontend
npm test
npm run lint
npm run format:check
npm run test:visual
npm run build
```

首次运行视觉测试前执行 `npx playwright install chromium`。只有在确认界面变更符合需求后，才可运行 `npm run test:visual:update` 更新截图基线。

## 提交要求

- 每个提交只处理一个清晰主题，提交信息使用祈使语气并说明结果
- 新增或修改后端行为时补充 Go 测试
- 修改前端模板、样式或交互时补充 Vitest 测试，并检查 Playwright 截图差异
- 不提交 token、带认证信息的 URL、本地配置、构建产物或依赖目录
- 不绕过仓库权限、组织策略或平台保护规则
- 涉及强制推送、远端删除或凭据处理的改动必须说明安全边界

## 合并请求

合并请求应包含问题背景、实现方式、验证步骤和必要的界面截图。若改动影响用户可见行为，请同步更新 README、故障排查或变更记录。
