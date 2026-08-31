import './style.css'
import * as App from '../wailsjs/go/main/App.js'
import { EventsOn as RuntimeEventsOn } from '../wailsjs/runtime/runtime.js'

const icons = {
  git: '<svg viewBox="0 0 24 24"><path d="m12 2 10 10-10 10L2 12z"/><circle cx="8.5" cy="8.5" r="1.5"/><circle cx="15.5" cy="15.5" r="1.5"/><path d="m9.6 9.6 4.8 4.8M8.5 10v5"/></svg>',
  github: '<svg viewBox="0 0 24 24"><path d="M12 .7a11.5 11.5 0 0 0-3.64 22.4c.58.1.79-.25.79-.56v-2.23c-3.22.7-3.9-1.37-3.9-1.37-.53-1.34-1.29-1.7-1.29-1.7-1.05-.72.08-.7.08-.7 1.16.08 1.78 1.2 1.78 1.2 1.04 1.77 2.72 1.26 3.38.96.1-.75.4-1.26.74-1.55-2.57-.29-5.27-1.28-5.27-5.68 0-1.26.45-2.28 1.19-3.08-.12-.29-.52-1.46.11-3.04 0 0 .97-.31 3.16 1.18a10.97 10.97 0 0 1 5.76 0c2.2-1.49 3.16-1.18 3.16-1.18.63 1.58.23 2.75.11 3.04.74.8 1.19 1.82 1.19 3.08 0 4.41-2.7 5.38-5.28 5.67.42.36.79 1.07.79 2.16v3.2c0 .31.21.67.8.56A11.5 11.5 0 0 0 12 .7Z"/></svg>',
  gitlab: '<svg viewBox="0 0 24 24"><path d="m12 21 8.5-6.2L18.2 6h-3.1L12 15 8.9 6H5.8l-2.3 8.8z"/></svg>',
  generic: '<svg viewBox="0 0 24 24"><circle cx="6" cy="6" r="2"/><circle cx="18" cy="6" r="2"/><circle cx="12" cy="18" r="2"/><path d="M7.7 7.1 11 16M16.3 7.1 13 16M8 6h8"/></svg>',
  sync: '<svg viewBox="0 0 24 24"><path d="M20 7v5h-5M4 17v-5h5"/><path d="M6.1 8.5A7 7 0 0 1 18.8 7L20 9M4 15l1.2 2A7 7 0 0 0 18 15.5"/></svg>',
  plus: '<svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>',
  settings: '<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06-2.83 2.83-.06-.06a1.7 1.7 0 0 0-1.88-.34 1.7 1.7 0 0 0-1.03 1.56V21h-4v-.08A1.7 1.7 0 0 0 9 19.37a1.7 1.7 0 0 0-1.88.34l-.06.06-2.83-2.83.06-.06A1.7 1.7 0 0 0 4.63 15 1.7 1.7 0 0 0 3.08 14H3v-4h.08A1.7 1.7 0 0 0 4.63 9a1.7 1.7 0 0 0-.34-1.88l-.06-.06 2.83-2.83.06.06A1.7 1.7 0 0 0 9 4.63 1.7 1.7 0 0 0 10 3.08V3h4v.08A1.7 1.7 0 0 0 15 4.63a1.7 1.7 0 0 0 1.88-.34l.06-.06 2.83 2.83-.06.06A1.7 1.7 0 0 0 19.37 9 1.7 1.7 0 0 0 20.92 10H21v4h-.08A1.7 1.7 0 0 0 19.4 15Z"/></svg>',
  terminal: '<svg viewBox="0 0 24 24"><path d="m6 8 4 4-4 4M12 16h6"/></svg>',
  trash: '<svg viewBox="0 0 24 24"><path d="M4 7h16M9 7V4h6v3M7 7l1 13h8l1-13M10 11v5M14 11v5"/></svg>',
  external: '<svg viewBox="0 0 24 24"><path d="M14 5h5v5M19 5l-8 8"/><path d="M17 13v6H5V7h6"/></svg>'
}

