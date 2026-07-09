import { fizzaApi } from "@/lib/api";

export const statsApi = {
  get: (project?: string, board?: string) => fizzaApi.stats(project, board),
};
