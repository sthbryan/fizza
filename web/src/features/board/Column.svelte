<script lang="ts">
  import type { ColumnSnapshot, Task } from "@/lib/api";
  import { cn } from "@/lib/cn";
  import TaskCard from "./TaskCard.svelte";

  type Theme = { shell: string; dot: string; title: string };

  const PALETTE: Theme[] = [
    {
      shell: "bg-[var(--color-col-pink)]/90",
      dot: "bg-[#f9a8d4]",
      title: "text-[#fbcfe8]",
    },
    {
      shell: "bg-[var(--color-col-peach)]/90",
      dot: "bg-[#fdba74]",
      title: "text-[#fed7aa]",
    },
    {
      shell: "bg-[var(--color-col-sky)]/90",
      dot: "bg-[#7dd3fc]",
      title: "text-[#bae6fd]",
    },
    {
      shell: "bg-[var(--color-col-lilac)]/90",
      dot: "bg-[#c4b5fd]",
      title: "text-[#ddd6fe]",
    },
    {
      shell: "bg-[var(--color-col-mint)]/90",
      dot: "bg-[#6ee7b7]",
      title: "text-[#a7f3d0]",
    },
  ];

  function themeFor(name: string, index: number): Theme {
    const n = name.toLowerCase();
    if (n.includes("todo") || n.includes("to do") || n.includes("backlog")) {
      return PALETTE[0];
    }
    if (n.includes("progress") || n.includes("doing") || n.includes("week")) {
      return PALETTE[1];
    }
    if (n.includes("review")) return PALETTE[2];
    if (n.includes("blocked") || n.includes("hold")) return PALETTE[4];
    if (n.includes("done") || n.includes("complete")) return PALETTE[3];
    return PALETTE[index % PALETTE.length];
  }

  interface Props {
    column: ColumnSnapshot;
    index: number;
    dragOver: boolean;
    draggingId: number | null;
    canDelete: boolean;
    ondragstart: (task: Task) => void;
    ondragend: () => void;
    ondragover: (columnName: string) => void;
    ondragleave: () => void;
    ondrop: (columnName: string, beforeId?: string) => void;
    onedit: (task: Task) => void;
    ondelete: (task: Task) => void;
    onadd: (columnName: string) => void;
    ondeletecolumn: (column: ColumnSnapshot) => void;
  }

  let {
    column,
    index,
    dragOver,
    draggingId,
    canDelete,
    ondragstart,
    ondragend,
    ondragover,
    ondragleave,
    ondrop,
    onedit,
    ondelete,
    onadd,
    ondeletecolumn,
  }: Props = $props();

  const tasks = $derived(column.tasks || []);
  const theme = $derived(themeFor(column.name, index));
  const countLabel = $derived(
    column.wip_limit != null
      ? `${tasks.length}/${column.wip_limit}`
      : String(tasks.length)
  );
</script>

<section
  class={cn(
    "flex w-[min(100%,300px)] shrink-0 flex-col rounded-[1.75rem] p-3 sm:w-[280px] sm:p-3.5",
    "max-h-[calc(100dvh-8.5rem)] sm:max-h-[calc(100vh-8rem)]",
    theme.shell,
    dragOver && "ring-2 ring-[var(--color-accent)]/50"
  )}
>
  <header class="mb-3 flex items-center justify-between gap-2 px-1">
    <div class="flex min-w-0 items-center gap-2">
      <span class={cn("h-2 w-2 shrink-0 rounded-full", theme.dot)}></span>
      <h3
        class={cn(
          "truncate text-sm font-semibold capitalize tracking-tight",
          theme.title
        )}
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
          class="flex h-7 w-7 cursor-pointer items-center justify-center rounded-lg text-white/40 transition hover:bg-black/20 hover:text-[var(--color-danger)]"
        >
          ×
        </button>
      {/if}
      <button
        type="button"
        title="Add task"
        onclick={() => onadd(column.name)}
        class="flex h-7 w-7 cursor-pointer items-center justify-center rounded-lg text-white/50 transition hover:bg-black/20 hover:text-white"
      >
        +
      </button>
    </div>
  </header>

  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="flex flex-1 flex-col gap-2.5 overflow-y-auto"
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
    {#if tasks.length === 0}
      <button
        type="button"
        onclick={() => onadd(column.name)}
        class="flex min-h-20 cursor-pointer items-center justify-center rounded-2xl border border-dashed border-white/15 bg-black/10 text-xs text-white/45 transition hover:border-white/25 hover:text-white/70"
      >
        + Add task
      </button>
    {:else}
      {#each tasks as task (task.id)}
        <div data-task-id={task.id}>
          <TaskCard
            {task}
            dragging={draggingId === task.id}
            {ondragstart}
            {ondragend}
            {onedit}
            {ondelete}
          />
        </div>
      {/each}
    {/if}
  </div>
</section>