const preview = {
  GetData: async () => ({ repositories: [], settings: { concurrency: 2, syncLfs: false, proxy: '', prefix: '', suffix: '' } }),
  CheckEnvironment: async () => ({ git: { installed: true }, gitLfs: { installed: false }, github: { installed: true, authenticated: true, login: 'workspace' }, gitlab: { installed: false, authenticated: false } }),
  StartLogin: async () => {}, CreateMirror: async () => {}, SyncRepository: async () => {}, SyncAll: async () => {}, RemoveRepository: async () => {}, SaveSettings: async () => {}, OpenURL: async () => {}, Minimise: async () => {}, ToggleMaximise: async () => {}, Close: async () => {}
}

const backend = import.meta.env.DEV && !window.go ? preview : App
const EventsOn = window.runtime ? RuntimeEventsOn : () => () => {}
const app = document.querySelector('#app')
let state = { data: { repositories: [], settings: { concurrency: 2 } }, env: {}, logs: [], progress: {}, busy: new Set(), view: 'repositories' }

app.innerHTML = `
  <main class="window-shell">
    <aside class="sidebar">
      <div class="sidebar-chrome" data-wails-drag>
        <div class="traffic" data-wails-no-drag>
          <button class="light close" id="closeBtn" aria-label="关闭"></button>
          <button class="light min" id="minBtn" aria-label="最小化"></button>
          <button class="light max" id="maxBtn" aria-label="最大化"></button>
        </div>
      </div>
      <div class="sidebar-navigation">
        <div class="sidebar-label">工作台</div>
        <button class="nav active" data-view="repositories">${icons.git}<span>镜像仓库</span><b id="repoCount">0</b></button>
        <button class="nav" data-view="logs">${icons.terminal}<span>运行日志</span></button>
        <div class="sidebar-label settings-label">偏好设置</div>
        <button class="nav" data-view="settings">${icons.settings}<span>设置</span></button>
      </div>
      <div class="environment-card" id="environmentCard"></div>
    </aside>
    <div class="main-area">
      <header class="titlebar" data-wails-drag>
        <div class="title"><span class="title-mark">${icons.git}</span> GitRepoMirror</div>
        <div class="titlebar-spacer"></div>
        <div class="env-pill" id="envPill" data-wails-no-drag><i></i><span>正在检查环境</span></div>
      </header>
      <section class="content">
        <div id="repositoriesView" class="view"></div>
        <div id="logsView" class="view hidden"></div>
        <div id="settingsView" class="view hidden"></div>
      </section>
      <section class="console collapsed" id="console">
        <button class="console-head" id="consoleToggle"><span>${icons.terminal} 实时输出</span><span id="consoleStatus">空闲</span><i>⌃</i></button>
        <div class="console-body" id="consoleBody"></div>
      </section>
    </div>
  </main>
  <div class="toast-stack" id="toasts"></div>
  <div class="modal-backdrop hidden" id="modalBackdrop">
    <form class="modal" id="mirrorForm">
      <div class="modal-head"><div><h2>创建仓库镜像</h2><p>在代码托管平台之间建立独立镜像。</p></div><button type="button" class="icon-close" id="modalClose">×</button></div>
      <label>源仓库 <span>GitHub 可用 owner/repo，其他平台使用完整 URL</span><input name="source" required placeholder="https://gitlab.com/team/project.git" autocomplete="off"></label>
      <div class="two-col">
        <label>目标平台<select name="targetPlatform"><option value="github">GitHub</option><option value="gitlab">GitLab</option><option value="generic">其他 Git 平台</option></select></label>
        <label>备份模式<select name="mode"><option value="mirror">完整镜像 · 所有分支与 Tag</option><option value="shallow">浅克隆 · 默认分支最新版本</option></select></label>
      </div>
      <div id="targetFields"></div>
      <div class="two-col">
        <label>名称前缀<input name="prefix" placeholder="mirror-"></label>
        <label>名称后缀<input name="suffix" placeholder="-backup"></label>
      </div>
      <label class="check-row"><input type="checkbox" name="syncLfs"><span><strong>同步 Git LFS 对象</strong><small>需要本机已安装 Git LFS，可能显著增加耗时和流量</small></span></label>
      <div class="steps" id="modalSteps"><span class="active">1 准备目标库</span><span>2 拉取源代码</span><span>3 推送镜像</span><span>4 清理完成</span></div>
      <div class="modal-actions"><button type="button" class="button secondary" id="modalCancel">取消</button><button type="submit" class="button primary" id="createSubmit">开始备份</button></div>
    </form>
  </div>`

