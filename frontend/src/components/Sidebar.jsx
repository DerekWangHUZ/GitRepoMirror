// 来源：macUI-SolidGlass 模板本机副本（web/src/macos/sidebar.tsx），复制日期 2026-11-22。
// token 禁止手工微调 —— 悬浮玻璃侧栏骨架（.sidebar/.nav-* 类）来自
// skin/macos/chrome.css；业务内容（环境卡片、导航数据）为 GitRepoMirror 版。
//
// 五段解剖对齐模板规范 S3：
// sidebar__chrome 40px 行 → logo header → 搜索钮(30px 清玻璃) →
// sidebar__scroll → footer(border-top)。
import { Search, Settings, Terminal } from "lucide-react";

import { GitMark } from "@/components/PlatformIcon";
import { TrafficLights } from "@/components/TrafficLights";
import { Kbd } from "@/components/ui/kbd";
import {
  cn,
  environmentRowModel,
  loginButtonLabel,
  managedPlatforms,
} from "@/lib/utils";

/** 悬浮玻璃侧栏：红绿灯 + 标题 + 搜索钮 + 导航 + 环境卡片。 */
export function Sidebar({
  view,
  onViewChange,
  onOpenCommand,
  repoCount,
  env,
  envOk,
  connectedCount,
  onRefreshEnv,
  onLoginPlatform,
  windowActions,
}) {
  // envOk / connectedCount 汇总进侧栏环境徽标（grm-env-pill）。
  const envSummary = envOk
    ? `Git 可用 · ${connectedCount} 个平台已连接`
    : "环境需要处理";
  const navItems = [
    { id: "repositories", label: "镜像仓库", icon: GitMark, badge: repoCount },
    { id: "logs", label: "运行日志", icon: Terminal },
  ];

  return (
    <aside className="sidebar" aria-label="导航">
      {/* 第一段：红绿灯行（拖动区） */}
      <div className="sidebar__chrome" data-wails-drag>
        <TrafficLights
          maximized={windowActions.maximized}
          onClose={windowActions.onClose}
          onMinimize={windowActions.onMinimize}
          onMaximize={windowActions.onMaximize}
        />
      </div>

      {/* 第二段：logo header + 环境徽标（拖动区） */}
      <div className="grm-sidebar-identity" data-wails-drag>
        <div className="sidebar__header">
          <span className="sidebar__logo-mark" aria-hidden="true">
            <GitMark size={14} />
          </span>
          <span className="sidebar__logo-text">GitRepoMirror</span>
        </div>
        <div className="grm-env-pill" data-wails-no-drag>
          <i className={cn("grm-dot", envOk ? "ok" : "bad")} />
          <span>{envSummary}</span>
        </div>
      </div>

      {/* 第三段：搜索钮（30px 清玻璃，打开命令面板） */}
      <div className="sidebar__search-wrap">
        <button
          type="button"
          className="sidebar__search-btn"
          onClick={onOpenCommand}
        >
          <Search className="size-3.5 shrink-0" />
          <span className="flex-1 truncate text-left">快速跳转或执行命令…</span>
          <Kbd className="text-[10px]">⌘K</Kbd>
        </button>
      </div>

      {/* 第四段：导航滚动区 */}
      <nav className="sidebar__scroll">
        <div className="nav-section-title">工作台</div>
        <NavGroup entries={navItems} view={view} onViewChange={onViewChange} />
      </nav>

      {/* 第五段：footer（border-top）—— 设置 + 环境卡片 */}
      <div className="sidebar__footer">
        <NavGroup
          entries={[{ id: "settings", label: "设置", icon: Settings }]}
          view={view}
          onViewChange={onViewChange}
          compact
        />
        <EnvironmentCard
          env={env}
          onRefreshEnv={onRefreshEnv}
          onLoginPlatform={onLoginPlatform}
        />
      </div>
    </aside>
  );
}

function NavGroup({ entries, view, onViewChange, compact }) {
  return (
    <ul className={cn("nav-group", compact && "nav-group--compact")}>
      {entries.map((entry) => {
        const Icon = entry.icon;
        const selected = entry.id === view;
        return (
          <li
            key={entry.id}
            className={cn("nav-item", selected && "is-selected")}
          >
            <button
              type="button"
              className="nav-link"
              aria-current={selected ? "page" : undefined}
              onClick={() => onViewChange(entry.id)}
            >
              <span className="nav-icon" aria-hidden="true">
                <Icon size={16} />
              </span>
              <span className="nav-label">{entry.label}</span>
              {typeof entry.badge === "number" && (
                <span className="nav-badge">{entry.badge}</span>
              )}
            </button>
          </li>
        );
      })}
    </ul>
  );
}

function EnvironmentCard({ env, onRefreshEnv, onLoginPlatform }) {
  const platforms = managedPlatforms();
  return (
    <div className="grm-env-card">
      <div className="grm-env-row grm-env-title">
        <span>环境状态</span>
        <button
          type="button"
          aria-label="刷新环境状态"
          title="刷新环境状态"
          onClick={onRefreshEnv}
          className="hover:text-foreground"
        >
          ↻
        </button>
      </div>
      <EnvironmentRow label="Git" tool={env.git} />
      {platforms.map((platform) => (
        <EnvironmentRow
          key={platform.id}
          label={`${platform.label} CLI`}
          tool={env[platform.id]}
          platform={platform.id}
        />
      ))}
      <div className="grm-login-actions">
        {platforms.map((platform) => {
          const tool = env[platform.id] || {};
          if (!tool.installed || tool.authenticated) return null;
          return (
            <button
              key={platform.id}
              type="button"
              className="login-button"
              data-login={platform.id}
              onClick={() => onLoginPlatform(platform.id)}
            >
              {loginButtonLabel(tool, platform.id)}
            </button>
          );
        })}
      </div>
    </div>
  );
}

function EnvironmentRow({ label, tool, platform = "" }) {
  const row = environmentRowModel(label, tool, platform);
  return (
    <div className="grm-env-row" title={row.message}>
      <i className={cn("grm-dot", row.ok ? "ok" : "bad")} />
      <span>{row.label}</span>
      <b>{row.detail}</b>
    </div>
  );
}
