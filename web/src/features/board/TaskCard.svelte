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
    "group cursor-grab rounded-md border border-neutral-800 bg-neutral-950 p-3.5 transition-colors sm:p-4",
    "min-h-[5rem] hover:border-neutral-700 hover:bg-neutral-900",
    "active:cursor-grabbing",
    dragging && "opacity-30 ring-1 ring-white/30"
  )}
>
  <div class="mb-2 flex items-start justify-between gap-2">
    <Badge priority={task.priority} />
    <div class="shrink-0">
      <Menu items={menuItems} label="Task actions" />
    </div>
  </div>

  <h4
    class="mb-2 text-[14px] font-medium leading-snug tracking-tight text-neutral-100 sm:text-[15px]"
  >
    {task.title}
  </h4>

  {#if task.description}
    <p class="mb-3 line-clamp-3 text-xs leading-relaxed text-neutral-500">
      {task.description}
    </p>
  {/if}

  <div class="flex flex-wrap items-center gap-3 text-[10px] font-mono uppercase tracking-[0.08em] text-neutral-500">
    <span>#{task.id}</span>
    {#if due}
      <span>{due}</span>
    {/if}
  </div>
</article>