const $ = selector => document.querySelector(selector)
const escapeHTML = (value = '') => String(value).replace(/[&<>'"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' })[c])
const formatTime = value => value && !value.startsWith('0001-') ? new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : '尚未同步'
const platformLabel = platform => ({ github: 'GitHub', gitlab: 'GitLab', generic: '其他 Git' })[platform] || 'Git'
const platformIcon = platform => icons[platform] || icons.generic
const errorText = error => String(error).replace(/^Error:\s*/, '')

function render() {
  renderRepositories()
  renderLogs()
  renderSettings()
  renderEnvironment()
  document.querySelectorAll('.view').forEach(el => el.classList.add('hidden'))
  $(`#${state.view}View`).classList.remove('hidden')
  document.querySelectorAll('.nav').forEach(el => el.classList.toggle('active', el.dataset.view === state.view))
}

function renderRepositories() {
  const repos = state.data.repositories || []
  $('#repoCount').textContent = repos.length
  $('#repositoriesView').innerHTML = `<div class="page-head"><div><h1>镜像仓库</h1><p>在不同 Git 托管平台之间保存独立副本</p></div><div class="head-actions"><button class="button secondary" id="syncAll" ${!repos.length ? 'disabled' : ''}>${icons.sync} 全部同步</button><button class="button primary" id="newMirror">${icons.plus} 新建镜像</button></div></div>${repos.length ? `<div class="repo-grid">${repos.map(repoCard).join('')}</div>` : emptyState()}`
  $('#newMirror').onclick = openModal
  $('#syncAll').onclick = syncAllRepositories
  document.querySelectorAll('[data-sync]').forEach(el => el.onclick = () => syncOne(el.dataset.sync))
  document.querySelectorAll('[data-remove]').forEach(el => el.onclick = () => removeOne(el.dataset.remove))
  document.querySelectorAll('[data-open]').forEach(el => el.onclick = () => backend.OpenURL(el.dataset.open))
}

function repoCard(repo) {
  const status = repo.status || 'healthy'
  const labels = { healthy: '正常', syncing: '同步中', failed: '失败' }
  const isBusy = state.busy.has(repo.id) || status === 'syncing'
  return `<article class="repo-card ${status}"><div class="repo-top"><div class="repo-icon ${escapeHTML(repo.source.platform)}">${platformIcon(repo.source.platform)}</div><div class="repo-title"><h3>${escapeHTML(repo.source.displayName)}</h3>${repo.target.webUrl ? `<button data-open="${escapeHTML(repo.target.webUrl)}">${escapeHTML(repo.target.displayName)} ${icons.external}</button>` : `<span class="target-name">${escapeHTML(repo.target.displayName)}</span>`}</div><span class="status ${status}"><i></i>${labels[status] || status}</span></div><div class="repo-meta"><span>平台 <b>${platformLabel(repo.source.platform)} → ${platformLabel(repo.target.platform)}</b></span><span>模式 <b>${repo.mode === 'shallow' ? '浅克隆' : '完整镜像'}</b></span><span>上次同步 <b>${formatTime(repo.lastSync)}</b></span></div>${repo.lastError ? `<div class="error-line" title="${escapeHTML(repo.lastError)}">${escapeHTML(repo.lastError)}</div>` : ''}<div class="repo-actions"><button class="button sync-button" data-sync="${repo.id}" ${isBusy ? 'disabled' : ''}>${icons.sync} ${isBusy ? '同步中…' : '立即同步'}</button><button class="icon-button danger" data-remove="${repo.id}" title="解除绑定">${icons.trash}</button></div></article>`
}

function emptyState() {
  return `<div class="empty"><div class="empty-visual"><div>${icons.git}</div><span>${icons.sync}</span></div><h2>还没有镜像仓库</h2><p>创建第一个仓库镜像，保存分支、Tag 和提交历史。</p><button class="button primary" onclick="document.querySelector('#newMirror').click()">${icons.plus} 创建第一个镜像</button></div>`
}

