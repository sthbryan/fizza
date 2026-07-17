<script lang="ts">
  import { createMutation, createQuery, useQueryClient } from "@tanstack/svelte-query";
  import type { ColumnSnapshot, Task } from "@/lib/api";
  import { queryKeys } from "@/lib/api";
  import { showStatus } from "@/lib/status/status.svelte";
  import {
    archivedPath,
    boardPath,
    navigate,
    rememberBoard,
  } from "@/lib/router/router.svelte";
  import { cn } from "@/lib/cn";
  import AppShell from "@/shared/layout/AppShell.svelte";
  import Button from "@/shared/ui/Button.svelte";
  import ConfirmDialog from "@/shared/ui/ConfirmDialog.svelte";
  import Plus from "lucide-svelte/icons/plus";
  import Board from "./Board.svelte";
  import { boardApi } from "./api";
  import { isTerminalColumn } from "./terminal";
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
  const SHOW_DONE_KEY = "fizza.showCompleted";

  let projectDialog = $state(false);
  let boardDialog = $state(false);
  let columnDialog = $state(false);
  let taskDialog = $state(false);
  let taskDefaultColumn = $state("");
  let editing = $state<Task | null>(null);
  let showCompleted = $state(
    typeof localStorage !== "undefined" &&
      localStorage.getItem(SHOW_DONE_KEY) === "1"
  );

  let draggingId = $state<number | null>(null);
  let dragOverColumn = $state<string | null>(null);

  type ConfirmSpec = {
    title: string;
    description: string;
    confirmLabel: string;
    cancelLabel?: string;
    run: () => void | Promise<void>;
  };

  let pendingConfirm = $state<ConfirmSpec | null>(null);

  function ask(spec: ConfirmSpec) {
    pendingConfirm = spec;
  }

  async function runConfirm() {
    const spec = pendingConfirm;
    pendingConfirm = null;
    if (spec) await spec.run();
  }

  $effect(() => {
    rememberBoard(project, board);
  });

  $effect(() => {
    localStorage.setItem(SHOW_DONE_KEY, showCompleted ? "1" : "0");
  });

  const projectsQuery = createQuery(() => ({
    queryKey: queryKeys.projects,
    queryFn: () => projectsApi.list(),
  }));

  const boardsQuery = createQuery(() => ({
    queryKey: queryKeys.boards(project),
    queryFn: () => boardApi.list(project),
    enabled: !!project,
  }));

  const snapshotQuery = createQuery(() => ({
    queryKey: queryKeys.snapshot(project, board, showCompleted),
    queryFn: () => boardApi.snapshot(project, board, showCompleted),
    enabled: !!project && !!board,
  }));

  const columnOptions = $derived(
    (snapshotQuery.data?.columns || []).map((c) => ({
      value: c.name,
      label: c.name.replaceAll("_", " ").toUpperCase(),
    }))
  );

  const openColumn = $derived(
    (snapshotQuery.data?.columns || []).find((c) => !isTerminalColumn(c.name))
      ?.name || "todo"
  );

  const taskCount = $derived(
    (snapshotQuery.data?.columns || []).reduce(
      (n, c) => n + (c.task_count ?? c.tasks?.length ?? 0),
      0
    )
  );

  const doneCount = $derived(
    (snapshotQuery.data?.columns || [])
      .filter((c) => isTerminalColumn(c.name))
      .reduce((n, c) => n + (c.task_count ?? c.tasks?.length ?? 0), 0)
  );

  const archivedCount = $derived(snapshotQuery.data?.archived_count ?? 0);

  async function invalidateBoard() {
    await queryClient.invalidateQueries({
      queryKey: ["snapshot", project, board],
    });
    await queryClient.invalidateQueries({
      queryKey: queryKeys.archived(project, board),
    });
  }

  const moveMutation = createMutation(() => ({
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
      await invalidateBoard();
    },
    onError: async (err) => {
      showStatus(err instanceof Error ? err.message : String(err), "error");
      await invalidateBoard();
    },
    onSettled: () => {
      draggingId = null;
      dragOverColumn = null;
    },
  }));

  const deleteMutation = createMutation(() => ({
    mutationFn: (task: Task) => tasksApi.delete(task.id),
    onSuccess: async (_data, task) => {
      await invalidateBoard();
      showStatus(`Task #${task.id} deleted`);
    },
    onError: (err) => {
      showStatus(err instanceof Error ? err.message : String(err), "error");
    },
  }));

  const archiveMutation = createMutation(() => ({
    mutationFn: (task: Task) => tasksApi.archive(task.id),
    onSuccess: async (_data, task) => {
      await invalidateBoard();
      showStatus(`Task #${task.id} archived`);
    },
    onError: (err) => {
      showStatus(err instanceof Error ? err.message : String(err), "error");
    },
  }));

  const restoreMutation = createMutation(() => ({
    mutationFn: (task: Task) =>
      tasksApi.move(task.id, { project, board, column: openColumn }),
    onSuccess: async (_data, task) => {
      await invalidateBoard();
      showStatus(`Task #${task.id} restored to ${openColumn}`);
    },
    onError: (err) => {
      showStatus(err instanceof Error ? err.message : String(err), "error");
    },
  }));

  const archiveDoneMutation = createMutation(() => ({
    mutationFn: () => boardApi.archiveDone(project, board),
    onSuccess: async (data) => {
      await invalidateBoard();
      showStatus(
        data.archived
          ? `Archived ${data.archived} completed task${data.archived === 1 ? "" : "s"}`
          : "No completed tasks to archive"
      );
    },
    onError: (err) => {
      showStatus(err instanceof Error ? err.message : String(err), "error");
    },
  }));

  const deleteColumnMutation = createMutation(() => ({
    mutationFn: (input: { name: string; force: boolean }) =>
      boardApi.deleteColumn(project, board, input.name, input.force),
    onSuccess: async (_data, input) => {
      await invalidateBoard();
      showStatus(`Column “${input.name}” deleted`);
    },
    onError: (err) => {
      showStatus(err instanceof Error ? err.message : String(err), "error");
    },
  }));

  const deleteBoardMutation = createMutation(() => ({
    mutationFn: (name: string) => boardApi.delete(project, name),
    onSuccess: async (_data, name) => {
      showStatus(`Board “${name}” deleted`);
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
      showStatus(err instanceof Error ? err.message : String(err), "error");
    },
  }));

  const deleteProjectMutation = createMutation(() => ({
    mutationFn: () => projectsApi.delete(project),
    onSuccess: async () => {
      rememberBoard("", "");
      await queryClient.invalidateQueries({ queryKey: queryKeys.projects });
      showStatus(`Project “${project}” deleted`);
      navigate("/projects");
    },
    onError: (err) => {
      showStatus(err instanceof Error ? err.message : String(err), "error");
    },
  }));

  function openTask(col?: string) {
    taskDefaultColumn = col || snapshotQuery.data?.columns?.[0]?.name || "";
    taskDialog = true;
  }

  function handleDelete(task: Task) {
    ask({
      title: `Delete task #${task.id}?`,
      description: `“${task.title}” will be permanently removed. This cannot be undone.`,
      confirmLabel: "Delete task",
      run: async () => {
        await deleteMutation.mutateAsync(task);
      },
    });
  }

  function handleArchive(task: Task) {
    void archiveMutation.mutateAsync(task);
  }

  function handleRestore(task: Task) {
    void restoreMutation.mutateAsync(task);
  }

  function handleArchiveDone() {
    if (doneCount <= 0) return;
    ask({
      title: `Archive ${doneCount} completed task${doneCount === 1 ? "" : "s"}?`,
      description: `They will move out of the board and stay recoverable from the Archived view.`,
      confirmLabel: "Archive all",
      run: async () => {
        await archiveDoneMutation.mutateAsync();
      },
    });
  }

  function handleDeleteColumn(col: ColumnSnapshot) {
    const n = col.task_count ?? col.tasks?.length ?? 0;
    const label = col.name.replaceAll("_", " ");
    if (n > 0) {
      ask({
        title: `Delete column “${label}”?`,
        description: `${n} task${n === 1 ? "" : "s"} in this column will also be permanently deleted. This cannot be undone.`,
        confirmLabel: "Delete column",
        run: async () => {
          await deleteColumnMutation.mutateAsync({
            name: col.name,
            force: true,
          });
        },
      });
      return;
    }
    ask({
      title: `Delete empty column “${label}”?`,
      description: `This column has no tasks and can be safely removed.`,
      confirmLabel: "Delete column",
      run: async () => {
        await deleteColumnMutation.mutateAsync({
          name: col.name,
          force: false,
        });
      },
    });
  }

  function handleDeleteBoard(name: string) {
    ask({
      title: `Delete board “${name}”?`,
      description: `All columns and tasks on this board will be permanently deleted. This cannot be undone.`,
      confirmLabel: "Delete board",
      run: async () => {
        await deleteBoardMutation.mutateAsync(name);
      },
    });
  }

  function handleDeleteProject() {
    ask({
      title: `Delete project “${project}”?`,
      description: `All boards, columns, and tasks in this project will be permanently deleted. This cannot be undone.`,
      confirmLabel: "Delete project",
      run: async () => {
        await deleteProjectMutation.mutateAsync();
      },
    });
  }

  function handleDrop(column: string, beforeId?: string) {
    if (draggingId == null) return;
    void moveMutation.mutateAsync({
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
    class="border-b border-neutral-800 bg-black"
  >
    <div
      class="flex flex-col gap-3.5 px-4 pt-4 sm:flex-row sm:items-start sm:justify-between sm:px-6 sm:pt-5"
    >
      <div class="min-w-0">
        <nav
          class="mb-1.5 flex items-center gap-1.5 text-[10px] font-mono uppercase tracking-[0.1em] text-neutral-500"
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
          <span class="truncate text-neutral-400"
            >{project}</span
          >
        </nav>
        <div class="flex flex-wrap items-baseline gap-2 sm:gap-3">
          <h1 class="truncate text-2xl font-semibold tracking-tight sm:text-3xl text-white">
            {project}
          </h1>
          {#if taskCount > 0}
            <span class="font-mono text-sm tabular-nums text-neutral-500">
              {taskCount} tasks
            </span>
          {/if}
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button
          variant="ghost"
          onclick={() => (showCompleted = !showCompleted)}
          disabled={!project || !board}
          class="!hidden sm:!inline-flex"
        >
          {showCompleted ? "Hide completed" : "Show completed"}
          {#if doneCount > 0}
            <span class="opacity-70">({doneCount})</span>
          {/if}
        </Button>
        <Button
          variant="ghost"
          onclick={() => navigate(archivedPath(project, board))}
          disabled={!project || !board}
        >
          Archived
          {#if archivedCount > 0}
            <span class="opacity-70">({archivedCount})</span>
          {/if}
        </Button>
        {#if showCompleted && doneCount > 0}
          <Button
            variant="ghost"
            onclick={handleArchiveDone}
            disabled={archiveDoneMutation.isPending}
            class="!hidden sm:!inline-flex"
          >
            Archive all done
          </Button>
        {/if}
        <Button
          variant="ghost"
          onclick={handleDeleteProject}
          disabled={!project}
          class="!hidden hover:text-red-500 sm:!inline-flex"
        >
          Delete project
        </Button>
        <Button
          variant="ghost"
          onclick={() => (columnDialog = true)}
          disabled={!project || !board}
          class="!hidden sm:!inline-flex"
        >
          + Column
        </Button>
        <Button
          variant="primary"
          onclick={() => openTask()}
          disabled={!project || !board}
          title="New task"
          aria-label="New task"
          class="size-9 shrink-0 p-0"
        >
          <Plus size={16} strokeWidth={1.5} />
        </Button>
      </div>
    </div>

    <!-- Board switcher: tabs, not selects -->
    <div
      class="mt-3.5 flex items-center gap-1 overflow-x-auto px-4 pb-0 sm:px-6"
      role="tablist"
      aria-label="Boards"
    >
      {#each boardsQuery.data || [] as b (b.id)}
        {@const active = b.name === board}
        <div
          class={cn(
            "group/tab relative flex shrink-0 items-center border-b-2",
            active
              ? "border-white"
              : "border-transparent"
          )}
        >
          <a
            href={boardPath(project, b.name)}
            role="tab"
            aria-selected={active}
            class={cn(
              "px-3.5 py-3 text-sm font-medium transition-colors",
              active
                ? "text-white"
                : "text-neutral-500 hover:text-neutral-300"
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
              "mr-1.5 flex h-6 w-6 cursor-pointer items-center justify-center rounded text-sm font-mono transition-colors",
              "text-neutral-500 hover:bg-red-500/15 hover:text-red-500",
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
        class="mb-0.5 ml-1 flex h-9 w-9 shrink-0 cursor-pointer items-center justify-center rounded-md text-lg font-mono text-neutral-500 transition-colors hover:bg-neutral-900 hover:text-white"
        onclick={() => (boardDialog = true)}
      >
        +
      </button>
    </div>
  </header>

  <main class="min-h-0 flex-1 overflow-hidden">
    {#if snapshotQuery.isPending && !snapshotQuery.data}
      <div class="p-8 text-[10px] font-mono uppercase tracking-[0.1em] text-neutral-500">Loading board…</div>
    {:else if snapshotQuery.isError}
      <div class="p-8 text-sm font-mono text-red-500">
        {snapshotQuery.error.message}
      </div>
    {:else}
      <Board
        snapshot={snapshotQuery.data ?? null}
        hasProjects={(projectsQuery.data?.length || 0) > 0}
        {showCompleted}
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
        onarchive={handleArchive}
        onrestore={handleRestore}
        onshowcompleted={() => (showCompleted = true)}
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
<ConfirmDialog
  open={pendingConfirm !== null}
  title={pendingConfirm?.title ?? ""}
  description={pendingConfirm?.description ?? ""}
  confirmLabel={pendingConfirm?.confirmLabel}
  cancelLabel={pendingConfirm?.cancelLabel}
  onclose={() => (pendingConfirm = null)}
  onconfirm={runConfirm}
/>
