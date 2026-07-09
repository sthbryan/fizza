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
    class="flex h-full gap-3 overflow-x-auto overflow-y-hidden px-3 pb-4 pt-3 sm:gap-4 sm:px-5 sm:pb-6 sm:pt-4"
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
        class="flex h-28 w-[min(100%,200px)] shrink-0 cursor-pointer flex-col items-center justify-center gap-1.5 rounded-[1.75rem] border border-dashed border-[var(--color-border)] bg-[var(--color-bg-card)]/40 px-4 text-center text-sm text-[var(--color-text-muted)] transition hover:border-[var(--color-ok)]/40 hover:text-[var(--color-text-secondary)] sm:w-[200px]"
      >
        <span class="text-base font-medium text-[var(--color-ok)]">
          {hiddenDoneCount} done
        </span>
        <span class="text-xs">Show completed</span>
      </button>
    {/if}
    <button
      type="button"
      onclick={onaddcolumn}
      class="flex h-28 w-[min(100%,240px)] shrink-0 cursor-pointer flex-col items-center justify-center gap-2 rounded-[1.75rem] border border-dashed border-[var(--color-border)] bg-transparent text-sm text-[var(--color-text-muted)] transition hover:border-[var(--color-accent)]/40 hover:text-[var(--color-text-secondary)] sm:w-[240px]"
    >
      <span class="text-lg">+</span>
      Add column
    </button>
  </div>
{/if}
