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
];

const query = new URLSearchParams(window.location.search);
const preview = {
  GetData: async () => ({
    repositories: query.get("fixture") === "states" ? previewRepositories : [],
    settings: {
      concurrency: 2,
      syncLfs: false,
      proxy: "",
      prefix: "",
      suffix: "",
    },
  }),
  CheckEnvironment: async () => ({
    git: { installed: true },
    gitLfs: { installed: false },
    github: { installed: true, authenticated: true, login: "workspace" },
    gitlab: { installed: false, authenticated: false },
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
