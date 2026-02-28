import { useEffect, useState } from "react";
import { applyTheme, getStoredTheme, setStoredTheme } from "./theme";

export function useThemeMode() {
  const [theme, setTheme] = useState(() => getStoredTheme());

  useEffect(() => {
    const activeTheme = applyTheme(theme);
    setStoredTheme(activeTheme);
  }, [theme]);

  return [theme, setTheme];
}
