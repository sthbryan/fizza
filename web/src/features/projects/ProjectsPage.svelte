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
    class="border-b border-[var(--color-border-subtle)] bg-[var(--color-bg)] px-4 py-4 sm:px-6 sm:py-5"
  >
    <div class="flex flex-col gap-3.5 sm:flex-row sm:items-center sm:justify-between">
      <div class="min-w-0">
        <div class="mb-1.5 text-sm text-[var(--color-text-muted)]">
          fizza / projects
        </div>
        <div class="flex flex-wrap items-baseline gap-2 sm:gap-3">
          <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">
            Projects
          </h1>
          {#if projectsQuery.data}
            <span class="text-base text-[var(--color-text-muted)]">
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
      <div class="p-8 text-base text-[var(--color-text-muted)]">Loading…</div>
    {:else if projectsQuery.isError}
      <div class="p-8 text-base text-[var(--color-danger)]">
        {projectsQuery.error.message}
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
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {#each projectsQuery.data as p (p.id)}
            {@const active = hint?.project === p.name}
            <div
              class={
                "group relative rounded-2xl border p-5 text-left transition sm:p-6 " +
                (active
                  ? "border-[var(--color-accent)]/50 bg-[var(--color-bg-hover)] ring-1 ring-[var(--color-accent)]/30"
                  : "border-[var(--color-border-subtle)] bg-[var(--color-bg-card)] hover:border-[var(--color-border)] hover:bg-[var(--color-bg-hover)]")
              }
            >
              <button
                type="button"
                class="absolute inset-0 cursor-pointer rounded-2xl"
                aria-label={`Open project ${p.name}`}
                onclick={() => void openProject(p.name)}
              ></button>
              <div class="relative z-10 pointer-events-none">
                <div class="mb-2.5 flex items-start justify-between gap-2">
                  <h3 class="truncate text-lg font-semibold tracking-tight">
                    {p.name}
                  </h3>
                  <div class="flex shrink-0 items-center gap-1.5">
                    {#if active}
                      <span
                        class="rounded-lg bg-[var(--color-accent)]/15 px-2.5 py-1 text-xs font-medium text-[var(--color-accent)]"
                      >
                        Recent
                      </span>
                    {/if}
                    <button
                      type="button"
                      title="Edit project"
                      class="pointer-events-auto cursor-pointer rounded-lg px-2 py-1 text-sm text-[var(--color-text-muted)] opacity-100 transition hover:bg-white/5 hover:text-[var(--color-text)] sm:opacity-0 sm:group-hover:opacity-100"
                      onclick={(e) => handleEdit(e, p)}
                    >
                      Edit
                    </button>
                    <button
                      type="button"
                      title="Delete project"
                      class="pointer-events-auto cursor-pointer rounded-lg px-2 py-1 text-sm text-[var(--color-text-muted)] opacity-100 transition hover:bg-[var(--color-danger)]/10 hover:text-[var(--color-danger)] sm:opacity-0 sm:group-hover:opacity-100"
                      onclick={(e) => handleDelete(e, p)}
                    >
                      Del
                    </button>
                  </div>
                </div>
                <p
                  class="line-clamp-2 min-h-12 text-base text-[var(--color-text-muted)]"
                >
                  {p.description?.trim() || "No description"}
                </p>
                <div class="mt-5 flex items-center justify-between gap-3">
                  <span class="text-sm text-[var(--color-text-muted)]">
                    <span class="font-mono text-xs">#{p.id}</span>
                    <span class="mx-1.5 opacity-40">·</span>
                    {boardLabel(p.board_count)}
                  </span>
                  <span class="shrink-0 text-sm text-[var(--color-text-secondary)]">
                    Open board →
                  </span>
                </div>
              </div>
            </div>
          {/each}
        </div>
        <div class="mt-5 flex justify-center sm:mt-7">
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
