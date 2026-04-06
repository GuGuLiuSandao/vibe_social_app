import { THEME_OPTIONS } from "../lib/theme";
import { Button } from "./ui/button";
import { cn } from "../lib/utils";

export default function ThemeSwitcher({ theme, onChange, compact = false }) {
  return (
    <div
      className={cn(
        "inline-flex items-center gap-1 rounded-lg border border-border bg-card p-1 backdrop-blur",
        compact ? "scale-[0.95]" : ""
      )}
      role="group"
      aria-label="主题切换"
    >
      {THEME_OPTIONS.map((item) => {
        const active = theme === item.id;
        return (
          <Button
            key={item.id}
            type="button"
            size="sm"
            variant={active ? "default" : "ghost"}
            aria-pressed={active}
            onClick={() => onChange(item.id)}
            className={cn(
              "h-8 gap-1.5 rounded-md px-2 text-[11px] font-semibold",
              active ? "text-primary-foreground" : "text-muted-foreground"
            )}
          >
            <span className={`theme-dot theme-dot-${item.id}`} />
            {item.label}
          </Button>
        );
      })}
    </div>
  );
}
