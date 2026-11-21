// cn 来自 macUI-SolidGlass 模板本机副本（web/src/lib/utils.ts），复制日期 2026-11-22。
// token 禁止手工微调。
import { clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs) {
  return twMerge(clsx(inputs));
}

// ---- GitRepoMirror 业务常量与工具（迁移自 src/ui/utils.js，逻辑不变） ----

export const formatTime = (value) =>
  value && !String(value).startsWith("0001-")
    ? new Intl.DateTimeFormat("zh-CN", {
        dateStyle: "medium",
        timeStyle: "short",
      }).format(new Date(value))
    : "尚未同步";

// PLATFORMS is the single source of truth for every platform-aware view.
// `id` matches the backend EnvironmentStatus/MirrorRequest identifiers.
export const PLATFORMS = [
  { id: "github", label: "GitHub", cli: "gh" },
  { id: "gitlab", label: "GitLab", cli: "glab" },
  { id: "gitee", label: "Gitee", cli: "gitee" },
  { id: "codeberg", label: "Codeberg", cli: "fj" },
  {
    id: "azuredevops",
    label: "Azure DevOps",
    cli: "az",
    namespaceHint: "组织/项目",
    fixedVisibility: true,
  },
  { id: "generic", label: "其他 Git 平台", cli: "" },
];

export const platformMeta = (platform) =>
  PLATFORMS.find((item) => item.id === platform);
export const platformLabel = (platform) =>
  platformMeta(platform)?.label || "Git";
export const managedPlatforms = () =>
  PLATFORMS.filter((platform) => platform.cli);
export const errorText = (error) => String(error).replace(/^Error:\s*/, "");

// 目标仓库表单在「托管创建 / 完整 URL」两种模式之间切换所需的派生数据。
// 迁移自 templates.js 的 targetFields，React 版直接消费数据而非 HTML 字符串。
export function targetFieldsModel(platform, env) {
  const meta = platformMeta(platform);
  const tool = meta ? env[platform] : undefined;
  const managed =
    Boolean(meta?.cli) && Boolean(tool?.installed && tool?.authenticated);
  if (managed) {
    const namespaceHint = meta.namespaceHint || "用户、组织或群组";
    return {
      managed,
      mode: "managed",
      namespaceHint,
      defaultNamespace: tool?.login || "",
      fixedVisibility: Boolean(meta?.fixedVisibility),
    };
  }
  const cli = meta?.cli;
  const message =
    platform === "generic"
      ? "请先在目标平台创建仓库，然后填写完整推送 URL。"
      : `${tool?.message || `${cli} 缺失或未登录`}；当前只能关联已存在仓库的完整推送 URL。`;
  return {
    managed,
    mode: "url",
    message,
    canLoginToTarget: platform !== "generic" && Boolean(tool?.installed),
  };
}

export const STATUS_LABELS = {
  healthy: "正常",
  syncing: "同步中",
  failed: "失败",
};

export const environmentRowModel = (label, tool = {}, platform = "") => {
  const ok = Boolean(tool.installed) && (!platform || tool.authenticated);
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
  return { label, ok, detail, message: tool.message || "" };
};

export const loginButtonLabel = (tool = {}, platform) =>
  tool.authState === "missing" || tool.authState === "invalid"
    ? `重新登录 ${platformLabel(platform)}`
    : `登录 ${platformLabel(platform)}`;
