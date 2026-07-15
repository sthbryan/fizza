<script lang="ts">
  import { cn } from "@/lib/cn";

  export type SelectOption = {
    value: string;
    label: string;
    disabled?: boolean;
  };

  interface Props {
    value: string;
    options: SelectOption[];
    onchange: (value: string) => void;
    placeholder?: string;
    label?: string;
    disabled?: boolean;
    class?: string;
    size?: "sm" | "md";
  }

  let {
    value,
    options,
    onchange,
    placeholder = "Select…",
    label,
    disabled = false,
    class: className = "",
    size = "md",
  }: Props = $props();

  let open = $state(false);
  let active = $state(-1);
  let triggerEl = $state<HTMLButtonElement | null>(null);
  let listEl = $state<HTMLDivElement | null>(null);
  let pos = $state({ top: 0, left: 0, width: 0 });

  const selected = $derived(options.find((o) => o.value === value));

  function updatePos() {
    if (!triggerEl) return;
    const r = triggerEl.getBoundingClientRect();
    pos = {
      top: r.bottom + 6,
      left: r.left,
      width: Math.max(r.width, 160),
    };
  }

  $effect(() => {
    if (!open) return;
    updatePos();
    const onScroll = () => updatePos();
    window.addEventListener("resize", updatePos);
    window.addEventListener("scroll", onScroll, true);
    return () => {
      window.removeEventListener("resize", updatePos);
      window.removeEventListener("scroll", onScroll, true);
    };
  });

  $effect(() => {
    if (!open) return;
    const idx = options.findIndex((o) => !o.disabled && o.value === value);
    active = idx >= 0 ? idx : options.findIndex((o) => !o.disabled);
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

  function commit(v: string) {
    onchange(v);
    open = false;
    triggerEl?.focus();
  }

  function moveActive(delta: number) {
    const enabledIdxs = options
      .map((o, i) => (!o.disabled ? i : -1))
      .filter((i) => i >= 0);
    if (!enabledIdxs.length) return;
    const cur = enabledIdxs.indexOf(active);
    const next =
      enabledIdxs[
        Math.max(0, Math.min(enabledIdxs.length - 1, (cur < 0 ? 0 : cur) + delta))
      ];
    active = next;
  }

  function onKeyDown(e: KeyboardEvent) {
    if (disabled) return;
    if (!open && (e.key === "ArrowDown" || e.key === "Enter" || e.key === " ")) {
      e.preventDefault();
      open = true;
      return;
    }
    if (!open) return;
    if (e.key === "Escape") {
      e.preventDefault();
      open = false;
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      moveActive(1);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      moveActive(-1);
    } else if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      const opt = options[active];
      if (opt && !opt.disabled) commit(opt.value);
    }
  }

  $effect(() => {
    if (!open || active < 0) return;
    const item = listEl?.querySelector(`[data-idx="${active}"]`);
    item?.scrollIntoView({ block: "nearest" });
  });
</script>

<div class={cn("block min-w-0", className)}>
  {#if label}
    <span
      class="mb-2 block text-sm font-medium text-[var(--color-text-secondary)]"
    >
      {label}
    </span>
  {/if}
  <button
    bind:this={triggerEl}
    type="button"
    {disabled}
    aria-haspopup="listbox"
    aria-expanded={open}
    onclick={() => {
      if (!disabled) open = !open;
    }}
    onkeydown={onKeyDown}
    class={cn(
      "flex w-full cursor-pointer items-center justify-between gap-2 rounded-2xl border border-[var(--color-border)] bg-[var(--color-bg-soft)] text-left text-[var(--color-text)] transition",
      "hover:border-[var(--color-border)] hover:bg-[var(--color-bg-hover)]",
      "focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-accent)]/40",
      "disabled:cursor-not-allowed disabled:opacity-40",
      size === "sm" ? "h-9 px-3 text-sm" : "h-11 px-4 text-base"
    )}
  >
    <span class={cn("truncate", !selected && "text-[var(--color-text-muted)]")}>
      {selected?.label ?? placeholder}
    </span>
    <svg
      width="14"
      height="14"
      viewBox="0 0 24 24"
      fill="none"
      class={cn(
        "shrink-0 text-[var(--color-text-muted)] transition-transform",
        open && "rotate-180"
      )}
      aria-hidden="true"
    >
      <path
        d="M6 9l6 6 6-6"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  </button>

  {#if open}
    <div
      bind:this={listEl}
      role="listbox"
      class="fixed z-[80] max-h-72 overflow-auto rounded-2xl border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-1.5 shadow-[0_16px_48px_rgba(0,0,0,0.55)]"
      style:top="{pos.top}px"
      style:left="{pos.left}px"
      style:width="{pos.width}px"
    >
      {#if options.length === 0}
        <div class="px-3.5 py-2.5 text-sm text-[var(--color-text-muted)]">
          No options
        </div>
      {:else}
        {#each options as opt, idx (opt.value)}
          {@const isActive = idx === active}
          {@const isSelected = opt.value === value}
          <button
            type="button"
            role="option"
            data-idx={idx}
            aria-selected={isSelected}
            disabled={opt.disabled}
            class={cn(
              "flex w-full cursor-pointer items-center justify-between gap-2 rounded-xl px-3.5 py-2.5 text-left text-base transition",
              "disabled:cursor-not-allowed disabled:opacity-40",
              isActive || isSelected
                ? "bg-[var(--color-bg-hover)] text-[var(--color-text)]"
                : "text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text)]"
            )}
            onmouseenter={() => {
              if (!opt.disabled) active = idx;
            }}
            onclick={() => {
              if (!opt.disabled) commit(opt.value);
            }}
          >
            <span class="truncate">{opt.label}</span>
            {#if isSelected}
              <span class="text-[var(--color-accent)]">✓</span>
            {/if}
          </button>
        {/each}
      {/if}
    </div>
  {/if}
</div>
