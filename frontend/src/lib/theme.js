/**
 * 外观状态 Store —— GitRepoMirror 精简版。
 *
 * 结构与行为参考 macUI-SolidGlass 模板本机副本（web/src/lib/theme.ts），
 * 复制日期 2026-11-22：`dark` class 机制、system/light/dark 偏好、
 * 8 色 Apple 强调色（html[data-accent]，默认 graphite）、
 * useSyncExternalStore 订阅全部保留；模糊半径、透明度等本应用未用到的
 * 材质开关已按需裁剪（透明度偏好改由系统 prefers-reduced-transparency
 * matchMedia 实时映射到 html.is-reduced-transparency；动态效果同理映射
 * html.is-reduced-motion）。token 禁止手工微调。
 *
 * 兼容 `?theme=light|dark` 与 `?accent=blue|...` URL 调试参数。
 *
 * @typedef {"dark" | "light"} Theme
 * @typedef {"system" | "dark" | "light"} ThemePreference
 * @typedef {"blue" | "purple" | "pink" | "red" | "orange" | "yellow" | "green" | "graphite"} AccentColor
 * @typedef {{ themePreference: ThemePreference, theme: Theme, accent: AccentColor }} MaterialPrefs
 */
import { useCallback, useSyncExternalStore } from "react";

export const ACCENT_OPTIONS = [
  { id: "blue", label: "蓝色", swatch: "#0088ff" },
  { id: "purple", label: "紫色", swatch: "#cb30e0" },
  { id: "pink", label: "粉色", swatch: "#ff2d55" },
  { id: "red", label: "红色", swatch: "#ff383c" },
  { id: "orange", label: "橙色", swatch: "#ff8d28" },
  { id: "yellow", label: "黄色", swatch: "#ffcc00" },
  { id: "green", label: "绿色", swatch: "#34c759" },
  { id: "graphite", label: "石墨（原版单色）", swatch: "#8e8e93" },
];

const THEME_KEY = "git-repo-mirror:theme-pref";
const LEGACY_THEME_KEY = "solid-glass:theme-pref";
const ACCENT_KEY = "git-repo-mirror:accent";
const VALID_PREFERENCES = new Set(["system", "dark", "light"]);
const VALID_ACCENTS = new Set(ACCENT_OPTIONS.map((option) => option.id));
const listeners = new Set();

function getQueryParams() {
  try {
    const hash = window.location.hash;
    const qIdx = hash.indexOf("?");
    const search = window.location.search || (qIdx >= 0 ? hash.slice(qIdx) : "");
    return new URLSearchParams(search);
  } catch {
    return null;
  }
}

function getQueryTheme() {
  const t = getQueryParams()?.get("theme");
  return t === "light" || t === "dark" ? t : null;
}

function getQueryAccent() {
  const a = getQueryParams()?.get("accent");
  return a && VALID_ACCENTS.has(a) ? a : null;
}

function systemDark() {
  try {
    return window.matchMedia("(prefers-color-scheme: dark)").matches;
  } catch {
    return true;
  }
}

function readStoredPref() {
  const forced = getQueryTheme();
  if (forced) return forced;
  try {
    const v =
      localStorage.getItem(THEME_KEY) ?? localStorage.getItem(LEGACY_THEME_KEY);
    if (v && VALID_PREFERENCES.has(v)) return v;
  } catch {
    // ignore storage errors
  }
  // 默认跟随系统（沿用旧版 prefers-color-scheme 行为）。
  return "system";
}

function readStoredAccent() {
  const forced = getQueryAccent();
  if (forced) return forced;
  try {
    const v = localStorage.getItem(ACCENT_KEY);
    if (v && VALID_ACCENTS.has(v)) return v;
  } catch {
    // ignore
  }
  // 默认石墨：保留 Solid Glass 原版黑白灰单色设计。
  return "graphite";
}

let currentPrefs = computeInitialPrefs();

function computeInitialPrefs() {
  const themePreference = readStoredPref();
  const resolved =
    themePreference === "system"
      ? systemDark()
        ? "dark"
        : "light"
      : themePreference;
  return { themePreference, theme: resolved, accent: readStoredAccent() };
}

function applyToDom(prefs) {
  // Solid Glass / macOS 骨架依赖 html 上的 `dark` class 与 data-accent。
  const root = document.documentElement;
  root.classList.toggle("dark", prefs.theme === "dark");
  root.dataset.accent = prefs.accent;
  syncMediaClasses();
}

/** 系统无障碍偏好实时映射：html.is-reduced-transparency / html.is-reduced-motion。 */
function syncMediaClasses() {
  try {
    document.documentElement.classList.toggle(
      "is-reduced-transparency",
      window.matchMedia("(prefers-reduced-transparency: reduce)").matches,
    );
    document.documentElement.classList.toggle(
      "is-reduced-motion",
      window.matchMedia("(prefers-reduced-motion: reduce)").matches,
    );
  } catch {
    // ignore
  }
}

// 模块加载即落定一次 DOM 状态，避免首帧闪烁；matchMedia 变化实时跟随。
if (typeof document !== "undefined") {
  applyToDom(currentPrefs);
  try {
    const mql = window.matchMedia("(prefers-color-scheme: dark)");
    mql.addEventListener("change", () => {
      if (currentPrefs.themePreference === "system" && !getQueryTheme()) {
        updatePrefs({ theme: mql.matches ? "dark" : "light" });
      }
    });
    const transparencyMql = window.matchMedia("(prefers-reduced-transparency: reduce)");
    transparencyMql.addEventListener("change", syncMediaClasses);
    const motionMql = window.matchMedia("(prefers-reduced-motion: reduce)");
    motionMql.addEventListener("change", syncMediaClasses);
    window.addEventListener("hashchange", () => {
      const forcedTheme = getQueryTheme();
      const forcedAccent = getQueryAccent();
      const patch = {};
      if (forcedTheme) {
        patch.themePreference = forcedTheme;
        patch.theme = forcedTheme;
      }
      if (forcedAccent) {
        patch.accent = forcedAccent;
      }
      if (Object.keys(patch).length > 0) {
        updatePrefs(patch);
      }
    });
  } catch {
    // ignore
  }
}

function updatePrefs(patch) {
  currentPrefs = { ...currentPrefs, ...patch };
  applyToDom(currentPrefs);
  listeners.forEach((notify) => notify());
}

function persist(key, value) {
  try {
    localStorage.setItem(key, value);
  } catch {
    // ignore
  }
}

export function useTheme() {
  const prefs = useSyncExternalStore(
    (notify) => {
      listeners.add(notify);
      return () => {
        listeners.delete(notify);
      };
    },
    () => currentPrefs,
    () => currentPrefs,
  );

  const setTheme = useCallback((next) => {
    const resolved =
      next === "system" ? (systemDark() ? "dark" : "light") : next;
    persist(THEME_KEY, next);
    persist(LEGACY_THEME_KEY, resolved);
    updatePrefs({ themePreference: next, theme: resolved });
  }, []);

  const setAccent = useCallback((next) => {
    if (!VALID_ACCENTS.has(next)) return;
    persist(ACCENT_KEY, next);
    updatePrefs({ accent: next });
  }, []);

  return {
    theme: prefs.theme,
    themePreference: prefs.themePreference,
    resolvedTheme: prefs.theme,
    accent: prefs.accent,
    setTheme,
    setAccent,
  };
}

/** 在首帧之前落定主题状态（供测试或 SSR 场景手动调用）。 */
export function applyStoredTheme() {
  applyToDom(currentPrefs);
}