function renderLogs() {
  const content = state.logs.length ? state.logs.map(logLine).join('') : '<div class="log-empty">命令输出将在这里显示</div>'
  $('#logsView').innerHTML = `<div class="page-head"><div><h1>运行日志</h1><p>实时查看 Git 与平台 CLI 输出</p></div><button class="button secondary" id="clearLogs">清空日志</button></div><div class="log-panel">${content}</div>`
  $('#clearLogs').onclick = () => { state.logs = []; renderLogs(); renderConsole() }
}

function logLine(log) { return `<div class="log-line ${escapeHTML(log.level)}"><time>${escapeHTML(log.time)}</time><span>${escapeHTML(log.message)}</span></div>` }

function renderSettings() {
  const s = state.data.settings || {}
  $('#settingsView').innerHTML = `<div class="page-head"><div><h1>设置</h1><p>配置网络、命名与同步行为</p></div></div><form class="settings-form" id="settingsForm"><section class="setting-section"><h3>网络</h3><label>独立代理地址<span>留空使用系统网络。支持 HTTP、HTTPS 和 SOCKS5。</span><input name="proxy" value="${escapeHTML(s.proxy || '')}" placeholder="http://127.0.0.1:7890"></label></section><section class="setting-section"><h3>默认命名</h3><div class="two-col"><label>名称前缀<input name="prefix" value="${escapeHTML(s.prefix || '')}" placeholder="mirror-"></label><label>名称后缀<input name="suffix" value="${escapeHTML(s.suffix || '')}" placeholder="-backup"></label></div></section><section class="setting-section"><h3>同步</h3><label>批量同步并发数<span>建议保持 1–2，避免触发托管平台频率限制。</span><select name="concurrency">${[1, 2, 3, 4].map(n => `<option ${n === s.concurrency ? 'selected' : ''}>${n}</option>`).join('')}</select></label><label class="check-row"><input type="checkbox" name="syncLfs" ${s.syncLfs ? 'checked' : ''}><span><strong>默认同步 Git LFS 对象</strong><small>仅在已安装 Git LFS 时生效</small></span></label></section><button class="button primary" type="submit">保存设置</button></form>`
  $('#settingsForm').onsubmit = saveSettings
}

function renderEnvironment() {
  const env = state.env
  const gitOK = Boolean(env.git?.installed)
  const connected = [env.github, env.gitlab].filter(tool => tool?.authenticated).length
  $('#envPill').className = `env-pill ${gitOK ? 'ok' : 'bad'}`
  $('#envPill').innerHTML = `<i></i><span>${gitOK ? `Git 可用 · ${connected} 个平台已连接` : '环境需要处理'}</span>`
  $('#environmentCard').innerHTML = `<div class="env-title"><span>环境状态</span><button id="refreshEnv">↻</button></div>${environmentRow('Git', env.git)}${environmentRow('GitHub CLI', env.github, 'github')}${environmentRow('GitLab CLI', env.gitlab, 'gitlab')}<div class="login-actions">${loginButton(env.github, 'github')}${loginButton(env.gitlab, 'gitlab')}</div>`
  $('#refreshEnv').onclick = checkEnvironment
  document.querySelectorAll('[data-login]').forEach(button => button.onclick = () => loginPlatform(button.dataset.login, button))
}

function environmentRow(label, tool = {}, platform = '') {
  const ok = tool.installed && (!platform || tool.authenticated)
  const authLabels = { missing: '缺少令牌', invalid: '凭据失效', unauthenticated: '认证失败' }
  const detail = !tool.installed ? '缺失' : platform && !tool.authenticated ? authLabels[tool.authState] || '未登录' : tool.login || '已安装'
  return `<div title="${escapeHTML(tool.message || '')}"><i class="${ok ? 'ok' : 'bad'}"></i><span>${label}</span><b>${escapeHTML(detail)}</b></div>`
}

function loginButton(tool = {}, platform) {
  const label = tool.authState === 'missing' || tool.authState === 'invalid' ? `重新登录 ${platformLabel(platform)}` : `登录 ${platformLabel(platform)}`
  return tool.installed && !tool.authenticated ? `<button type="button" class="login-button" data-login="${platform}">${label}</button>` : ''
}

