// 来源：macUI-SolidGlass 模板本机副本（web/src/macos/sidebar.tsx），复制日期 2026-11-22。
// token 禁止手工微调 —— 悬浮玻璃侧栏骨架（.sidebar/.nav-* 类）来自
// skin/macos/chrome.css；业务内容（环境卡片、导航数据）为 GitRepoMirror 版。
import { Settings, Terminal } from "lucide-react";

import { GitMark } from "@/components/PlatformIcon";
import { TrafficLights } from "@/components/TrafficLights";
import {
  cn,
  environmentRowModel,
  loginButtonLabel,
  managedPlatforms,
} from "@/lib/utils";

/** 悬浮玻璃侧栏：红绿灯 + 标题 + 导航 + 环境卡片。 */
export function Sidebar({
  view,
  onViewChange,
  repoCount,
  env,
  envOk,
  connectedCount,
  onRefreshEnv,
  onLoginPlatform,
  windowActions,
}) {
  const navItems = [
    { id: "repositories", label: "镜像仓库", icon: GitMark, badge: repoCount },
    { id: "logs", label: "运行日志", icon: Terminal },
  ];

  return (
    <aside className="sidebar" aria-label="导航">
      {/* 红绿灯行：交给 Wails 拖动区 */}
      <div className="sidebar__chrome" data-wails-drag>
        <TrafficLights
          maximized={windowActions.maximized}
          onClose={windowActions.onClose}
          onMinimize={windowActions.onMinimize}
          onMaximize={windowActions.onMaximize}
        />
      </div>

      {/* 标题 + 环境摘要：同为拖动区 */}
      <div className="grm-sidebar-identity" data-wails-drag>
        <div className="sidebar__header">
          <span className="sidebar__logo-mark" aria-hidden="true">
            <GitMark size={14} />
          </span>
          <span className="sidebar__logo-text">GitRepoMirror</span>
        </div>
        <div className="grm-env-pill" data-wails-no-drag>
          <i className={cn("grm-dot", envOk ? "ok" : "bad")} />
          <span>
            {envOk
              ? `Git 可用 · ${connectedCount} 个平台已连接`
              : "环境需要处理"}
          </span>
        </div>
      </div>

      <nav className="sidebar__scroll">
        <div className="grm-sidebar-label">工作台</div>
        <NavGroup entries={navItems} view={view} onViewChange={onViewChange} />
      </nav>

      <div className="grm-sidebar-footer">
        <NavGroup
          entries={[{ id: "settings", label: "设置", icon: Settings }]}
          view={view}
          onViewChange={onViewChange}
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

function NavGroup({ entries, view, onViewChange }) {
  return (
    <ul className="nav-group">
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
