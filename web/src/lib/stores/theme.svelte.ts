// Theme state. Defaults to dark when no preference has been stored.
const STORAGE_KEY = "netlog.theme";

function initial(): "dark" | "light" {
  if (typeof localStorage !== "undefined") {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === "light" || stored === "dark") return stored;
  }
  return "dark";
}

class ThemeStore {
  current = $state<"dark" | "light">(initial());

  toggle(): void {
    this.set(this.current === "dark" ? "light" : "dark");
  }

  set(value: "dark" | "light"): void {
    this.current = value;
    if (typeof document !== "undefined") {
      document.documentElement.classList.toggle("dark", value === "dark");
    }
    if (typeof localStorage !== "undefined") {
      localStorage.setItem(STORAGE_KEY, value);
    }
  }
}

export const theme = new ThemeStore();
