<script lang="ts">
  import type { Task } from "@/lib/api";
  import Badge from "@/shared/ui/Badge.svelte";
  import { cn } from "@/lib/cn";

  interface Props {
    task: Task;
    dragging?: boolean;
    terminal?: boolean;
    ondragstart: (task: Task) => void;
    ondragend: () => void;
    onedit: (task: Task) => void;
    ondelete: (task: Task) => void;
    onarchive?: (task: Task) => void;
    onrestore?: (task: Task) => void;
  }

  let {
    task,
    dragging = false,
    terminal = false,
    ondragstart,
    ondragend,
    onedit,
    ondelete,
    onarchive,
    onrestore,
  }: Props = $props();

  const due = $derived(task.due_date ? String(task.due_date).slice(0, 10) : null);
</script>

<article
  draggable="true"
  ondragstart={(e) => {
    e.dataTransfer!.effectAllowed = "move";
    e.dataTransfer!.setData("text/plain", String(task.id));
    ondragstart(task);
  }}
  ondragend={ondragend}
  class={cn(
    "group cursor-grab rounded-2xl border border-white/8 bg-[#141418] p-4 shadow-[0_4px_16px_rgba(0,0,0,0.25)] transition sm:p-5",
    "min-h-[5.75rem] hover:border-white/12 hover:bg-[#1a1a20]",
    "active:cursor-grabbing",
    dragging && "scale-[0.98] opacity-40 ring-1 ring-[var(--color-accent)]/40"
  )}
>
  <div class="mb-2.5 flex items-start justify-between gap-2">
    <Badge priority={task.priority} />
    <div
      class="flex shrink-0 gap-0.5 opacity-100 sm:opacity-0 sm:transition sm:group-hover:opacity-100"
    >
      {#if terminal && onrestore}
        <button
          type="button"
          class="cursor-pointer rounded-lg px-2 py-1 text-xs text-[var(--color-text-muted)] hover:bg-white/5 hover:text-[var(--color-ok)]"
          onclick={() => onrestore(task)}
        >
          Restore
        </button>
      {/if}
      {#if onarchive}
        <button
          type="button"
          class="cursor-pointer rounded-lg px-2 py-1 text-xs text-[var(--color-text-muted)] hover:bg-white/5 hover:text-[var(--color-text)]"
          onclick={() => onarchive(task)}
        >
          Archive
        </button>
      {/if}
      <button
        type="button"
        class="cursor-pointer rounded-lg px-2 py-1 text-xs text-[var(--color-text-muted)] hover:bg-white/5 hover:text-[var(--color-text)]"
        onclick={() => onedit(task)}
      >
        Edit
      </button>
      <button
        type="button"
        class="cursor-pointer rounded-lg px-2 py-1 text-xs text-[var(--color-text-muted)] hover:bg-[var(--color-danger)]/10 hover:text-[var(--color-danger)]"
        onclick={() => ondelete(task)}
      >
        Del
      </button>
    </div>
  </div>

  <h4
    class="mb-2 text-[15px] font-semibold leading-snug tracking-tight text-[var(--color-text)] sm:text-base"
  >
    {task.title}
  </h4>

  {#if task.description}
    <p class="mb-3 line-clamp-3 text-sm leading-relaxed text-[var(--color-text-muted)]">
      {task.description}
    </p>
  {/if}

  <div class="flex flex-wrap items-center gap-2">
    <span class="font-mono text-[11px] text-[var(--color-text-muted)]">
      #{task.id}
    </span>
    {#if due}
      <span class="text-[11px] text-[var(--color-text-muted)]">{due}</span>
    {/if}
  </div>
</article>
