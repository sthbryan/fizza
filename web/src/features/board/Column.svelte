<script lang="ts">
  import type { ColumnSnapshot, Task } from "@/lib/api";
  import { cn } from "@/lib/cn";
  import { animate } from "@/lib/animate";
  import TaskCard from "./TaskCard.svelte";

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
  const count = $derived(column.task_count ?? tasks.length);
  const countLabel = $derived(
    column.wip_limit != null ? `${count}/${column.wip_limit}` : String(count),
  );
  const indexLabel = $derived(String(index + 1).padStart(2, "0"));
  const overLimit = $derived(
    column.wip_limit != null && count > column.wip_limit,
  );

  let dropBeforeId = $state<number | null | undefined>(undefined);

  function handleDragOver(e: DragEvent) {
    const target = e.currentTarget as HTMLElement;
    const dt = e.dataTransfer;
    if (!dt) return;
    e.preventDefault();
    dt.dropEffect = "move";
    const cards = [...target.querySelectorAll<HTMLElement>("[data-task-id]")];
    let beforeId: number | null = null;
    for (const card of cards) {
      const rect = card.getBoundingClientRect();
      if (e.clientY < rect.top + rect.height / 2) {
        beforeId = Number(card.dataset.taskId);
        break;
      }
    }
    dropBeforeId = beforeId;
    ondragover(column.name);
  }

  function handleDragLeave(e: DragEvent) {
    const target = e.currentTarget as HTMLElement;
    if (!target.contains(e.relatedTarget as Node)) {
      dropBeforeId = undefined;
      ondragleave();
    }
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    const beforeId = dropBeforeId;
    dropBeforeId = undefined;
    ondrop(column.name, beforeId != null ? String(beforeId) : undefined);
  }
</script>

<section
  class={cn(
    "flex w-[min(88%,300px)] shrink-0 flex-col sm:w-[320px] md:w-[340px]",
    "max-h-[calc(100dvh-10rem)] sm:max-h-[calc(100dvh-9rem)]",
    "transition-colors duration-150",
    dragOver && "bg-neutral-950/80",
  )}
>
  <header class="mb-4 flex items-center justify-between gap-2 border-b border-neutral-800 pb-3">
    <div class="flex min-w-0 items-baseline gap-2">
      <span
        class="font-mono text-[11px] uppercase tracking-[0.08em] text-neutral-500"
      >
        {indexLabel}
      </span>
      <span
        class="truncate text-base tracking-tight text-neutral-100"
      >
        {column.name.replaceAll("_", " ")}
      </span>
      <span
        class="font-mono text-[11px] tabular-nums"
        class:text-[var(--color-accent)]={overLimit}
        class:text-neutral-500={!overLimit}
      >
        {countLabel}
      </span>
    </div>
    <div class="flex shrink-0 items-center">
      {#if canDelete}
        <button
          type="button"
          title="Delete column"
          onclick={() => ondeletecolumn(column)}
          class="flex h-11 w-11 cursor-pointer items-center justify-center font-mono text-base text-neutral-500 transition-colors hover:text-[var(--color-accent)]"
        >
          ×
        </button>
      {/if}
      <button
        type="button"
        title="Add task"
        onclick={() => onadd(column.name)}
        class="flex h-11 w-11 cursor-pointer items-center justify-center font-mono text-base text-neutral-500 transition-colors hover:text-white"
      >
        +
      </button>
    </div>
  </header>

  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="flex flex-1 flex-col gap-2 overflow-y-auto"
    role="list"
    ondragover={handleDragOver}
    ondragleave={handleDragLeave}
    ondrop={handleDrop}
    use:animate={{ duration: 180, easing: "ease-out" }}
  >
    {#if tasks.length === 0 && dropBeforeId === undefined}
      <button
        type="button"
        onclick={() => onadd(column.name)}
        class="flex min-h-16 cursor-pointer items-center justify-center border border-dashed border-neutral-800 bg-transparent text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-500 transition-colors hover:border-neutral-500 hover:text-neutral-300"
      >
        + Add task
      </button>
    {:else}
      {#each tasks as task (task.id)}
        <div data-task-id={task.id}>
          <TaskCard
            {task}
            {draggingId}
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
