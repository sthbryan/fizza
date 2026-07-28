<script lang="ts">
  import { createMutation, createQuery, useQueryClient } from "@tanstack/svelte-query";
  import AppShell from "@/shared/layout/AppShell.svelte";
  import Button from "@/shared/ui/Button.svelte";
  import EmptyState from "@/shared/ui/EmptyState.svelte";
  import { queryKeys } from "@/lib/api";
  import {
    boardPath,
    navigate,
    lastBoardHint,
    rememberBoard,
  } from "@/lib/router/router.svelte";
  import { fizzaApi } from "@/lib/api";
  import { showStatus } from "@/lib/status/status.svelte";
  import type { Project } from "@/lib/api";
  import CreateProjectDialog from "./CreateProjectDialog.svelte";
  import EditProjectDialog from "./EditProjectDialog.svelte";
  import ConfirmDialog from "@/shared/ui/ConfirmDialog.svelte";
  import { projectsApi } from "./api";
  import { cn } from "@/lib/cn";
  import { animate } from "@/lib/animate";

  let createOpen = $state(false);
  let editing = $state<Project | null>(null);

  let pendingDelete = $state<Project | null>(null);
  const hint = lastBoardHint();
  const queryClient = useQueryClient();

  const projectsQuery = createQuery(() => ({
    queryKey: queryKeys.projects,
    queryFn: () => projectsApi.list(),
  }));

  const deleteMutation = createMutation(() => ({
    mutationFn: (name: string) => projectsApi.delete(name),
    onSuccess: async (_data, name) => {
      const stored = lastBoardHint();
      if (stored?.project === name) rememberBoard("", "");
      await queryClient.invalidateQueries({ queryKey: queryKeys.projects });
      showStatus(`Project “${name}” deleted`);
    },
    onError: (err) => {
      showStatus(err instanceof Error ? err.message : String(err), "error");
    },
  }));

  async function openProject(name: string) {
    try {
      const boards = (await fizzaApi.listBoards(name)) || [];
      const board =
        boards.find((b) => b.is_default)?.name || boards[0]?.name || "main";
      navigate(boardPath(name, board));
    } catch {
      navigate(boardPath(name, "main"));
    }
  }

  function handleEdit(e: MouseEvent, project: Project) {
    e.stopPropagation();
    e.preventDefault();
    editing = project;
  }

  function handleDelete(e: MouseEvent, project: Project) {
    e.stopPropagation();
    e.preventDefault();
    pendingDelete = project;
  }

  async function confirmDelete() {
    const target = pendingDelete;
    pendingDelete = null;
    if (target) await deleteMutation.mutateAsync(target.name);
  }

  function boardLabel(count: number | undefined) {
    const n = count ?? 0;
    return n === 1 ? "1 board" : `${n} boards`;
  }
</script>

<AppShell>
  <header
    class="border-b border-neutral-800 bg-black px-4 py-4 sm:px-6 sm:py-5"
  >
    <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div class="min-w-0">
        <div class="mb-2 text-label font-mono uppercase text-neutral-500">
          fizza / projects
        </div>
        <div class="flex flex-wrap items-baseline gap-2 sm:gap-3">
          <h1 class="text-lg tracking-tight text-white">
            Projects
          </h1>
          {#if projectsQuery.data}
            <span class="font-mono text-label tabular-nums text-neutral-500">
              {projectsQuery.data.length} total
            </span>
          {/if}
        </div>
      </div>
      <Button variant="primary" onclick={() => (createOpen = true)}>
        + New project
      </Button>
    </div>
  </header>

  <main class="min-h-0 flex-1 overflow-hidden">
    {#if projectsQuery.isPending}
      <div class="p-8 text-label font-mono uppercase text-neutral-500">
        [LOADING]
      </div>
    {:else if projectsQuery.isError}
      <div class="p-8 text-label font-mono uppercase text-accent">
        [ERROR] {projectsQuery.error.message}
      </div>
    {:else if !projectsQuery.data?.length}
      <EmptyState
        title="No projects yet"
        description="Create a project to start managing boards and tasks."
        actionLabel="Create project"
        onaction={() => (createOpen = true)}
      />
    {:else}
      <div class="h-full overflow-y-auto px-4 py-5 sm:px-6 sm:py-6">
        <div
          class="divide-y divide-neutral-800 border-y border-neutral-800"
          use:animate={{ duration: 200, easing: "ease-out" }}
        >
          {#each projectsQuery.data as p (p.id)}
            {@const active = hint?.project === p.name}
            <div
              class={cn(
                "group flex min-h-14 items-center gap-3 py-3 transition-colors sm:gap-4",
                active && "bg-neutral-950"
              )}
            >
              <button
                type="button"
                class="flex min-w-0 flex-1 cursor-pointer items-center gap-3 text-left sm:gap-4"
                aria-label={`Open project ${p.name}`}
                onclick={() => void openProject(p.name)}
              >
                {#if active}
                  <span class="h-1.5 w-1.5 shrink-0 rounded-full bg-white"></span>
                {:else}
                  <span class="h-1.5 w-1.5 shrink-0 rounded-full bg-neutral-700"></span>
                {/if}
                <span class="truncate text-sm tracking-tight text-neutral-100">
                  {p.name}
                </span>
                <span class="hidden text-label font-mono uppercase text-neutral-500 sm:inline">
                  #{p.id}
                </span>
                <span class="hidden truncate text-sm text-neutral-500 md:inline">
                  {p.description?.trim() || "No description"}
                </span>
              </button>
              <span class="hidden text-label font-mono uppercase text-neutral-500 sm:inline">
                {boardLabel(p.board_count)}
              </span>
              <div class="flex shrink-0 items-center">
                <button
                  type="button"
                  title="Edit project"
                  class="flex min-h-8 cursor-pointer items-center px-3 text-label font-mono uppercase text-neutral-500 transition-colors hover:text-neutral-200"
                  onclick={(e) => handleEdit(e, p)}
                >
                  Edit
                </button>
                <button
                  type="button"
                  title="Delete project"
                  class="flex min-h-8 cursor-pointer items-center px-3 text-label font-mono uppercase text-neutral-500 transition-colors hover:text-accent"
                  onclick={(e) => handleDelete(e, p)}
                >
                  Del
                </button>
              </div>
            </div>
          {/each}
        </div>
        <div class="mt-8 flex justify-center">
          <Button variant="secondary" onclick={() => (createOpen = true)}>
            + Create project
          </Button>
        </div>
      </div>
    {/if}
  </main>
</AppShell>

<CreateProjectDialog open={createOpen} onclose={() => (createOpen = false)} />
<EditProjectDialog
  project={editing}
  open={editing !== null}
  onclose={() => (editing = null)}
/>
<ConfirmDialog
  open={pendingDelete !== null}
  title={pendingDelete ? `Delete project “${pendingDelete.name}”?` : ""}
  description={pendingDelete
    ? `All boards, columns, and tasks in this project will be permanently deleted. This cannot be undone.`
    : ""}
  confirmLabel="Delete project"
  onclose={() => (pendingDelete = null)}
  onconfirm={confirmDelete}
/>