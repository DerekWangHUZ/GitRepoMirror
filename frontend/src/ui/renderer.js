import {
  environmentRow,
  loginButton,
  logLine,
  logsView,
  repositoryView,
  settingsView,
  targetFields,
} from "./templates.js";
import { $, escapeHTML } from "./utils.js";

export function createRenderer(state) {
  function showView() {
    document
      .querySelectorAll(".view")
      .forEach((element) => element.classList.add("hidden"));
    $(`#${state.view}View`).classList.remove("hidden");
    document
      .querySelectorAll(".nav")
      .forEach((element) =>
        element.classList.toggle("active", element.dataset.view === state.view),
      );
  }

  function repositories() {
    const items = state.data.repositories || [];
    $("#repoCount").textContent = items.length;
    $("#repositoriesView").innerHTML = repositoryView(items, state.busy);
  }

  function logs() {
    $("#logsView").innerHTML = logsView(state.logs);
  }
  function settings() {
    $("#settingsView").innerHTML = settingsView(state.data.settings);
  }

  function environment() {
    const env = state.env;
    const gitOK = Boolean(env.git?.installed);
    const connected = [env.github, env.gitlab].filter(
      (tool) => tool?.authenticated,
    ).length;
    $("#envPill").className = `env-pill ${gitOK ? "ok" : "bad"}`;
    $("#envPill").innerHTML =
      `<i></i><span>${gitOK ? `Git 可用 · ${connected} 个平台已连接` : "环境需要处理"}</span>`;
    $("#environmentCard").innerHTML =
      `<div class="env-title"><span>环境状态</span><button id="refreshEnv" data-action="refresh-env">↻</button></div>${environmentRow("Git", env.git)}${environmentRow("GitHub CLI", env.github, "github")}${environmentRow("GitLab CLI", env.gitlab, "gitlab")}<div class="login-actions">${loginButton(env.github, "github")}${loginButton(env.gitlab, "gitlab")}</div>`;
  }

  function consoleOutput() {
    const items = state.logs.slice(-200);
    $("#consoleBody").innerHTML = items.length
      ? items.map(logLine).join("")
      : '<div class="log-empty">等待任务开始…</div>';
    $("#consoleStatus").textContent = state.busy.size
      ? `${state.busy.size} 个任务运行中`
      : "空闲";
    $("#consoleBody").scrollTop = $("#consoleBody").scrollHeight;
  }

  function target() {
    const form = $("#mirrorForm");
    const result = targetFields(form.targetPlatform.value, state.env);
    $("#targetFields").innerHTML = result.html;
    form.prefix.disabled = !result.managed;
    form.suffix.disabled = !result.managed;
  }

  function progress(event) {
    state.progress[event.repositoryId || "creating"] = event;
    $("#modalSteps")
      ?.querySelectorAll("span")
      .forEach((element, index) => {
        element.className =
          index + 1 < event.step
            ? "done"
            : index + 1 === event.step
              ? event.status
              : "";
      });
  }

  function data() {
    repositories();
    logs();
    settings();
    showView();
  }
  function all() {
    repositories();
    logs();
    settings();
    environment();
    showView();
  }

  return {
    all,
    data,
    repositories,
    logs,
    settings,
    environment,
    consoleOutput,
    target,
    progress,
    showView,
    escapeHTML,
  };
}
