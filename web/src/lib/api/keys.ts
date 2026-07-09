export const queryKeys = {
  projects: ["projects"] as const,
  boards: (project: string) => ["boards", project] as const,
  snapshot: (project: string, board: string) =>
    ["snapshot", project, board] as const,
};