async function loginPlatform(platform, button) {
  if (button) button.disabled = true
  toast(`请在新终端中完成 ${platformLabel(platform)} 登录`, 'info')
  try {
    await backend.StartLogin(platform)
    await checkEnvironment()
    const tool = platform === 'github' ? state.env.github : state.env.gitlab
    if (!tool?.authenticated) throw new Error(tool?.message || `${platformLabel(platform)} 登录未完成`)
    toast(`${platformLabel(platform)} 已连接`, 'success')
    if (!$('#modalBackdrop').classList.contains('hidden')) renderTargetFields()
  } catch (error) {
    toast(errorText(error), 'error')
    if (button?.isConnected) button.disabled = false
  }
}
function platformReady(platform) { const tool = platform === 'github' ? state.env.github : state.env.gitlab; return Boolean(tool?.installed && tool?.authenticated) }

function renderTargetFields() {
  const form = $('#mirrorForm')
  const platform = form.targetPlatform.value
  const managed = platform !== 'generic' && platformReady(platform)
  if (managed) {
    const tool = platform === 'github' ? state.env.github : state.env.gitlab
    $('#targetFields').innerHTML = `<div class="two-col"><label>目标命名空间 <span>用户、组织或群组</span><input name="targetNamespace" value="${escapeHTML(tool.login || '')}" required placeholder="team/platform"></label><label>自定义目标名称 <span>留空则使用源仓库名</span><input name="targetName" placeholder="my-repository-mirror"></label></div><div class="two-col"><label>目标可见性<select name="visibility"><option value="private">Private</option><option value="internal">Internal</option><option value="public">Public</option></select></label><label>同名仓库处理<select name="conflictPolicy"><option value="error">询问并停止</option><option value="associate">关联已有仓库</option><option value="rename">自动添加数字后缀</option></select></label></div>`
  } else {
    const cli = platform === 'github' ? 'gh' : 'glab'
    const tool = platform === 'github' ? state.env.github : state.env.gitlab
    const message = platform === 'generic' ? '请先在目标平台创建仓库，然后填写完整推送 URL。' : `${tool?.message || `${cli} 缺失或未登录`}；当前只能关联已存在仓库的完整推送 URL。`
    const login = platform !== 'generic' && tool?.installed ? `<button type="button" class="inline-login" data-target-login="${platform}">重新登录</button>` : ''
    $('#targetFields').innerHTML = `<div class="target-hint warning"><span>${escapeHTML(message)}</span>${login}</div><label>目标仓库 URL <span>支持 HTTPS、SSH 或 SCP 地址</span><input name="targetUrl" required placeholder="git@code.example:team/project.git" autocomplete="off"></label>`
    document.querySelector('[data-target-login]')?.addEventListener('click', event => loginPlatform(event.currentTarget.dataset.targetLogin, event.currentTarget))
  }
  form.prefix.disabled = !managed
  form.suffix.disabled = !managed
}

function renderConsole() {
  const logs = state.logs.slice(-200)
  $('#consoleBody').innerHTML = logs.length ? logs.map(logLine).join('') : '<div class="log-empty">等待任务开始…</div>'
  $('#consoleStatus').textContent = state.busy.size ? `${state.busy.size} 个任务运行中` : '空闲'
  $('#consoleBody').scrollTop = $('#consoleBody').scrollHeight
}

async function refreshData() { try { state.data = await backend.GetData(); render() } catch (error) { toast(errorText(error), 'error') } }
async function checkEnvironment() { try { state.env = await backend.CheckEnvironment(); renderEnvironment() } catch (error) { toast(errorText(error), 'error') } }

function openModal() {
  if (!state.env.git?.installed) { toast('请先安装 Git', 'error'); return }
  $('#mirrorForm').reset()
  $('#mirrorForm').prefix.value = state.data.settings?.prefix || ''
  $('#mirrorForm').suffix.value = state.data.settings?.suffix || ''
  $('#mirrorForm').syncLfs.checked = Boolean(state.data.settings?.syncLfs)
  renderTargetFields()
  $('#modalSteps').querySelectorAll('span').forEach((el, i) => el.className = i === 0 ? 'active' : '')
  $('#modalBackdrop').classList.remove('hidden')
  setTimeout(() => $('#mirrorForm').source.focus(), 100)
}

function closeModal() { if (!$('#createSubmit').disabled) $('#modalBackdrop').classList.add('hidden') }

