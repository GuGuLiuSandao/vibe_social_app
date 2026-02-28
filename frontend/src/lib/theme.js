export const DEFAULT_THEME = "dark";
export const THEME_STORAGE_KEY = "ui:theme";

export const THEME_OPTIONS = [
  { id: "dark", label: "暗" },
  { id: "light", label: "亮" },
  { id: "neon", label: "霓虹" },
  { id: "cold", label: "冷峻" },
];

const SUPPORTED_THEME_SET = new Set(THEME_OPTIONS.map((item) => item.id));
const LEGACY_THEME_ALIAS = {
  deepblue: "cold",
};

export function normalizeTheme(theme) {
  const mapped = LEGACY_THEME_ALIAS[theme] || theme;
  return SUPPORTED_THEME_SET.has(mapped) ? mapped : DEFAULT_THEME;
}

export function getStoredTheme() {
  if (typeof window === "undefined") return DEFAULT_THEME;
  try {
    return normalizeTheme(window.localStorage.getItem(THEME_STORAGE_KEY));
  } catch {
    return DEFAULT_THEME;
  }
}

export function setStoredTheme(theme) {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(THEME_STORAGE_KEY, normalizeTheme(theme));
  } catch {
    // ignore localStorage write errors (private mode or quota)
  }
}

export function applyTheme(theme) {
  const normalized = normalizeTheme(theme);
  if (typeof document !== "undefined") {
    document.documentElement.setAttribute("data-theme", normalized);
  }
  return normalized;
}
