import type {
  Board,
  BoardSnapshot,
  Column,
  Project,
  Task,
} from "./types";

interface Envelope<T> {
  ok: boolean;
  data?: T;
  error?: { code: string; message: string };
}

export class ApiError extends Error {
  status: number;
  code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export async function api<T>(
  method: string,
  path: string,
  body?: unknown
): Promise<T> {
  const res = await fetch(path, {
    method,
    headers:
      body !== undefined ? { "Content-Type": "application/json" } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  const text = await res.text();
  let payload: Envelope<T> | null = null;
  try {
    payload = text ? (JSON.parse(text) as Envelope<T>) : null;
  } catch {
    throw new ApiError(text || res.statusText, res.status);
  }

  if (!payload?.ok) {
    throw new ApiError(
      payload?.error?.message || res.statusText || "request failed",
      res.status,
      payload?.error?.code
    );
  }
  return payload.data as T;
}

function enc(s: string) {
  return encodeURIComponent(s);
}

export const fizzaApi = {
  listProjects: () => api<Project[]>("GET", "/v1/projects"),
  createProject: (name: string, description = "") =>
    api<Project>("POST", "/v1/projects", { name, description }),
  updateProject: (
    name: string,
    patch: { name?: string; description?: string }
  ) => api<Project>("PATCH", `/v1/projects/${enc(name)}`, patch),
  deleteProject: (name: string) =>
    api<{ deleted: string; id: number }>(
      "DELETE",
      `/v1/projects/${enc(name)}?force=true`
    ),

  listBoards: (project: string) =>
    api<Board[]>("GET", `/v1/projects/${enc(project)}/boards`),
  createBoard: (project: string, name: string, columns?: string) =>
    api<Board>("POST", `/v1/projects/${enc(project)}/boards`, {
      name,
      ...(columns ? { columns } : {}),
    }),
  deleteBoard: (project: string, board: string) =>
    api<{ deleted: string; id: number }>(
      "DELETE",
      `/v1/projects/${enc(project)}/boards/${enc(board)}?force=true`
    ),

  createColumn: (project: string, board: string, name: string) =>
    api<Column>(
      "POST",
      `/v1/projects/${enc(project)}/boards/${enc(board)}/columns`,
      { name }
    ),
  deleteColumn: (project: string, board: string, name: string, force = false) =>
    api<{ deleted: string }>(
      "DELETE",
      `/v1/projects/${enc(project)}/boards/${enc(board)}/columns/${enc(name)}${
        force ? "?force=true" : ""
      }`
    ),

  snapshot: (project: string, board: string) =>
    api<BoardSnapshot>(
      "GET",
      `/v1/projects/${enc(project)}/boards/${enc(board)}/snapshot`
    ),

  createTask: (
    project: string,
    board: string,
    input: {
      title: string;
      description?: string;
      column?: string;
      priority?: string;
      due?: string;
    }
  ) =>
    api<Task>(
      "POST",
      `/v1/projects/${enc(project)}/boards/${enc(board)}/tasks`,
      input
    ),

  updateTask: (
    id: number,
    patch: {
      title?: string;
      desc?: string;
      priority?: string;
      due?: string;
      clear_due?: boolean;
    }
  ) => api<Task>("PATCH", `/v1/tasks/${id}`, patch),

  moveTask: (
    id: number,
    input: { project: string; board: string; column: string; before?: string }
  ) => api<Task>("POST", `/v1/tasks/${id}/move`, input),

  deleteTask: (id: number) =>
    api<{ deleted: number }>("DELETE", `/v1/tasks/${id}?force=true`),
};
