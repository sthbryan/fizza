<script lang="ts">
  import type { BoardSnapshot, ColumnSnapshot, Task } from "@/lib/api";
  import { cn } from "@/lib/cn";
  import EmptyState from "@/shared/ui/EmptyState.svelte";
  import Column from "./Column.svelte";
  import { isTerminalColumn } from "./terminal";

  interface Props {
    snapshot: BoardSnapshot | null;
    hasProjects: boolean;
    showCompleted: boolean;
    dragOverColumn: string | null;
    draggingId: number | null;
    ondragstart: (task: Task) => void;
    ondragend: () => void;
    ondragover: (columnName: string) => void;
    ondragleave: () => void;
    ondrop: (columnName: string, beforeId?: string) => void;
    onmove: (task: Task, columnName: string) => void;
    onedit: (task: Task) => void;
    ondelete: (task: Task) => void;
    onarchive: (task: Task) => void;
    onrestore: (task: Task) => void;
    onshowcompleted: () => void;
    onnewproject: () => void;
    onaddincolumn: (columnName: string) => void;
    onaddcolumn: () => void;
    ondeletecolumn: (column: ColumnSnapshot) => void;
  }

  let {
    snapshot,
    hasProjects,
    showCompleted,
    dragOverColumn,
    draggingId,
    ondragstart,
    ondragend,
    ondragover,
    ondragleave,
    ondrop,
    onmove,
    onedit,
    ondelete,
    onarchive,
    onrestore,
    onshowcompleted,
    onnewproject,
    onaddincolumn,
    onaddcolumn,
    ondeletecolumn,
  }: Props = $props();

  const canDeleteColumn = $derived((snapshot?.columns?.length || 0) > 1);
  const columnNames = $derived((snapshot?.columns || []).map((c) => c.name));
  const visibleColumns = $derived(
    (snapshot?.columns || []).filter(
      (c) => showCompleted || !isTerminalColumn(c.name)
    )
  );
  const hiddenDone = $derived(
    (snapshot?.columns || []).filter(
      (c) => !showCompleted && isTerminalColumn(c.name)
    )
  );
  const hiddenDoneCount = $derived(
    hiddenDone.reduce((n, c) => n + (c.task_count ?? 0), 0)
  );

  let scroller = $state<HTMLDivElement | null>(null);
  let activeIndex = $state(0);

  function columnElements(): HTMLElement[] {
    return scroller ? [...scroller.querySelectorAll<HTMLElement>("section")] : [];
  }

  function syncActive() {
    if (!scroller) return;
    const centre = scroller.scrollLeft + scroller.clientWidth / 2;
    let nearest = 0;
    let best = Number.POSITIVE_INFINITY;
    columnElements().forEach((el, i) => {
      const mid = el.offsetLeft + el.offsetWidth / 2;
      const gap = Math.abs(mid - centre);
      if (gap < best) {
        best = gap;
        nearest = i;
      }
    });
    activeIndex = nearest;
  }

  function goTo(index: number) {
    const el = columnElements()[index];
    el?.scrollIntoView({ behavior: "smooth", inline: "center", block: "nearest" });
  }
</script>

{#if !hasProjects}
  <EmptyState
    title="No projects yet"
    description="Create a project to seed a board with todo, in progress, in review, and done."
    actionLabel="Create project"
    onaction={onnewproject}
  />
{:else if !snapshot?.columns?.length}
  <EmptyState
    title="Select a board"
    description="Choose a project and board above, or create a new board."
  />
{:else}
  <div class="flex h-full flex-col">
    {#if visibleColumns.length > 1}
    <nav
      aria-label="Columns"
      class="flex gap-1 overflow-x-auto px-4 pt-3 sm:hidden"
    >
      {#each visibleColumns as column, index (column.id)}
        <button
          type="button"
          aria-current={activeIndex === index}
          onclick={() => goTo(index)}
          class={cn(
            "flex shrink-0 items-center gap-1.5 border-b-2 px-2.5 py-2 text-label font-mono uppercase transition-colors",
            activeIndex === index
              ? "border-white text-neutral-100"
              : "border-transparent text-neutral-500"
          )}
        >
          <span>{column.name.replaceAll("_", " ")}</span>
          <span class="tabular-nums text-neutral-500">
            {column.task_count ?? column.tasks?.length ?? 0}
          </span>
        </button>
      {/each}
    </nav>
  {/if}
  <div
    bind:this={scroller}
    onscroll={syncActive}
    class="flex min-h-0 flex-1 snap-x snap-mandatory gap-3 overflow-x-auto overflow-y-hidden px-4 pb-4 pt-4 sm:snap-none sm:gap-3.5 sm:px-5 sm:pb-6"
  >
    {#each visibleColumns as column, index (column.id)}
      <Column
        {column}
        {index}
        dragOver={dragOverColumn === column.name}
        {draggingId}
        canDelete={canDeleteColumn}
        terminal={isTerminalColumn(column.name)}
        {ondragstart}
        {ondragend}
        {ondragover}
        {ondragleave}
        {ondrop}
        {onmove}
        {columnNames}
        {onedit}
        {ondelete}
        {onarchive}
        {onrestore}
        onadd={onaddincolumn}
        {ondeletecolumn}
      />
    {/each}
    {#if !showCompleted && hiddenDone.length > 0}
      <button
        type="button"
        onclick={onshowcompleted}
        class="flex h-full min-h-32 w-full shrink-0 snap-center cursor-pointer flex-col items-center justify-center gap-2 border border-dashed border-neutral-800 bg-transparent px-4 text-center transition-colors hover:border-neutral-500 hover:text-neutral-300 sm:w-80"
      >
        <span class="font-mono text-sm tabular-nums text-white">
          {hiddenDoneCount}
        </span>
        <span class="text-label font-mono uppercase text-neutral-500">
          Done · show
        </span>
      </button>
    {/if}
    <button
      type="button"
      onclick={onaddcolumn}
      class="flex h-full min-h-32 w-full shrink-0 snap-center cursor-pointer flex-col items-center justify-center gap-2 border border-dashed border-neutral-800 bg-transparent text-label font-mono uppercase text-neutral-500 transition-colors hover:border-neutral-500 hover:text-neutral-300 sm:w-80"
    >
      <span class="text-base">+</span>
      Add column
    </button>
    </div>
  </div>
{/if}