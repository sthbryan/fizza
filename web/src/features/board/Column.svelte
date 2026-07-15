<script lang="ts">
  import type { ColumnSnapshot, Task } from "@/lib/api";
  import { cn } from "@/lib/cn";
  import TaskCard from "./TaskCard.svelte";

  type Accent = { border: string; dot: string };

  const ACCENTS: Accent[] = [
    { border: "var(--color-col-pink)", dot: "var(--color-col-todo)" },
    { border: "var(--color-col-peach)", dot: "var(--color-col-progress)" },
    { border: "var(--color-col-sky)", dot: "var(--color-col-review)" },
    { border: "var(--color-col-lilac)", dot: "var(--color-col-done)" },
    { border: "var(--color-col-mint)", dot: "var(--color-pri-urgent)" },
  ];

  function accentFor(name: string, index: number): Accent {
    const n = name.toLowerCase();
    if (n.includes("todo") || n.includes("to do") || n.includes("backlog")) {
      return ACCENTS[0];
    }
    if (n.includes("progress") || n.includes("doing") || n.includes("week")) {
      return ACCENTS[1];
    }
    if (n.includes("review")) return ACCENTS[2];
    if (n.includes("done") || n.includes("complete")) return ACCENTS[3];
    if (n.includes("blocked") || n.includes("hold")) return ACCENTS[4];
    return ACCENTS[index % ACCENTS.length];
  }

  interface Props {
    column: ColumnSnapshot;
    index: number;
    dragOver: boolean;
    draggingId: number | null;
    canDelete: boolean;
    terminal?: boolean;
    ondragstart: (task: Task) => void;
    ondragend: () => void;
    ondragover: (columnName: string) => void;
    ondragleave: () => void;
    ondrop: (columnName: string, beforeId?: string) => void;
    onedit: (task: Task) => void;
    ondelete: (task: Task) => void;
    onarchive: (task: Task) => void;
    onrestore: (task: Task) => void;
    onadd: (columnName: string) => void;
    ondeletecolumn: (column: ColumnSnapshot) => void;
  }

  let {
    column,
    index,
    dragOver,
    draggingId,
    canDelete,
    terminal = false,
    ondragstart,
    ondragend,
    ondragover,
    ondragleave,
    ondrop,
    onedit,
    ondelete,
    onarchive,
    onrestore,
    onadd,
    ondeletecolumn,
  }: Props = $props();

  const tasks = $derived(column.tasks || []);
  const accent = $derived(accentFor(column.name, index));
  const count = $derived(column.task_count ?? tasks.length);
  const countLabel = $derived(
    column.wip_limit != null ? `${count}/${column.wip_limit}` : String(count)
  );
</script>

<section
  class={cn(
    "flex w-[min(100%,380px)] shrink-0 flex-col rounded-2xl p-3 sm:w-[360px] sm:p-3.5",
    "max-h-[calc(100dvh-8.5rem)] sm:max-h-[calc(100vh-8rem)]",
    "border border-[var(--color-border-subtle)] border-t-2 bg-[var(--color-bg-card)]",
    dragOver && "ring-2 ring-[var(--color-accent)]/50"
  )}
  style:border-top-color={accent.border}
>
  <header class="mb-3 flex items-center justify-between gap-2 px-1">
    <div class="flex min-w-0 items-center gap-2">
      <span
        class="h-2 w-2 shrink-0 rounded-full"
        style:background={accent.dot}
      ></span>
      <h3
        class="truncate text-sm font-semibold capitalize tracking-tight text-[var(--color-text)]"
      >
        {column.name.replaceAll("_", " ")}
      </h3>
      <span class="text-xs font-medium tabular-nums text-[var(--color-text-muted)]">
        {countLabel}
      </span>
    </div>
    <div class="flex shrink-0 items-center gap-0.5">
      {#if canDelete}
        <button
          type="button"
          title="Delete column"
          onclick={() => ondeletecolumn(column)}
          class="flex h-7 w-7 cursor-pointer items-center justify-center rounded-lg text-[var(--color-text-muted)] transition hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-danger)]"
        >
          ×
        </button>
      {/if}
      <button
        type="button"
        title="Add task"
        onclick={() => onadd(column.name)}
        class="flex h-7 w-7 cursor-pointer items-center justify-center rounded-lg text-[var(--color-text-muted)] transition hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text)]"
      >
        +
      </button>
    </div>
  </header>

  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="flex flex-1 flex-col gap-3 overflow-y-auto"
    role="list"
    ondragover={(e) => {
      e.preventDefault();
      e.dataTransfer!.dropEffect = "move";
      ondragover(column.name);
    }}
    ondragleave={(e) => {
      if (!e.currentTarget.contains(e.relatedTarget as Node)) {
        ondragleave();
      }
    }}
    ondrop={(e) => {
      e.preventDefault();
      const cards = [
        ...e.currentTarget.querySelectorAll<HTMLElement>("[data-task-id]"),
      ];
      let beforeId: string | undefined;
      for (const card of cards) {
        const rect = card.getBoundingClientRect();
        if (e.clientY < rect.top + rect.height / 2) {
          beforeId = card.dataset.taskId;
          break;
        }
      }
      ondrop(column.name, beforeId);
    }}
  >
    {#if column.truncated && tasks.length === 0}
      <div
        class="flex min-h-20 items-center justify-center rounded-xl border border-dashed border-[var(--color-border)] bg-[var(--color-bg-soft)] px-3 text-center text-xs text-[var(--color-text-muted)]"
      >
        {count} completed · hidden for a lean board
      </div>
    {:else if tasks.length === 0}
      <button
        type="button"
        onclick={() => onadd(column.name)}
        class="flex min-h-20 cursor-pointer items-center justify-center rounded-xl border border-dashed border-[var(--color-border)] bg-[var(--color-bg-soft)] text-xs text-[var(--color-text-muted)] transition hover:border-[var(--color-accent)]/40 hover:text-[var(--color-text-secondary)]"
      >
        + Add task
      </button>
    {:else}
      {#each tasks as task (task.id)}
        <div data-task-id={task.id}>
          <TaskCard
            {task}
            dragging={draggingId === task.id}
            {terminal}
            {ondragstart}
            {ondragend}
            {onedit}
            {ondelete}
            {onarchive}
            {onrestore}
          />
        </div>
      {/each}
    {/if}
  </div>
</section>
