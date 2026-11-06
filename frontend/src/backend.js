import * as App from "../wailsjs/go/main/App.js";
import { EventsOn as RuntimeEventsOn } from "../wailsjs/runtime/runtime.js";

const previewRepositories = [
  {
    id: "healthy",
    source: { platform: "github", displayName: "team/source" },
    target: {
      platform: "gitlab",
      displayName: "backup/source",
      webUrl: "https://gitlab.com/backup/source",
    },
    managementMode: "cli",
    mode: "mirror",
    status: "healthy",
    lastSync: "2026-08-31T08:00:00Z",
  },
  {
    id: "syncing",
    source: { platform: "gitlab", displayName: "team/app" },
    target: { platform: "github", displayName: "backup/app" },
    managementMode: "cli",
    mode: "shallow",
    status: "syncing",
    lastSync: "0001-01-01T00:00:00Z",
  },
  {
    id: "failed",
    source: { platform: "generic", displayName: "code.example/tools" },
    target: { platform: "generic", displayName: "mirror.example/tools" },
    managementMode: "git-only",
    mode: "mirror",
    status: "failed",
    lastSync: "0001-01-01T00:00:00Z",
    lastError: "network unavailable",
  },
  {
    id: "gitee-mirror",
    source: { platform: "gitee", displayName: "team/gitee-app" },
    target: { platform: "codeberg", displayName: "team/gitee-app" },
    managementMode: "cli",
    mode: "mirror",
    status: "healthy",
    lastSync: "2026-11-01T10:30:00Z",
  },
];

const query = new URLSearchParams(window.location.search);
const preview = {
  GetData: async () => ({
    repositories: query.get("fixture") === "states" ? previewRepositories : [],
    settings: {
      concurrency: 2,
      syncLfs: false,
      pruneTargetRefs: false,
      proxy: "",
      prefix: "",
      suffix: "",
    },
  }),
  CheckEnvironment: async () => ({
    git: { installed: true },
    gitLfs: { installed: false },
    github: { installed: true, authenticated: true, login: "workspace" },
    gitlab: { installed: true, authenticated: false, authState: "missing" },
    gitee: { installed: true, authenticated: true, login: "DerekWangHUZ" },
    codeberg: {
      installed: false,
      authenticated: false,
      message: "未安装 fj，可使用完整 URL 降级同步",
    },
    azuredevops: {
      installed: true,
      authenticated: false,
      authState: "invalid",
      message: "Azure DevOps CLI 凭据已失效，请重新登录",
    },
  }),
  StartLogin: async () => {},
  CreateMirror: async () => {},
  SyncRepository: async () => {},
  SyncAll: async () => {},
  RemoveRepository: async () => {},
  SaveSettings: async () => {},
  OpenURL: async () => {},
  Minimise: async () => {},
  ToggleMaximise: async () => {},
  Close: async () => {},
};

export const backend = import.meta.env.DEV && !window.go ? preview : App;
export const EventsOn = window.runtime ? RuntimeEventsOn : () => () => {};
