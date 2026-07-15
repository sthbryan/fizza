<script lang="ts">
  import type { Snippet } from "svelte";
  import { cn } from "@/lib/cn";

  export type MenuItem = {
    label: string;
    icon?: Snippet;
    danger?: boolean;
    disabled?: boolean;
    onSelect: () => void;
  };

  interface Props {
    items: MenuItem[];
    label?: string;
    align?: "start" | "end";
    class?: string;
  }

  let {
    items,
    label = "Open menu",
    align = "end",
    class: className = "",
  }: Props = $props();

  let open = $state(false);
  let active = $state(-1);
  let triggerEl = $state<HTMLButtonElement | null>(null);
  let listEl = $state<HTMLDivElement | null>(null);
  let pos = $state({ top: 0, left: 0, width: 0 });

  function updatePos() {
    if (!triggerEl) return;
    const r = triggerEl.getBoundingClientRect();
    pos = {
      top: r.bottom + 6,
      left: align === "end" ? r.right : r.left,
      width: Math.max(r.width, 180),
    };
  }

  $effect(() => {
    if (!open) return;
    updatePos();
    window.addEventListener("resize", updatePos);
    window.addEventListener(
      "scroll",
      () => {
        updatePos();
      },
      true
    );
    return () => {
      window.removeEventListener("resize", updatePos);
      window.removeEventListener("scroll", updatePos, true);
    };
  });

  $effect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      const t = e.target as Node;
      if (triggerEl?.contains(t)) return;
      if (listEl?.contains(t)) return;
      open = false;
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  });

  $effect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        open = false;
        triggerEl?.focus();
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  });

  function commit(item: MenuItem) {
    if (item.disabled) return;
    open = false;
    triggerEl?.focus();
    item.onSelect();
  }

  function moveActive(delta: number) {
    const enabledIdxs = items
      .map((_, i) => (!items[i].disabled ? i : -1))
      .filter((i) => i >= 0);
    if (!enabledIdxs.length) return;
    const cur = enabledIdxs.indexOf(active);
    const next =
      enabledIdxs[
        Math.max(
          0,
          Math.min(enabledIdxs.length - 1, (cur < 0 ? 0 : cur) + delta)
        )
      ];
    active = next;
  }

  function onKeyDown(e: KeyboardEvent) {
    if (!open && (e.key === "ArrowDown" || e.key === "Enter" || e.key === " ")) {
      e.preventDefault();
      open = true;
      active = 0;
      return;
    }
    if (!open) return;
    if (e.key === "ArrowDown") {
      e.preventDefault();
      moveActive(1);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      moveActive(-1);
    } else if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      const item = items[active];
      if (item) commit(item);
    }
  }

  $effect(() => {
    if (!open || active < 0) return;
    const item = listEl?.querySelector(`[data-idx="${active}"]`);
    item?.scrollIntoView({ block: "nearest" });
  });
</script>

<button
  bind:this={triggerEl}
  type="button"
  aria-label={label}
  aria-haspopup="menu"
  aria-expanded={open}
  onclick={() => {
    open = !open;
    if (open) active = 0;
  }}
  onkeydown={onKeyDown}
  class={cn(
    "flex h-8 w-8 cursor-pointer items-center justify-center rounded-md text-neutral-500 transition-colors",
    "hover:bg-neutral-900 hover:text-neutral-200",
    "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/30",
    className
  )}
>
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <circle cx="12" cy="5" r="1.5" fill="currentColor" />
    <circle cx="12" cy="12" r="1.5" fill="currentColor" />
    <circle cx="12" cy="19" r="1.5" fill="currentColor" />
  </svg>
</button>

{#if open}
  <div
    bind:this={listEl}
    role="menu"
    class="fixed z-[80] min-w-[10rem] overflow-auto rounded-md border border-neutral-700 bg-neutral-950 p-1"
    style:top="{pos.top}px"
    style:left="{pos.left}px"
    style:transform={align === "end" ? "translateX(-100%)" : "none"}
  >
    {#each items as item, idx (idx)}
      {@const isActive = idx === active}
      <button
        type="button"
        role="menuitem"
        data-idx={idx}
        disabled={item.disabled}
        onmouseenter={() => {
          if (!item.disabled) active = idx;
        }}
        onclick={() => commit(item)}
        class={cn(
          "flex w-full cursor-pointer items-center gap-2.5 rounded-sm px-3 py-2 text-left text-[11px] font-mono uppercase tracking-[0.08em] transition-colors",
          "disabled:cursor-not-allowed disabled:opacity-30",
          item.danger
            ? "text-red-500 hover:bg-red-500/10"
            : "text-neutral-400 hover:bg-neutral-900 hover:text-neutral-200",
          isActive && !item.danger && "bg-neutral-900 text-neutral-200",
          isActive && item.danger && "bg-red-500/10"
        )}
      >
        {#if item.icon}
          <span class="shrink-0">{@render item.icon()}</span>
        {/if}
        <span class="truncate">{item.label}</span>
      </button>
    {/each}
  </div>
{/if}