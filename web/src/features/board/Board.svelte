<script lang="ts">
  import type { BoardSnapshot, ColumnSnapshot, Task } from "@/lib/api";
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
  <div
    class="flex h-full snap-x snap-mandatory gap-3 overflow-x-auto overflow-y-hidden px-4 pb-4 pt-4 sm:snap-none sm:gap-3.5 sm:px-5 sm:pb-6"
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
{/if}