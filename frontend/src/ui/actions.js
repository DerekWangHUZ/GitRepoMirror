import { $, errorText, platformLabel } from "./utils.js";

export function createActions(state, backend, renderer) {
  function toast(message, type = "info") {
    const element = document.createElement("div");
    element.className = `toast ${type}`;
    element.textContent = message;
    $("#toasts").appendChild(element);
    setTimeout(() => {
      element.classList.add("leaving");
      setTimeout(() => element.remove(), 250);
    }, 4200);
  }

  async function refreshData() {
    try {
      state.data = await backend.GetData();
      renderer.data();
    } catch (error) {
      toast(errorText(error), "error");
    }
  }

  async function checkEnvironment() {
    try {
      state.env = await backend.CheckEnvironment();
      renderer.environment();
    } catch (error) {
      toast(errorText(error), "error");
    }
  }

  function openModal() {
    if (!state.env.git?.installed) {
      toast("请先安装 Git", "error");
      return;
    }
    const form = $("#mirrorForm");
    form.reset();
    form.prefix.value = state.data.settings?.prefix || "";
    form.suffix.value = state.data.settings?.suffix || "";
    form.syncLfs.checked = Boolean(state.data.settings?.syncLfs);
    renderer.target();
    $("#modalSteps")
      .querySelectorAll("span")
      .forEach((element, index) => {
        element.className = index === 0 ? "active" : "";
      });
    $("#modalBackdrop").classList.remove("hidden");
    setTimeout(() => form.source.focus(), 100);
  }

  function closeModal() {
    if (!$("#createSubmit").disabled)
      $("#modalBackdrop").classList.add("hidden");
  }

  async function loginPlatform(platform, button) {
    if (button) button.disabled = true;
    toast(`请在新终端中完成 ${platformLabel(platform)} 登录`, "info");
    try {
      await backend.StartLogin(platform);
      await checkEnvironment();
      const tool = platform === "github" ? state.env.github : state.env.gitlab;
      if (!tool?.authenticated)
        throw new Error(
          tool?.message || `${platformLabel(platform)} 登录未完成`,
        );
      toast(`${platformLabel(platform)} 已连接`, "success");
      if (!$("#modalBackdrop").classList.contains("hidden")) renderer.target();
    } catch (error) {
      toast(errorText(error), "error");
      if (button?.isConnected) button.disabled = false;
    }
  }

  async function createMirror(form) {
    const request = Object.fromEntries(new FormData(form).entries());
    request.syncLfs = form.syncLfs.checked;
    request.visibility ||= "private";
    request.conflictPolicy ||= "error";
    $("#createSubmit").disabled = true;
    $("#createSubmit").textContent = "正在备份…";
    $("#console").classList.remove("collapsed");
    state.busy.add("creating");
    renderer.consoleOutput();
    try {
      await backend.CreateMirror(request);
      toast("仓库镜像创建成功", "success");
      $("#modalBackdrop").classList.add("hidden");
      await refreshData();
    } catch (error) {
      toast(errorText(error), "error");
      await refreshData();
    } finally {
      state.busy.delete("creating");
      renderer.consoleOutput();
      $("#createSubmit").disabled = false;
      $("#createSubmit").textContent = "开始备份";
    }
  }

  async function syncOne(id) {
    state.busy.add(id);
    renderer.repositories();
    renderer.consoleOutput();
    $("#console").classList.remove("collapsed");
    try {
      await backend.SyncRepository(id);
      toast("同步完成", "success");
    } catch (error) {
      toast(errorText(error), "error");
    } finally {
      state.busy.delete(id);
      await refreshData();
      renderer.consoleOutput();
    }
  }

  async function syncAll() {
    if (!confirm(`确认同步全部 ${state.data.repositories.length} 个仓库？`))
      return;
    state.data.repositories.forEach((repo) => state.busy.add(repo.id));
    renderer.repositories();
    renderer.consoleOutput();
    $("#console").classList.remove("collapsed");
    try {
      await backend.SyncAll();
      toast("全部仓库同步完成", "success");
    } catch (error) {
      toast(`部分仓库同步失败：${errorText(error)}`, "error");
    } finally {
      state.busy.clear();
      await refreshData();
      renderer.consoleOutput();
    }
  }

  async function removeOne(id) {
    const repo = state.data.repositories.find((item) => item.id === id);
    if (!repo) return;
    let deleteRemote = false;
    if (repo.managementMode === "cli")
      deleteRemote = confirm(
        `是否同时删除远端仓库 ${repo.target.displayName}？\n\n选择“确定”会删除远端仓库；选择“取消”只解除本地绑定。`,
      );
    if (!deleteRemote && !confirm("确认只解除本地绑定并保留远端仓库？")) return;
    try {
      await backend.RemoveRepository(id, deleteRemote);
      toast(
        deleteRemote ? "远端仓库和本地记录已删除" : "已解除本地绑定",
        "success",
      );
      await refreshData();
    } catch (error) {
      toast(errorText(error), "error");
    }
  }

  async function saveSettings(form) {
    const settings = Object.fromEntries(new FormData(form).entries());
    settings.concurrency = Number(settings.concurrency);
    settings.syncLfs = form.syncLfs.checked;
    try {
      await backend.SaveSettings(settings);
      toast("设置已保存", "success");
      await refreshData();
    } catch (error) {
      toast(errorText(error), "error");
    }
  }

  function clearLogs() {
    state.logs = [];
    renderer.logs();
    renderer.consoleOutput();
  }

  function bind() {
    document.addEventListener("click", (event) => {
      const target = event.target.closest("button");
      if (!target) return;
      if (target.id === "closeBtn") backend.Close();
      else if (target.id === "minBtn") backend.Minimise();
      else if (target.id === "maxBtn") backend.ToggleMaximise();
      else if (target.id === "modalClose" || target.id === "modalCancel")
        closeModal();
      else if (target.id === "consoleToggle")
        $("#console").classList.toggle("collapsed");
      else if (target.dataset.view) {
        state.view = target.dataset.view;
        renderer.showView();
      } else if (target.dataset.action === "new-mirror") openModal();
      else if (target.dataset.action === "sync-all") syncAll();
      else if (target.dataset.action === "clear-logs") clearLogs();
      else if (target.dataset.action === "refresh-env") checkEnvironment();
      else if (target.dataset.sync) syncOne(target.dataset.sync);
      else if (target.dataset.remove) removeOne(target.dataset.remove);
      else if (target.dataset.open) backend.OpenURL(target.dataset.open);
      else if (target.dataset.login)
        loginPlatform(target.dataset.login, target);
      else if (target.dataset.targetLogin)
        loginPlatform(target.dataset.targetLogin, target);
    });
    document.addEventListener("submit", (event) => {
      event.preventDefault();
      if (event.target.id === "mirrorForm") createMirror(event.target);
      if (event.target.id === "settingsForm") saveSettings(event.target);
    });
    $("#mirrorForm").targetPlatform.addEventListener("change", renderer.target);
    $("#modalBackdrop").addEventListener("click", (event) => {
      if (event.target === event.currentTarget) closeModal();
    });
  }

  return { bind, refreshData, checkEnvironment, toast };
}
