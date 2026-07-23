<script lang="ts">
  import { createMutation, createQuery, useQueryClient } from "@tanstack/svelte-query";
  import AppShell from "@/shared/layout/AppShell.svelte";
  import Button from "@/shared/ui/Button.svelte";
  import EmptyState from "@/shared/ui/EmptyState.svelte";
  import { queryKeys, type Task } from "@/lib/api";
  import {
    boardPath,
    navigate,
    rememberBoard,
  } from "@/lib/router/router.svelte";
  import { showStatus } from "@/lib/status/status.svelte";
  import { boardApi } from "./api";
  import { tasksApi } from "@/features/tasks/api";
  import ConfirmDialog from "@/shared/ui/ConfirmDialog.svelte";
  import { animate } from "@/lib/animate";
  import { cn } from "@/lib/cn";

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
      showStatus(`Task #${task.id} unarchived`);
    },
    onError: (err) => {
      showStatus(err instanceof Error ? err.message : String(err), "error");
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
      showStatus(`Task #${task.id} deleted`);
    },
    onError: (err) => {
      showStatus(err instanceof Error ? err.message : String(err), "error");
    },
  }));

  function fmt(v?: string | null) {
    if (!v) return "-";
    return String(v).slice(0, 10);
  }

  function priorityClass(p?: string | null): string {
    const k = String(p || "medium").toLowerCase();
    if (k === "urgent") return "text-accent";
    if (k === "high") return "text-neutral-200";
    if (k === "low") return "text-neutral-600";
    return "text-neutral-400";
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
    class="border-b border-neutral-800 bg-black px-4 py-4 sm:px-6 sm:py-5"
  >
    <div
      class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between"
    >
      <div class="min-w-0">
        <nav
          class="mb-2 flex flex-wrap items-center gap-1.5 text-label font-mono uppercase text-neutral-500"
          aria-label="Breadcrumb"
        >
          <a
            href="/projects"
            class="transition-colors hover:text-neutral-300"
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
            class="truncate transition-colors hover:text-neutral-300"
            onclick={(e) => {
              e.preventDefault();
              navigate(boardPath(project, board));
            }}
          >
            {project}
          </a>
          <span class="opacity-40">/</span>
          <span class="text-neutral-400">{board}</span>
          <span class="opacity-40">/</span>
          <span class="text-neutral-400">archived</span>
        </nav>
        <div class="flex flex-wrap items-baseline gap-2 sm:gap-3">
          <h1 class="text-lg tracking-tight text-white">
            Archived
          </h1>
          {#if archivedQuery.data}
            <span class="font-mono text-label tabular-nums text-neutral-500">
              {archivedQuery.data.length} total
            </span>
          {/if}
        </div>
      </div>
      <Button
        variant="secondary"
        onclick={() => navigate(boardPath(project, board))}
      >
        Back to board
      </Button>
    </div>
  </header>

  <main class="min-h-0 flex-1 overflow-y-auto">
    {#if archivedQuery.isPending}
      <div class="p-8 text-label font-mono uppercase text-neutral-500">[LOADING]</div>
    {:else if archivedQuery.isError}
      <div class="p-8 text-label font-mono uppercase text-accent">
        [ERROR] {archivedQuery.error.message}
      </div>
    {:else if !archivedQuery.data?.length}
      <EmptyState
        title="No archived tasks"
        description="Archive completed work from the board to keep it lean. Unarchive anytime to bring tasks back."
        actionLabel="Back to board"
        onaction={() => navigate(boardPath(project, board))}
      />
    {:else}
      <div class="h-full overflow-y-auto px-4 py-5 sm:px-6 sm:py-6">
        <div
          class="divide-y divide-neutral-800 border-y border-neutral-800"
          use:animate={{ duration: 180, easing: "ease-out" }}
        >
          {#each archivedQuery.data as task (task.id)}
            <div
              class="group flex min-h-14 flex-wrap items-center gap-2 py-3 sm:flex-nowrap sm:gap-3"
            >
              <span
                class={cn(
                  "w-16 shrink-0 text-label font-mono uppercase",
                  priorityClass(task.priority)
                )}
              >
                {String(task.priority || "medium").toLowerCase()}
              </span>
              <span
                class="hidden w-24 shrink-0 truncate text-label font-mono uppercase text-neutral-500 md:inline"
                title={task.status}
              >
                {task.status?.replaceAll("_", " ") || "-"}
              </span>
              <span class="min-w-0 flex-1 truncate text-sm text-neutral-200">
                {task.title}
              </span>
              <span
                class="hidden shrink-0 text-label font-mono uppercase text-neutral-500 sm:inline"
                title={`Archived ${fmt(task.archived_at)}`}
              >
                {fmt(task.archived_at)}
              </span>
              <span class="shrink-0 text-label font-mono uppercase text-neutral-600">
                #{task.id}
              </span>
              <div class="flex w-full shrink-0 items-center sm:w-auto">
                <button
                  type="button"
                  class="flex min-h-11 cursor-pointer items-center px-3 text-label font-mono uppercase text-neutral-500 transition-colors hover:text-neutral-200 disabled:cursor-not-allowed disabled:opacity-30"
                  disabled={unarchiveMutation.isPending}
                  onclick={() => void unarchiveMutation.mutateAsync(task)}
                >
                  Unarchive
                </button>
                <button
                  type="button"
                  class="flex min-h-11 cursor-pointer items-center px-3 text-label font-mono uppercase text-neutral-500 transition-colors hover:text-accent disabled:cursor-not-allowed disabled:opacity-30"
                  disabled={deleteMutation.isPending}
                  onclick={() => handleDelete(task)}
                >
                  Del
                </button>
              </div>
            </div>
          {/each}
        </div>
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