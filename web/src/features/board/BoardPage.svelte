<script lang="ts">
  import { createMutation, createQuery, useQueryClient } from "@tanstack/svelte-query";
  import type { ColumnSnapshot, Task } from "@/lib/api";
  import { queryKeys } from "@/lib/api";
  import { showToast } from "@/lib/toast/toast.svelte";
  import {
    boardPath,
    navigate,
    rememberBoard,
  } from "@/lib/router/router.svelte";
  import { cn } from "@/lib/cn";
  import AppShell from "@/shared/layout/AppShell.svelte";
  import Button from "@/shared/ui/Button.svelte";
  import Board from "./Board.svelte";
  import { boardApi } from "./api";
  import CreateBoardDialog from "./CreateBoardDialog.svelte";
  import CreateColumnDialog from "./CreateColumnDialog.svelte";
  import CreateProjectDialog from "@/features/projects/CreateProjectDialog.svelte";
  import CreateTaskDialog from "@/features/tasks/CreateTaskDialog.svelte";
  import EditTaskDialog from "@/features/tasks/EditTaskDialog.svelte";
  import { tasksApi } from "@/features/tasks/api";
  import { projectsApi } from "@/features/projects/api";
  import { onMount } from "svelte";

  interface Props {
    project: string;
    board: string;
  }

  let { project, board }: Props = $props();

  const queryClient = useQueryClient();

  let projectDialog = $state(false);
  let boardDialog = $state(false);
  let columnDialog = $state(false);
  let taskDialog = $state(false);
  let taskDefaultColumn = $state("");
  let editing = $state<Task | null>(null);

  let draggingId = $state<number | null>(null);
  let dragOverColumn = $state<string | null>(null);

  $effect(() => {
    rememberBoard(project, board);
  });

  const projectsQuery = createQuery({
    queryKey: queryKeys.projects,
    queryFn: () => projectsApi.list(),
  });

  const boardsQuery = createQuery({
    queryKey: queryKeys.boards(project),
    queryFn: () => boardApi.list(project),
    enabled: !!project,
  });

  const snapshotQuery = createQuery({
    queryKey: queryKeys.snapshot(project, board),
    queryFn: () => boardApi.snapshot(project, board),
    enabled: !!project && !!board,
  });

  const columnOptions = $derived(
    ($snapshotQuery.data?.columns || []).map((c) => ({
      value: c.name,
      label: c.name.replaceAll("_", " ").toUpperCase(),
    }))
  );

  const taskCount = $derived(
    ($snapshotQuery.data?.columns || []).reduce(
      (n, c) => n + (c.tasks?.length || 0),
      0
    )
  );

  const moveMutation = createMutation({
    mutationFn: (input: {
      taskId: number;
      column: string;
      beforeId?: string;
    }) =>
      tasksApi.move(input.taskId, {
        project,
        board,
        column: input.column,
        ...(input.beforeId ? { before: input.beforeId } : {}),
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: queryKeys.snapshot(project, board),
      });
    },
    onError: async (err) => {
      showToast(err instanceof Error ? err.message : String(err), "error");
      await queryClient.invalidateQueries({
        queryKey: queryKeys.snapshot(project, board),
      });
    },
    onSettled: () => {
      draggingId = null;
      dragOverColumn = null;
    },
  });

  const deleteMutation = createMutation({
    mutationFn: (task: Task) => tasksApi.delete(task.id),
    onSuccess: async (_data, task) => {
      await queryClient.invalidateQueries({
        queryKey: queryKeys.snapshot(project, board),
      });
      showToast(`Task #${task.id} deleted`);
    },
    onError: (err) => {
      showToast(err instanceof Error ? err.message : String(err), "error");
    },
  });

  const deleteColumnMutation = createMutation({
    mutationFn: (input: { name: string; force: boolean }) =>
      boardApi.deleteColumn(project, board, input.name, input.force),
    onSuccess: async (_data, input) => {
      await queryClient.invalidateQueries({
        queryKey: queryKeys.snapshot(project, board),
      });
      showToast(`Column “${input.name}” deleted`);
    },
    onError: (err) => {
      showToast(err instanceof Error ? err.message : String(err), "error");
    },
  });

  const deleteBoardMutation = createMutation({
    mutationFn: (name: string) => boardApi.delete(project, name),
    onSuccess: async (_data, name) => {
      showToast(`Board “${name}” deleted`);
      const remaining =
        (await queryClient.fetchQuery({
          queryKey: queryKeys.boards(project),
          queryFn: () => boardApi.list(project),
        })) || [];
      if (remaining.length === 0) {
        rememberBoard("", "");
        navigate("/projects");
        return;
      }
      if (name === board) {
        navigate(boardPath(project, remaining[0].name));
      }
    },
    onError: (err) => {
      showToast(err instanceof Error ? err.message : String(err), "error");
    },
  });

  const deleteProjectMutation = createMutation({
    mutationFn: () => projectsApi.delete(project),
    onSuccess: async () => {
      rememberBoard("", "");
      await queryClient.invalidateQueries({ queryKey: queryKeys.projects });
      showToast(`Project “${project}” deleted`);
      navigate("/projects");
    },
    onError: (err) => {
      showToast(err instanceof Error ? err.message : String(err), "error");
    },
  });

  function openTask(col?: string) {
    taskDefaultColumn = col || $snapshotQuery.data?.columns?.[0]?.name || "";
    taskDialog = true;
  }

  function handleDelete(task: Task) {
    if (!confirm(`Delete task #${task.id}: “${task.title}”?`)) return;
    void $deleteMutation.mutateAsync(task);
  }

  function handleDeleteColumn(col: ColumnSnapshot) {
    const n = col.tasks?.length || 0;
    const label = col.name.replaceAll("_", " ");
    if (n > 0) {
      if (
        !confirm(
          `Delete column “${label}” and its ${n} task(s)? This cannot be undone.`
        )
      ) {
        return;
      }
      void $deleteColumnMutation.mutateAsync({ name: col.name, force: true });
      return;
    }
    if (!confirm(`Delete empty column “${label}”?`)) return;
    void $deleteColumnMutation.mutateAsync({ name: col.name, force: false });
  }

  function handleDeleteBoard(name: string) {
    if (
      !confirm(
        `Delete board “${name}” and all of its columns and tasks? This cannot be undone.`
      )
    ) {
      return;
    }
    void $deleteBoardMutation.mutateAsync(name);
  }

  function handleDeleteProject() {
    if (
      !confirm(
        `Delete project “${project}” and all boards, columns, and tasks? This cannot be undone.`
      )
    ) {
      return;
    }
    void $deleteProjectMutation.mutateAsync();
  }

  function handleDrop(column: string, beforeId?: string) {
    if (draggingId == null) return;
    void $moveMutation.mutateAsync({
      taskId: draggingId,
      column,
      beforeId,
    });
  }

  onMount(() => {
    const onNew = () => openTask();
    window.addEventListener("fizza:new-task", onNew);
    return () => window.removeEventListener("fizza:new-task", onNew);
  });
