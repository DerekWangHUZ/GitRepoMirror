// App 外壳：迁移自 main.js + actions.js + renderer.js。
// - Wails 事件订阅（log/progress/repositories-changed）集中在 useEffect 中管理生命周期，
//   返回卸载清理函数，严禁轮询。
// - backend.js 仍是唯一 API 层（DEV 下 ?fixture=states mock 继续可用）。
// - 外壳链路对齐模板规范 S1：.window → .main → .page → .page-inner → .toolbar；
//   .page 是唯一滚动容器（原生滚动条隐藏，自绘 mac-scrollbar 悬浮）。
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { Activity, Moon, Plus, RefreshCw, Sun } from "lucide-react";

import { ConfirmDialog } from "@/components/ConfirmDialog";
import { Console } from "@/components/Console";
import { CreateMirrorDialog } from "@/components/CreateMirrorDialog";
import { LogsView } from "@/components/LogsView";
import { MacScrollbar } from "@/components/MacScrollbar";
import { RepositoriesView } from "@/components/RepositoriesView";
import { SettingsView } from "@/components/SettingsView";
import { Sidebar } from "@/components/Sidebar";
import {
  Command,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command";
import { Toaster } from "@/components/ui/sonner";
import { backend, EventsOn } from "@/backend";
import { NAV_ENTRIES } from "@/lib/nav";
import { useTheme } from "@/lib/theme";
import { appendLog, createLogs } from "@/lib/logs";
import { cn, errorText, managedPlatforms, platformLabel } from "@/lib/utils";

const EMPTY_DATA = { repositories: [], settings: { concurrency: 2 } };

export default function App() {
  const [data, setData] = useState(EMPTY_DATA);
  const [env, setEnv] = useState({});
  const [logs, setLogs] = useState(createLogs);
  const [progress, setProgress] = useState({});
  const [busy, setBusy] = useState(() => new Set());
  const [view, setView] = useState("repositories");
  const [modalOpen, setModalOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [consoleCollapsed, setConsoleCollapsed] = useState(true);
  const [confirmRequest, setConfirmRequest] = useState(null);
  const [settingsDraft, setSettingsDraft] = useState(EMPTY_DATA.settings);
  const [maximized, setMaximized] = useState(
    () =>
      typeof document !== "undefined" &&
      document.documentElement.classList.contains("is-maximized"),
  );
  const [cmdOpen, setCmdOpen] = useState(false);
  const pageRef = useRef(null);
  const { theme, setTheme } = useTheme();

  // 最大化状态由桌面接线（skin/desktop.js）驱动：is-maximized 同步时广播
  // grm:window-maximized 事件。初值经 useState 惰性初始化读取（避免 effect
  // 内同步 setState），事件驱动，零轮询。
  useEffect(() => {
    const sync = (event) => setMaximized(Boolean(event?.detail));
    window.addEventListener("grm:window-maximized", sync);
    return () => window.removeEventListener("grm:window-maximized", sync);
  }, []);

  // ⌘K / Ctrl+K 命令面板快捷键
  useEffect(() => {
    const onKeyDown = (event) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setCmdOpen((open) => !open);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  // 切视图时把 .page 滚动位置带回顶部
  useEffect(() => {
    pageRef.current?.scrollTo({ top: 0 });
  }, [view]);

  const applyLogs = useCallback((updater) => {
    setLogs((previous) => {
      const next = [...previous];
      updater(next);
      return next;
    });
  }, []);

  const refreshData = useCallback(async () => {
    try {
      const next = await backend.GetData();
      setData(next);
      setSettingsDraft((draft) => {
        // 用户正在编辑的表单不被后台刷新覆盖（沿用旧 renderer.settings 的保护）。
        if (document.activeElement?.closest?.("#settingsForm")) return draft;
        return next.settings || EMPTY_DATA.settings;
      });
    } catch (error) {
      toast.error(errorText(error));
    }
  }, []);

  const checkEnvironment = useCallback(async () => {
    try {
      setEnv(await backend.CheckEnvironment());
    } catch (error) {
      toast.error(errorText(error));
    }
  }, []);

  // Wails 事件订阅：挂载时注册、卸载时反注册（EventsOn 返回 off 函数）。
  useEffect(() => {
    const offLog = EventsOn("log", (log) => {
      applyLogs((next) => appendLog(next, log));
    });
    const offProgress = EventsOn("progress", (event) => {
      setProgress((previous) => ({
        ...previous,
        [event.repositoryId || "creating"]: event,
      }));
    });
    const offChanged = EventsOn("repositories-changed", () => {
      refreshData();
    });
    return () => {
      offLog?.();
      offProgress?.();
      offChanged?.();
    };
  }, [applyLogs, refreshData]);

  // 首次加载：数据 + 环境（沿用旧 main.js 的 Promise.all）。
  // setState 均发生在 await 之后（异步微任务），非 effect 同步调用。
  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      if (cancelled) return;
      await Promise.all([refreshData(), checkEnvironment()]);
    };
    load();
    return () => {
      cancelled = true;
    };
  }, [refreshData, checkEnvironment]);

  const busyCount = busy.size;

  const addBusy = useCallback((id) => {
    setBusy((previous) => {
      const next = new Set(previous);
      next.add(id);
      return next;
    });
  }, []);

  const deleteBusy = useCallback((ids) => {
    setBusy((previous) => {
      const next = new Set(previous);
      ids.forEach((id) => next.delete(id));
      return next;
    });
  }, []);

  // ---- 窗口控制（红绿灯）：maximized 接真实状态（桌面接线同步） ----
  const windowActions = useMemo(
    () => ({
      maximized,
      onClose: () => backend.Close(),
      onMinimize: () => backend.Minimise(),
      onMaximize: () => backend.ToggleMaximise(),
    }),
    [maximized],
  );

  // ---- 业务动作 ----
  const openModal = useCallback(() => {
    if (!env.git?.installed) {
      toast.error("请先安装 Git");
      return;
    }
    setModalOpen(true);
  }, [env]);

  const createMirror = useCallback(
    async (request) => {
      setSubmitting(true);
      setConsoleCollapsed(false);
      addBusy("creating");
      try {
        await backend.CreateMirror(request);
        toast.success("仓库镜像创建成功");
        setModalOpen(false);
        await refreshData();
      } catch (error) {
        toast.error(errorText(error));
        await refreshData();
      } finally {
        deleteBusy(["creating"]);
        setSubmitting(false);
      }
    },
    [addBusy, deleteBusy, refreshData],
  );

  const syncOne = useCallback(
    async (id) => {
      if (busy.has(id) || busy.has("creating")) return;
      addBusy(id);
      setConsoleCollapsed(false);
      try {
        await backend.SyncRepository(id);
        toast.success("同步完成");
      } catch (error) {
        toast.error(errorText(error));
      } finally {
        deleteBusy([id]);
        await refreshData();
      }
    },
    [addBusy, busy, deleteBusy, refreshData],
  );

  const syncAll = useCallback(async () => {
    if (busy.size) return;
    const confirmed = await new Promise((resolve) => {
      setConfirmRequest({
        title: "全部同步",
        message: `确认同步全部 ${data.repositories.length} 个仓库？`,
        confirmText: "全部同步",
        resolve,
      });
    });
    setConfirmRequest(null);
    if (!confirmed) return;
    const ids = data.repositories.map((repo) => repo.id);
    ids.forEach(addBusy);
    setConsoleCollapsed(false);
    try {
      await backend.SyncAll();
      toast.success("全部仓库同步完成");
    } catch (error) {
      toast.error(`部分仓库同步失败：${errorText(error)}`);
    } finally {
      deleteBusy(ids);
      await refreshData();
    }
  }, [addBusy, busy.size, data.repositories, deleteBusy, refreshData]);

  const removeOne = useCallback(
    async (id) => {
      const repo = data.repositories.find((item) => item.id === id);
      if (!repo) return;
      // 解除绑定只移除本地记录；远端仓库永远不会被删除。
      const confirmed = await new Promise((resolve) => {
        setConfirmRequest({
          title: "解除本地绑定",
          message: `将解除 ${repo.source.displayName} → ${repo.target.displayName} 的本地绑定。\n远端仓库不会被删除。`,
          confirmText: "解除绑定",
          danger: true,
          resolve,
        });
      });
      setConfirmRequest(null);
      if (!confirmed) return;
      try {
        await backend.RemoveRepository(id);
        toast.success("已解除本地绑定");
        await refreshData();
      } catch (error) {
        toast.error(errorText(error));
      }
    },
    [data.repositories, refreshData],
  );

  const saveSettings = useCallback(async () => {
    const settings = {
      concurrency: Number(settingsDraft.concurrency ?? 2),
      syncLfs: Boolean(settingsDraft.syncLfs),
      pruneTargetRefs: Boolean(settingsDraft.pruneTargetRefs),
      proxy: settingsDraft.proxy || "",
      prefix: settingsDraft.prefix || "",
      suffix: settingsDraft.suffix || "",
    };
    try {
      await backend.SaveSettings(settings);
      toast.success("设置已保存");
      await refreshData();
    } catch (error) {
      toast.error(errorText(error));
    }
  }, [refreshData, settingsDraft]);

  const clearLogs = useCallback(() => {
    setLogs(createLogs());
  }, []);

  const loginPlatform = useCallback(
    async (platform) => {
      toast.info(`请在新终端中完成 ${platformLabel(platform)} 登录`);
      try {
        await backend.StartLogin(platform);
        await checkEnvironment();
      } catch (error) {
        toast.error(errorText(error));
      }
    },
    [checkEnvironment],
  );

  // 派生：环境摘要
  const envOk = Boolean(env.git?.installed);
  const connectedCount = managedPlatforms().filter(
    (platform) => env[platform.id]?.authenticated,
  ).length;

  const jumpToView = useCallback((id) => {
    setCmdOpen(false);
    setView(id);
  }, []);

  return (
    <div className={cn("window", maximized && "is-maximized")}>
      <main className="main">
        {/* .page：唯一滚动容器；原生滚动条隐藏，自绘滚动条悬浮在右侧 */}
        <div className="page" ref={pageRef}>
          <div className="page-inner">
            <div className={cn("grm-view", view !== "repositories" && "hidden")}>
              <RepositoriesView
                repositories={data.repositories || []}
                busy={busy}
                onSyncAll={syncAll}
                onNewMirror={openModal}
                onSyncOne={syncOne}
                onRemove={removeOne}
                onOpenURL={(url) => backend.OpenURL(url)}
              />
            </div>
            <div className={cn("grm-view", view !== "logs" && "hidden")}>
              <LogsView logs={logs} onClear={clearLogs} />
            </div>
            <div className={cn("grm-view", view !== "settings" && "hidden")}>
              <SettingsView
                draft={settingsDraft}
                onDraftChange={setSettingsDraft}
                onSave={saveSettings}
              />
            </div>
          </div>
        </div>
        <Console
          logs={logs}
          busyCount={busyCount}
          collapsed={consoleCollapsed}
          onToggle={() => setConsoleCollapsed((value) => !value)}
        />
      </main>

      {/* 真悬浮玻璃侧栏：position: fixed 位于 .main 之上 */}
      <Sidebar
        view={view}
        onViewChange={setView}
        onOpenCommand={() => setCmdOpen(true)}
        repoCount={(data.repositories || []).length}
        env={env}
        envOk={envOk}
        connectedCount={connectedCount}
        onRefreshEnv={checkEnvironment}
        onLoginPlatform={loginPlatform}
        windowActions={windowActions}
      />

      <MacScrollbar targetRef={pageRef} route={view} />

      <CreateMirrorDialog
        key={String(modalOpen)}
        open={modalOpen}
        onOpenChange={setModalOpen}
        env={env}
        defaults={data.settings}
        progress={progress.creating}
        submitting={submitting}
        onSubmit={createMirror}
        onLoginPlatform={loginPlatform}
      />
      <ConfirmDialog
        request={confirmRequest}
        onResolve={() => setConfirmRequest(null)}
      />

      {/* 全局 ⌘K 命令面板：视图跳转 + 快捷操作 + 外观切换，不发明新动作 */}
      <CommandDialog
        open={cmdOpen}
        onOpenChange={setCmdOpen}
        title="GitRepoMirror 命令面板"
        description="快速切换视图、执行同步操作或调整外观。"
      >
        <Command>
          <CommandInput placeholder="输入视图、命令或外观操作…" />
          <CommandList>
            <CommandEmpty>未找到匹配的视图或命令。</CommandEmpty>
            <CommandGroup heading="视图">
              {NAV_ENTRIES.map((entry) => {
                const Icon = entry.icon;
                return (
                  <CommandItem
                    key={entry.id}
                    value={`${entry.label} ${entry.id}`}
                    onSelect={() => jumpToView(entry.id)}
                  >
                    <Icon className="size-4" />
                    <span>{entry.label}</span>
                  </CommandItem>
                );
              })}
            </CommandGroup>
            <CommandSeparator />
            <CommandGroup heading="快捷操作">
              <CommandItem
                value="新建镜像 create mirror"
                onSelect={() => {
                  setCmdOpen(false);
                  openModal();
                }}
              >
                <Plus className="size-4" />
                <span>新建镜像</span>
              </CommandItem>
              <CommandItem
                value="全部同步 sync all"
                onSelect={() => {
                  setCmdOpen(false);
                  syncAll();
                }}
              >
                <RefreshCw className="size-4" />
                <span>全部同步</span>
              </CommandItem>
              <CommandItem
                value="检查环境 environment check"
                onSelect={() => {
                  setCmdOpen(false);
                  checkEnvironment().then(() => toast.success("环境状态已刷新"));
                }}
              >
                <Activity className="size-4" />
                <span>检查环境</span>
              </CommandItem>
            </CommandGroup>
            <CommandSeparator />
            <CommandGroup heading="外观">
              <CommandItem
                value="切换深浅色主题 theme dark light"
                onSelect={() => {
                  const next = theme === "dark" ? "light" : "dark";
                  setTheme(next);
                  setCmdOpen(false);
                  toast.success(`已切换为${next === "dark" ? "深色" : "浅色"}外观`);
                }}
              >
                {theme === "dark" ? (
                  <Sun className="size-4" />
                ) : (
                  <Moon className="size-4" />
                )}
                <span>切换至{theme === "dark" ? "浅色" : "深色"}模式</span>
              </CommandItem>
            </CommandGroup>
          </CommandList>
        </Command>
      </CommandDialog>

      <Toaster position="top-right" />
    </div>
  );
}
