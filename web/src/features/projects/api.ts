import { fizzaApi } from "@/lib/api";

export const projectsApi = {
  list: () => fizzaApi.listProjects(),
  create: (name: string, description: string) => fizzaApi.createProject(name, description),
  update: (name: string, patch: { name?: string; description?: string }) =>
    fizzaApi.updateProject(name, patch),
  delete: (name: string) => fizzaApi.deleteProject(name),
};
