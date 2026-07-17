export const queryKeys = {
  projects: ["projects"] as const,
  boards: (project: string) => ["boards", project] as const,
  snapshot: (project: string, board: string, includeDone = false) =>
    ["snapshot", project, board, includeDone] as const,
  archived: (project: string, board: string) => ["archived", project, board] as const,
  stats: (project = "", board = "") => ["stats", project, board] as const,
};
