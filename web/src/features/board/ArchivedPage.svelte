<script lang="ts">
  import { createMutation, createQuery, useQueryClient } from "@tanstack/svelte-query";
  import AppShell from "@/shared/layout/AppShell.svelte";
  import Button from "@/shared/ui/Button.svelte";
  import EmptyState from "@/shared/ui/EmptyState.svelte";
  import Badge from "@/shared/ui/Badge.svelte";
  import { queryKeys, type Task } from "@/lib/api";
  import {
    boardPath,
    navigate,
    rememberBoard,
  } from "@/lib/router/router.svelte";
  import { showToast } from "@/lib/toast/toast.svelte";
  import { boardApi } from "./api";
  import { tasksApi } from "@/features/tasks/api";
  import ConfirmDialog from "@/shared/ui/ConfirmDialog.svelte";

  interface Props {
    project: string;
    board: string;
  }

  let { project, board }: Props = $props();
  const queryClient = useQueryClient();

  let pendingDelete = $state<Task | null>(null);

  $effect(() => {
    rememberBoard(project, board);
  });

  const archivedQuery = createQuery(() => ({
    queryKey: queryKeys.archived(project, board),
    queryFn: () => boardApi.listArchived(project, board),
    enabled: !!project && !!board,
  }));

  const unarchiveMutation = createMutation(() => ({
    mutationFn: (task: Task) => tasksApi.unarchive(task.id),
    onSuccess: async (_data, task) => {
      await queryClient.invalidateQueries({
        queryKey: queryKeys.archived(project, board),
      });
      await queryClient.invalidateQueries({
        queryKey: ["snapshot", project, board],
      });
      showToast(`Task #${task.id} unarchived`);
    },
    onError: (err) => {
      showToast(err instanceof Error ? err.message : String(err), "error");
    },
  }));

  const deleteMutation = createMutation(() => ({
    mutationFn: (task: Task) => tasksApi.delete(task.id),
    onSuccess: async (_data, task) => {
      await queryClient.invalidateQueries({
        queryKey: queryKeys.archived(project, board),
      });
      await queryClient.invalidateQueries({
        queryKey: ["snapshot", project, board],
      });
      showToast(`Task #${task.id} deleted`);
    },
    onError: (err) => {
      showToast(err instanceof Error ? err.message : String(err), "error");
    },
  }));

  function fmt(v?: string | null) {
    if (!v) return "—";
    return String(v).slice(0, 10);
  }

  function handleDelete(task: Task) {
    pendingDelete = task;
  }

  async function confirmDelete() {
    const target = pendingDelete;
    pendingDelete = null;
    if (target) await deleteMutation.mutateAsync(target);
  }
</script>

<AppShell>
  <header
    class="border-b border-[var(--color-border-subtle)] bg-[var(--color-bg)] px-4 py-4 sm:px-6 sm:py-5"
  >
    <div
      class="flex flex-col gap-3.5 sm:flex-row sm:items-end sm:justify-between"
    >
      <div class="min-w-0">
        <nav
          class="mb-1.5 flex flex-wrap items-center gap-1.5 text-sm text-[var(--color-text-muted)]"
          aria-label="Breadcrumb"
        >
          <a
            href="/projects"
            class="transition hover:text-[var(--color-text-secondary)]"
            onclick={(e) => {
              e.preventDefault();
              navigate("/projects");
            }}
          >
            Projects
          </a>
          <span class="opacity-40">/</span>
          <a
            href={boardPath(project, board)}
            class="truncate transition hover:text-[var(--color-text-secondary)]"
            onclick={(e) => {
              e.preventDefault();
              navigate(boardPath(project, board));
            }}
          >
            {project}
          </a>
          <span class="opacity-40">/</span>
          <span class="text-[var(--color-text-secondary)]">{board}</span>
          <span class="opacity-40">/</span>
          <span class="text-[var(--color-text-secondary)]">archived</span>
        </nav>
        <div class="flex flex-wrap items-baseline gap-2 sm:gap-3">
          <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">
            Archived
          </h1>
          {#if archivedQuery.data}
            <span class="text-base text-[var(--color-text-muted)]">
              {archivedQuery.data.length} total
            </span>
          {/if}
        </div>
      </div>
      <Button
        variant="secondary"
        onclick={() => navigate(boardPath(project, board))}
      >
        ← Back to board
      </Button>
    </div>
  </header>

  <main class="min-h-0 flex-1 overflow-y-auto">
    {#if archivedQuery.isPending}
      <div class="p-8 text-base text-[var(--color-text-muted)]">Loading…</div>
    {:else if archivedQuery.isError}
      <div class="p-8 text-base text-[var(--color-danger)]">
        {archivedQuery.error.message}
      </div>
    {:else if !archivedQuery.data?.length}
      <EmptyState
        title="No archived tasks"
        description="Archive completed work from the board to keep it lean. Unarchive anytime to bring tasks back."
        actionLabel="Back to board"
        onaction={() => navigate(boardPath(project, board))}
      />
    {:else}
      <div class="space-y-3 p-4 sm:p-6">
        {#each archivedQuery.data as task (task.id)}
          <article
            class="rounded-2xl border border-[var(--color-border-subtle)] bg-[var(--color-bg-card)] p-4 sm:p-5"
          >
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div class="min-w-0">
                <div class="mb-2 flex flex-wrap items-center gap-2">
                  <Badge priority={task.priority} />
                  <span class="font-mono text-xs text-[var(--color-text-muted)]">
                    #{task.id}
                  </span>
                  <span class="text-xs text-[var(--color-text-muted)]">
                    {task.status?.replaceAll("_", " ") || "—"}
                  </span>
                </div>
                <h2 class="text-lg font-semibold tracking-tight">{task.title}</h2>
                {#if task.description}
                  <p class="mt-1 line-clamp-2 text-sm text-[var(--color-text-muted)]">
                    {task.description}
                  </p>
                {/if}
                <div
                  class="mt-3 flex flex-wrap gap-3 text-xs text-[var(--color-text-muted)]"
                >
                  <span>Completed {fmt(task.completed_at)}</span>
                  <span>Archived {fmt(task.archived_at)}</span>
                </div>
              </div>
              <div class="flex shrink-0 flex-wrap gap-2">
                <Button
                  variant="secondary"
                  size="sm"
                  onclick={() => void unarchiveMutation.mutateAsync(task)}
                  disabled={unarchiveMutation.isPending}
                >
                  Unarchive
                </Button>
                <Button
                  variant="danger"
                  size="sm"
                  onclick={() => handleDelete(task)}
                  disabled={deleteMutation.isPending}
                >
                  Delete
                </Button>
              </div>
            </div>
          </article>
        {/each}
      </div>
    {/if}
  </main>
</AppShell>

<ConfirmDialog
  open={pendingDelete !== null}
  title={pendingDelete ? `Permanently delete task #${pendingDelete.id}?` : ""}
  description={pendingDelete
    ? `“${pendingDelete.title}” will be removed from the archive. This cannot be undone.`
    : ""}
  confirmLabel="Delete permanently"
  onclose={() => (pendingDelete = null)}
  onconfirm={confirmDelete}
/>