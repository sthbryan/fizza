import { fizzaApi } from "@/lib/api";

export const projectsApi = {
  list: () => fizzaApi.listProjects(),
  create: (name: string, description: string) =>
    fizzaApi.createProject(name, description),
  delete: (name: string) => fizzaApi.deleteProject(name),
};
