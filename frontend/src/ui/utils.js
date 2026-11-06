export const $ = (selector, root = document) => root.querySelector(selector);
export const escapeHTML = (value = "") =>
  String(value).replace(
    /[&<>'"]/g,
    (character) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" })[
        character
      ],
  );
export const formatTime = (value) =>
  value && !value.startsWith("0001-")
    ? new Intl.DateTimeFormat("zh-CN", {
        dateStyle: "medium",
        timeStyle: "short",
      }).format(new Date(value))
    : "尚未同步";

// PLATFORMS is the single source of truth for every platform-aware template.
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
