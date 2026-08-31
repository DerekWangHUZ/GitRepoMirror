import { icons } from "./icons.js";
import { escapeHTML, formatTime, platformLabel } from "./utils.js";

const platformIcon = (platform) => icons[platform] || icons.generic;

export function repositoryView(repositories, busy) {
  return `<div class="page-head"><div><h1>镜像仓库</h1><p>在不同 Git 托管平台之间保存独立副本</p></div><div class="head-actions"><button class="button secondary" id="syncAll" data-action="sync-all" ${!repositories.length ? "disabled" : ""}>${icons.sync} 全部同步</button><button class="button primary" id="newMirror" data-action="new-mirror">${icons.plus} 新建镜像</button></div></div>${repositories.length ? `<div class="repo-grid">${repositories.map((repo) => repositoryCard(repo, busy)).join("")}</div>` : emptyState()}`;
}

export function repositoryCard(repo, busy) {
  const status = repo.status || "healthy";
  const labels = { healthy: "正常", syncing: "同步中", failed: "失败" };
  const isBusy = busy.has(repo.id) || status === "syncing";
  return `<article class="repo-card ${status}"><div class="repo-top"><div class="repo-icon ${escapeHTML(repo.source.platform)}">${platformIcon(repo.source.platform)}</div><div class="repo-title"><h3>${escapeHTML(repo.source.displayName)}</h3>${repo.target.webUrl ? `<button data-open="${escapeHTML(repo.target.webUrl)}">${escapeHTML(repo.target.displayName)} ${icons.external}</button>` : `<span class="target-name">${escapeHTML(repo.target.displayName)}</span>`}</div><span class="status ${status}"><i></i>${labels[status] || status}</span></div><div class="repo-meta"><span>平台 <b>${platformLabel(repo.source.platform)} → ${platformLabel(repo.target.platform)}</b></span><span>模式 <b>${repo.mode === "shallow" ? "浅克隆" : "完整镜像"}</b></span><span>上次同步 <b>${formatTime(repo.lastSync)}</b></span></div>${repo.lastError ? `<div class="error-line" title="${escapeHTML(repo.lastError)}">${escapeHTML(repo.lastError)}</div>` : ""}<div class="repo-actions"><button class="button sync-button" data-sync="${repo.id}" ${isBusy ? "disabled" : ""}>${icons.sync} ${isBusy ? "同步中…" : "立即同步"}</button><button class="icon-button danger" data-remove="${repo.id}" title="解除绑定">${icons.trash}</button></div></article>`;
}

export function emptyState() {
  return `<div class="empty"><div class="empty-visual"><div>${icons.git}</div><span>${icons.sync}</span></div><h2>还没有镜像仓库</h2><p>创建第一个仓库镜像，保存分支、Tag 和提交历史。</p><button class="button primary" data-action="new-mirror">${icons.plus} 创建第一个镜像</button></div>`;
}

export const logLine = (log) =>
  `<div class="log-line ${escapeHTML(log.level)}"><time>${escapeHTML(log.time)}</time><span>${escapeHTML(log.message)}</span></div>`;

export function logsView(logs) {
  const content = logs.length
    ? logs.map(logLine).join("")
    : '<div class="log-empty">命令输出将在这里显示</div>';
  return `<div class="page-head"><div><h1>运行日志</h1><p>实时查看 Git 与平台 CLI 输出</p></div><button class="button secondary" id="clearLogs" data-action="clear-logs">清空日志</button></div><div class="log-panel">${content}</div>`;
}

export function settingsView(settings = {}) {
  return `<div class="page-head"><div><h1>设置</h1><p>配置网络、命名与同步行为</p></div></div><form class="settings-form" id="settingsForm"><section class="setting-section"><h3>网络</h3><label>独立代理地址<span>留空使用系统网络。支持 HTTP、HTTPS 和 SOCKS5。</span><input name="proxy" value="${escapeHTML(settings.proxy || "")}" placeholder="http://127.0.0.1:7890"></label></section><section class="setting-section"><h3>默认命名</h3><div class="two-col"><label>名称前缀<input name="prefix" value="${escapeHTML(settings.prefix || "")}" placeholder="mirror-"></label><label>名称后缀<input name="suffix" value="${escapeHTML(settings.suffix || "")}" placeholder="-backup"></label></div></section><section class="setting-section"><h3>同步</h3><label>批量同步并发数<span>建议保持 1–2，避免触发托管平台频率限制。</span><select name="concurrency">${[1, 2, 3, 4].map((number) => `<option ${number === settings.concurrency ? "selected" : ""}>${number}</option>`).join("")}</select></label><label class="check-row"><input type="checkbox" name="syncLfs" ${settings.syncLfs ? "checked" : ""}><span><strong>默认同步 Git LFS 对象</strong><small>仅在已安装 Git LFS 时生效</small></span></label></section><button class="button primary" type="submit">保存设置</button></form>`;
}

export function environmentRow(label, tool = {}, platform = "") {
  const ok = tool.installed && (!platform || tool.authenticated);
  const authLabels = {
    missing: "缺少令牌",
    invalid: "凭据失效",
    unauthenticated: "认证失败",
  };
  const detail = !tool.installed
    ? "缺失"
    : platform && !tool.authenticated
      ? authLabels[tool.authState] || "未登录"
      : tool.login || "已安装";
  return `<div title="${escapeHTML(tool.message || "")}"><i class="${ok ? "ok" : "bad"}"></i><span>${label}</span><b>${escapeHTML(detail)}</b></div>`;
}

export function loginButton(tool = {}, platform) {
  const label =
    tool.authState === "missing" || tool.authState === "invalid"
      ? `重新登录 ${platformLabel(platform)}`
      : `登录 ${platformLabel(platform)}`;
  return tool.installed && !tool.authenticated
    ? `<button type="button" class="login-button" data-login="${platform}">${label}</button>`
    : "";
}

export function targetFields(platform, env) {
  const tool = platform === "github" ? env.github : env.gitlab;
  const managed =
    platform !== "generic" && Boolean(tool?.installed && tool?.authenticated);
  if (managed) {
    return {
      managed,
      html: `<div class="two-col"><label>目标命名空间 <span>用户、组织或群组</span><input name="targetNamespace" value="${escapeHTML(tool.login || "")}" required placeholder="team/platform"></label><label>自定义目标名称 <span>留空则使用源仓库名</span><input name="targetName" placeholder="my-repository-mirror"></label></div><div class="two-col"><label>目标可见性<select name="visibility"><option value="private">Private</option><option value="internal">Internal</option><option value="public">Public</option></select></label><label>同名仓库处理<select name="conflictPolicy"><option value="error">询问并停止</option><option value="associate">关联已有仓库</option><option value="rename">自动添加数字后缀</option></select></label></div>`,
    };
  }
  const cli = platform === "github" ? "gh" : "glab";
  const message =
    platform === "generic"
      ? "请先在目标平台创建仓库，然后填写完整推送 URL。"
      : `${tool?.message || `${cli} 缺失或未登录`}；当前只能关联已存在仓库的完整推送 URL。`;
  const login =
    platform !== "generic" && tool?.installed
      ? `<button type="button" class="inline-login" data-target-login="${platform}">重新登录</button>`
      : "";
  return {
    managed,
    html: `<div class="target-hint warning"><span>${escapeHTML(message)}</span>${login}</div><label>目标仓库 URL <span>支持 HTTPS、SSH 或 SCP 地址</span><input name="targetUrl" required placeholder="git@code.example:team/project.git" autocomplete="off"></label>`,
  };
}
