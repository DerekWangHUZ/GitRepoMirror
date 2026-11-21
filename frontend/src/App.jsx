// App 外壳：迁移自 main.js + actions.js + renderer.js。
// - Wails 事件订阅（log/progress/repositories-changed）集中在 useEffect 中管理生命周期，
//   返回卸载清理函数，严禁轮询。
// - backend.js 仍是唯一 API 层（DEV 下 ?fixture=states mock 继续可用）。
import { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "sonner";

import { ConfirmDialog } from "@/components/ConfirmDialog";
import { Console } from "@/components/Console";
import { CreateMirrorDialog } from "@/components/CreateMirrorDialog";
import { LogsView } from "@/components/LogsView";
import { RepositoriesView } from "@/components/RepositoriesView";
import { SettingsView } from "@/components/SettingsView";
import { Sidebar } from "@/components/Sidebar";
import { Toaster } from "@/components/ui/sonner";
import { backend, EventsOn } from "@/backend";
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

  // ---- 窗口控制（红绿灯） ----
  const windowActions = useMemo(
    () => ({
      maximized: false,
      onClose: () => backend.Close(),
      onMinimize: () => backend.Minimise(),
      onMaximize: () => backend.ToggleMaximise(),
    }),
    [],
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

  return (
    <div className="window">
      <main className="grm-main">
        <section className="grm-content">
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
        </section>
        <Console
          logs={logs}
          busyCount={busyCount}
          collapsed={consoleCollapsed}
          onToggle={() => setConsoleCollapsed((value) => !value)}
        />
      </main>

      <Sidebar
        view={view}
        onViewChange={setView}
        repoCount={(data.repositories || []).length}
        env={env}
        envOk={envOk}
        connectedCount={connectedCount}
        onRefreshEnv={checkEnvironment}
        onLoginPlatform={loginPlatform}
        windowActions={windowActions}
      />

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
      <Toaster position="top-right" />
    </div>
  );
}
