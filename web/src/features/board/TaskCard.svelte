<script lang="ts">
  import type { Task } from "@/lib/api";
  import Badge from "@/shared/ui/Badge.svelte";
  import Menu from "@/shared/ui/Menu.svelte";
  import type { MenuItem } from "@/shared/ui/Menu.svelte";
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

  const menuItems = $derived<MenuItem[]>(
    [
      ...(terminal && onrestore
        ? [{ label: "Restore", onSelect: () => onrestore(task) }]
        : []),
      ...(onarchive
        ? [{ label: "Archive", onSelect: () => onarchive(task) }]
        : []),
      { label: "Edit", onSelect: () => onedit(task) },
      {
        label: "Delete",
        danger: true,
        onSelect: () => ondelete(task),
      },
    ]
  );
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
    <div class="shrink-0">
      <Menu items={menuItems} label="Task actions" />
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