</script>

<AppShell>
  <header
    class="border-b border-[var(--color-border-subtle)] bg-[var(--color-bg)]"
  >
    <div
      class="flex flex-col gap-3 px-3 pt-3 sm:flex-row sm:items-start sm:justify-between sm:px-5 sm:pt-4"
    >
      <div class="min-w-0">
        <nav
          class="mb-1 flex items-center gap-1.5 text-xs text-[var(--color-text-muted)]"
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
          <span class="truncate text-[var(--color-text-secondary)]"
            >{project}</span
          >
        </nav>
        <div class="flex flex-wrap items-baseline gap-2 sm:gap-3">
          <h1 class="truncate text-xl font-semibold tracking-tight sm:text-2xl">
            {project}
          </h1>
          {#if taskCount > 0}
            <span class="text-sm text-[var(--color-text-muted)]">
              {taskCount} tasks
            </span>
          {/if}
        </div>
      </div>

      <div class="flex flex-wrap gap-2">
        <Button
          variant="ghost"
          size="sm"
          onclick={handleDeleteProject}
          disabled={!project}
          class="text-[var(--color-text-muted)] hover:text-[var(--color-danger)]"
        >
          Delete project
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onclick={() => (columnDialog = true)}
          disabled={!project || !board}
        >
          + Column
        </Button>
        <Button
          variant="primary"
          size="sm"
          onclick={() => openTask()}
          disabled={!project || !board}
          class="flex-1 sm:flex-none"
        >
          + Task
        </Button>
      </div>
    </div>

    <!-- Board switcher: tabs, not selects -->
    <div
      class="mt-3 flex items-center gap-1 overflow-x-auto px-3 pb-0 sm:px-5"
      role="tablist"
      aria-label="Boards"
    >
      {#each $boardsQuery.data || [] as b (b.id)}
        {@const active = b.name === board}
        <div
          class={cn(
            "group/tab relative flex shrink-0 items-center border-b-2",
            active
              ? "border-[var(--color-accent)]"
              : "border-transparent"
          )}
        >
          <a
            href={boardPath(project, b.name)}
            role="tab"
            aria-selected={active}
            class={cn(
              "px-3 py-2.5 text-sm font-medium transition",
              active
                ? "text-[var(--color-text)]"
                : "text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"
            )}
            onclick={(e) => {
              e.preventDefault();
              navigate(boardPath(project, b.name));
            }}
          >
            {b.name}
          </a>
          <button
            type="button"
            title={`Delete board ${b.name}`}
            class={cn(
              "mr-1 flex h-5 w-5 cursor-pointer items-center justify-center rounded text-xs transition",
              "text-[var(--color-text-muted)] hover:bg-[var(--color-danger)]/15 hover:text-[var(--color-danger)]",
              active ? "opacity-100" : "opacity-0 group-hover/tab:opacity-100"
            )}
            onclick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              handleDeleteBoard(b.name);
            }}
          >
            ×
          </button>
        </div>
      {/each}
      <button
        type="button"
        title="New board"
        class="mb-0.5 ml-1 flex h-8 w-8 shrink-0 cursor-pointer items-center justify-center rounded-xl text-[var(--color-text-muted)] transition hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text)]"
        onclick={() => (boardDialog = true)}
      >
        +
      </button>
    </div>
  </header>

  <main class="min-h-0 flex-1 overflow-hidden">
    {#if $snapshotQuery.isPending && !$snapshotQuery.data}
      <div class="p-8 text-sm text-[var(--color-text-muted)]">Loading board…</div>
    {:else if $snapshotQuery.isError}
      <div class="p-8 text-sm text-[var(--color-danger)]">
        {$snapshotQuery.error.message}
      </div>
    {:else}
      <Board
        snapshot={$snapshotQuery.data ?? null}
        hasProjects={($projectsQuery.data?.length || 0) > 0}
        {dragOverColumn}
        {draggingId}
        ondragstart={(t) => (draggingId = t.id)}
        ondragend={() => {
          draggingId = null;
          dragOverColumn = null;
        }}
        ondragover={(col) => (dragOverColumn = col)}
        ondragleave={() => (dragOverColumn = null)}
        ondrop={handleDrop}
        onedit={(t) => (editing = t)}
        ondelete={handleDelete}
        onnewproject={() => (projectDialog = true)}
        onaddincolumn={(col) => openTask(col)}
        onaddcolumn={() => (columnDialog = true)}
        ondeletecolumn={handleDeleteColumn}
      />
    {/if}
  </main>
</AppShell>

<CreateProjectDialog open={projectDialog} onclose={() => (projectDialog = false)} />
<CreateBoardDialog
  open={boardDialog}
  {project}
  onclose={() => (boardDialog = false)}
/>
<CreateColumnDialog
  open={columnDialog}
  {project}
  {board}
  onclose={() => (columnDialog = false)}
/>
<CreateTaskDialog
  open={taskDialog}
  {project}
  {board}
  columns={columnOptions}
  defaultColumn={taskDefaultColumn}
  onclose={() => (taskDialog = false)}
/>
<EditTaskDialog
  open={editing !== null}
  {project}
  {board}
  task={editing}
  columns={columnOptions}
  onclose={() => (editing = null)}
/>
