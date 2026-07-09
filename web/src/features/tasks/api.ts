import { fizzaApi } from "@/lib/api";

export const tasksApi = {
  create: (
    project: string,
    board: string,
    input: {
      title: string;
      description?: string;
      column?: string;
      priority?: string;
      due?: string;
    }
  ) => fizzaApi.createTask(project, board, input),

  update: (
    id: number,
    patch: {
      title?: string;
      desc?: string;
      priority?: string;
      due?: string;
      clear_due?: boolean;
    }
  ) => fizzaApi.updateTask(id, patch),

  move: (
    id: number,
    input: { project: string; board: string; column: string; before?: string }
  ) => fizzaApi.moveTask(id, input),

  delete: (id: number) => fizzaApi.deleteTask(id),
};
