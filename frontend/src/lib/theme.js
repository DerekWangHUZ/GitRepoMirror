/**
 * 外观状态 Store —— GitRepoMirror 精简版。
 *
 * 结构与行为参考 macUI-SolidGlass 模板本机副本（web/src/lib/theme.ts），
 * 复制日期 2026-11-22：`dark` class 机制、system/light/dark 偏好、
 * useSyncExternalStore 订阅全部保留；强调色、模糊半径等本应用未用到的
 * 材质开关已按需裁剪。token 禁止手工微调。
 *
 * @typedef {"dark" | "light"} Theme
 * @typedef {"system" | "dark" | "light"} ThemePreference
 * @typedef {{ themePreference: ThemePreference, theme: Theme }} MaterialPrefs
 */
import { useCallback, useSyncExternalStore } from "react";

const THEME_KEY = "git-repo-mirror:theme-pref";
const LEGACY_THEME_KEY = "solid-glass:theme-pref";
const VALID_PREFERENCES = new Set(["system", "dark", "light"]);
const listeners = new Set();

function getQueryParams() {
  try {
    const hash = window.location.hash;
    const qIdx = hash.indexOf("?");
    const search =
      window.location.search || (qIdx >= 0 ? hash.slice(qIdx) : "");
    return new URLSearchParams(search);
  } catch {
    return null;
  }
}

function getQueryTheme() {
  const t = getQueryParams()?.get("theme");
  return t === "light" || t === "dark" ? t : null;
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

let currentPrefs = computeInitialPrefs();

function computeInitialPrefs() {
  const themePreference = readStoredPref();
  const resolved =
    themePreference === "system"
      ? systemDark()
        ? "dark"
        : "light"
      : themePreference;
  return { themePreference, theme: resolved };
}

function applyToDom(prefs) {
  // Solid Glass / macOS 骨架依赖 html 上的 `dark` class 切换深浅色。
  document.documentElement.classList.toggle("dark", prefs.theme === "dark");
}

// 模块加载即落定一次 DOM 状态，避免首帧闪烁。
if (typeof document !== "undefined") {
  applyToDom(currentPrefs);
  try {
    const mql = window.matchMedia("(prefers-color-scheme: dark)");
    mql.addEventListener("change", () => {
      if (currentPrefs.themePreference === "system" && !getQueryTheme()) {
        updatePrefs({ theme: mql.matches ? "dark" : "light" });
      }
    });
    window.addEventListener("hashchange", () => {
      const forcedTheme = getQueryTheme();
      if (forcedTheme) {
        updatePrefs({ themePreference: forcedTheme, theme: forcedTheme });
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

  return {
    theme: prefs.theme,
    themePreference: prefs.themePreference,
    resolvedTheme: prefs.theme,
    setTheme,
  };
}

/** 在首帧之前落定主题状态（供测试或 SSR 场景手动调用）。 */
export function applyStoredTheme() {
  applyToDom(currentPrefs);
}
