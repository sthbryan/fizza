import { fizzaApi } from "@/lib/api";

export const boardApi = {
  list: (project: string) => fizzaApi.listBoards(project),
  create: (project: string, name: string, columns?: string) =>
    fizzaApi.createBoard(project, name, columns),
  delete: (project: string, board: string) =>
    fizzaApi.deleteBoard(project, board),
  createColumn: (project: string, board: string, name: string) =>
    fizzaApi.createColumn(project, board, name),
  deleteColumn: (
    project: string,
    board: string,
    name: string,
    force = false
  ) => fizzaApi.deleteColumn(project, board, name, force),
  snapshot: (project: string, board: string, includeDone = false) =>
    fizzaApi.snapshot(project, board, includeDone),
  listArchived: (project: string, board: string) =>
    fizzaApi.listArchived(project, board),
  archiveDone: (project: string, board: string) =>
    fizzaApi.archiveDone(project, board),
};
