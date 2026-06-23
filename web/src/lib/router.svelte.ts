// A tiny history-based router. The Go backend serves index.html for any non-API
// path, so client-side deep links work.
class Router {
  path = $state(typeof location !== "undefined" ? location.pathname : "/");

  constructor() {
    if (typeof window !== "undefined") {
      window.addEventListener("popstate", () => {
        this.path = location.pathname;
      });
    }
  }

  navigate(to: string): void {
    if (to === this.path) return;
    history.pushState({}, "", to);
    this.path = to;
  }

  // matched returns the route name and any :id parameter.
  get matched(): { name: string; id?: string } {
    const p = this.path;
    if (p === "/login") return { name: "login" };
    if (p === "/register") return { name: "register" };
    if (p === "/admin") return { name: "admin" };
    if (p === "/settings") return { name: "settings" };
    if (p === "/" || p === "") return { name: "nets" };
    const m = p.match(/^\/nets\/([^/]+)$/);
    if (m) return { name: "net", id: m[1] };
    return { name: "notfound" };
  }
}

export const router = new Router();

// navigate is a convenience for use in event handlers and the link action.
export function navigate(to: string): void {
  router.navigate(to);
}

// link is a Svelte action: <a href="/x" use:link> performs client-side nav.
export function link(node: HTMLAnchorElement) {
  function onClick(e: MouseEvent) {
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
    const href = node.getAttribute("href");
    if (!href || href.startsWith("http")) return;
    e.preventDefault();
    router.navigate(href);
  }
  node.addEventListener("click", onClick);
  return { destroy: () => node.removeEventListener("click", onClick) };
}