async function createMirror(event) {
  event.preventDefault()
  const req = Object.fromEntries(new FormData(event.currentTarget).entries())
  req.syncLfs = event.currentTarget.syncLfs.checked
  req.visibility ||= 'private'; req.conflictPolicy ||= 'error'
  $('#createSubmit').disabled = true; $('#createSubmit').textContent = '正在备份…'; $('#console').classList.remove('collapsed'); state.busy.add('creating'); renderConsole()
  try { await backend.CreateMirror(req); toast('仓库镜像创建成功', 'success'); $('#modalBackdrop').classList.add('hidden'); await refreshData() }
  catch (error) { toast(errorText(error), 'error'); await refreshData() }
  finally { state.busy.delete('creating'); renderConsole(); $('#createSubmit').disabled = false; $('#createSubmit').textContent = '开始备份' }
}

async function syncOne(id) {
  state.busy.add(id); renderRepositories(); renderConsole(); $('#console').classList.remove('collapsed')
  try { await backend.SyncRepository(id); toast('同步完成', 'success') } catch (error) { toast(errorText(error), 'error') }
  finally { state.busy.delete(id); await refreshData(); renderConsole() }
}

async function syncAllRepositories() {
  if (!confirm(`确认同步全部 ${state.data.repositories.length} 个仓库？`)) return
  state.data.repositories.forEach(repo => state.busy.add(repo.id)); renderRepositories(); renderConsole(); $('#console').classList.remove('collapsed')
  try { await backend.SyncAll(); toast('全部仓库同步完成', 'success') } catch (error) { toast(`部分仓库同步失败：${errorText(error)}`, 'error') }
  finally { state.busy.clear(); await refreshData(); renderConsole() }
}

async function removeOne(id) {
  const repo = state.data.repositories.find(item => item.id === id); if (!repo) return
  let deleteRemote = false
  if (repo.managementMode === 'cli') deleteRemote = confirm(`是否同时删除远端仓库 ${repo.target.displayName}？\n\n选择“确定”会删除远端仓库；选择“取消”只解除本地绑定。`)
  if (!deleteRemote && !confirm('确认只解除本地绑定并保留远端仓库？')) return
  try { await backend.RemoveRepository(id, deleteRemote); toast(deleteRemote ? '远端仓库和本地记录已删除' : '已解除本地绑定', 'success'); await refreshData() } catch (error) { toast(errorText(error), 'error') }
}

async function saveSettings(event) {
  event.preventDefault(); const form = event.currentTarget; const settings = Object.fromEntries(new FormData(form).entries()); settings.concurrency = Number(settings.concurrency); settings.syncLfs = form.syncLfs.checked
  try { await backend.SaveSettings(settings); toast('设置已保存', 'success'); await refreshData() } catch (error) { toast(errorText(error), 'error') }
}

function toast(message, type = 'info') {
  const el = document.createElement('div'); el.className = `toast ${type}`; el.textContent = message; $('#toasts').appendChild(el)
  setTimeout(() => { el.classList.add('leaving'); setTimeout(() => el.remove(), 250) }, 4200)
}

EventsOn('log', log => { state.logs.push(log); if (state.logs.length > 1000) state.logs.shift(); renderConsole(); if (state.view === 'logs') renderLogs() })
EventsOn('progress', progress => { state.progress[progress.repositoryId || 'creating'] = progress; $('#modalSteps')?.querySelectorAll('span').forEach((el, i) => { el.className = i + 1 < progress.step ? 'done' : i + 1 === progress.step ? progress.status : '' }) })
EventsOn('repositories-changed', refreshData)

$('#closeBtn').onclick = () => backend.Close()
$('#minBtn').onclick = () => backend.Minimise()
$('#maxBtn').onclick = () => backend.ToggleMaximise()
$('#modalClose').onclick = closeModal
$('#modalCancel').onclick = closeModal
$('#mirrorForm').onsubmit = createMirror
$('#mirrorForm').targetPlatform.onchange = renderTargetFields
$('#modalBackdrop').onclick = event => { if (event.target === event.currentTarget) closeModal() }
$('#consoleToggle').onclick = () => $('#console').classList.toggle('collapsed')
document.querySelectorAll('.nav').forEach(el => el.onclick = () => { state.view = el.dataset.view; render() })

await Promise.all([refreshData(), checkEnvironment()])
renderConsole()
