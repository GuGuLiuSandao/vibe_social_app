import { THEME_OPTIONS } from "../lib/theme";

export default function ThemeSwitcher({ theme, onChange, compact = false }) {
  return (
    <div
      className={`theme-switcher ${compact ? "theme-switcher-compact" : ""}`}
      role="group"
      aria-label="主题切换"
    >
      {THEME_OPTIONS.map((item) => {
        const active = theme === item.id;
        return (
          <button
            key={item.id}
            type="button"
            aria-pressed={active}
            onClick={() => onChange(item.id)}
            className={`theme-switcher-btn ${active ? "is-active" : ""}`}
          >
            <span className={`theme-dot theme-dot-${item.id}`} />
            {item.label}
          </button>
        );
      })}
    </div>
  );
}
