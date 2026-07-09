<script lang="ts">
  import type { HTMLInputAttributes } from "svelte/elements";
  import { cn } from "@/lib/cn";

  interface Props extends HTMLInputAttributes {
    label?: string;
    class?: string;
    value?: string;
  }

  let {
    label,
    class: className = "",
    value = $bindable(""),
    ...rest
  }: Props = $props();

  const fieldClass =
    "w-full rounded-2xl border border-[var(--color-border)] bg-[var(--color-bg-soft)] px-4 py-3 text-base text-[var(--color-text)] placeholder:text-[var(--color-text-muted)] transition " +
    "focus:border-[var(--color-accent)]/50 focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]/20";
</script>

<label class="block min-w-0">
  {#if label}
    <span
      class="mb-2 block text-xs font-medium uppercase tracking-[0.06em] text-[var(--color-text-muted)]"
    >
      {label}
    </span>
  {/if}
  <input class={cn(fieldClass, className)} bind:value {...rest} />
</label>
