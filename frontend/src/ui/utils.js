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
export const platformLabel = (platform) =>
  ({ github: "GitHub", gitlab: "GitLab", generic: "其他 Git" })[platform] ||
  "Git";
export const errorText = (error) => String(error).replace(/^Error:\s*/, "");
