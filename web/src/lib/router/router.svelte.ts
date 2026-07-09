/**
 * Minimal history-based SPA router.
 * URL is the source of truth for navigation — not a global store.
 */

export type Route =
  | { name: "home" }
  | { name: "projects" }
  | { name: "stats" }
  | { name: "board"; project: string; board: string }
  | { name: "notfound" };

const STORAGE_PROJECT = "fizza.project";
const STORAGE_BOARD = "fizza.board";

let path = $state(window.location.pathname);

function syncFromLocation() {
  path = window.location.pathname;
}

export function getPath(): string {
  return path;
}

export function parseRoute(pathname: string = path): Route {
  if (pathname === "/" || pathname === "") return { name: "home" };
  if (pathname === "/projects") return { name: "projects" };
  if (pathname === "/stats") return { name: "stats" };

  const m = pathname.match(/^\/p\/([^/]+)\/b\/([^/]+)\/?$/);
  if (m) {
    return {
      name: "board",
      project: decodeURIComponent(m[1]),
      board: decodeURIComponent(m[2]),
    };
  }
  return { name: "notfound" };
}

export function getRoute(): Route {
  return parseRoute(path);
}

export function boardPath(project: string, board: string): string {
  return `/p/${encodeURIComponent(project)}/b/${encodeURIComponent(board)}`;
}

export function navigate(to: string, replace = false) {
  if (replace) {
    history.replaceState({}, "", to);
  } else {
    history.pushState({}, "", to);
  }
  syncFromLocation();
}

export function rememberBoard(project: string, board: string) {
  if (project) localStorage.setItem(STORAGE_PROJECT, project);
  else localStorage.removeItem(STORAGE_PROJECT);
  if (board) localStorage.setItem(STORAGE_BOARD, board);
  else localStorage.removeItem(STORAGE_BOARD);
}

export function lastBoardHint(): { project: string; board: string } | null {
  const project = localStorage.getItem(STORAGE_PROJECT) || "";
  const board = localStorage.getItem(STORAGE_BOARD) || "";
  if (project && board) return { project, board };
  return null;
}

if (typeof window !== "undefined") {
  window.addEventListener("popstate", syncFromLocation);
}
