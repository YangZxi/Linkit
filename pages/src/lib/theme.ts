import { useCallback, useEffect, useMemo, useSyncExternalStore } from "react";

const ThemeProps = {
  key: "theme",
  light: "light",
  dark: "dark",
} as const;

export type Theme = typeof ThemeProps.light | typeof ThemeProps.dark;

const canUseDOM = typeof window !== "undefined" && typeof document !== "undefined";

let currentTheme: Theme = ThemeProps.light;
const listeners = new Set<() => void>();

function readThemeFromStorage(): Theme {
  if (!canUseDOM) return ThemeProps.light;
  const stored = window.localStorage.getItem(ThemeProps.key) as Theme | null;
  return stored || ThemeProps.light;
}

function applyThemeToDom(theme: Theme) {
  if (!canUseDOM) return;

  window.localStorage.setItem(ThemeProps.key, theme);

  document.documentElement.classList.remove(ThemeProps.light, ThemeProps.dark);
  document.documentElement.classList.add(theme);

  applyThemeWrapperBg(theme);
}

function applyThemeWrapperBg(theme: Theme) {
  if (!canUseDOM) return;
  const themeWrapper = document.querySelector(".theme-wrapper");
  const bgImg = theme === "light" ? "/web_bg.png" : "/web_bg_dark.png";
  if (themeWrapper) (themeWrapper as HTMLElement).style.backgroundImage = `url(${bgImg})`;
}

function emit() {
  listeners.forEach((l) => l());
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function getSnapshot() {
  return currentTheme;
}

function getServerSnapshot() {
  // SSR 时的快照
  return ThemeProps.light;
}

// 初始化：只在浏览器做一次
if (canUseDOM) {
  currentTheme = readThemeFromStorage();
  applyThemeToDom(currentTheme);
}

function setThemeGlobal(next: Theme) {
  if (next === currentTheme) return;
  currentTheme = next;
  applyThemeToDom(next);
  emit();
}

export function useTheme() {
  const theme = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);

  const isDark = useMemo(() => theme === ThemeProps.dark, [theme]);
  const isLight = useMemo(() => theme === ThemeProps.light, [theme]);

  useEffect(() => {
    applyThemeWrapperBg(theme);
  }, [theme]);

  const setLightTheme = useCallback(() => setThemeGlobal(ThemeProps.light), []);
  const setDarkTheme = useCallback(() => setThemeGlobal(ThemeProps.dark), []);
  const toggleTheme = useCallback(() => {
    setThemeGlobal(theme === ThemeProps.dark ? ThemeProps.light : ThemeProps.dark);
  }, [theme]);

  return { theme, isDark, isLight, setLightTheme, setDarkTheme, toggleTheme };
}
