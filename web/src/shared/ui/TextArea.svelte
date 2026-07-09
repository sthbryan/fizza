<script lang="ts">
  import type { HTMLTextareaAttributes } from "svelte/elements";
  import { cn } from "@/lib/cn";

  interface Props extends HTMLTextareaAttributes {
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
    "w-full rounded-2xl border border-[var(--color-border)] bg-[var(--color-bg-soft)] px-3.5 py-2.5 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-muted)] transition " +
    "focus:border-[var(--color-accent)]/50 focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]/20 min-h-24 resize-y leading-relaxed";
</script>

<label class="block min-w-0">
  {#if label}
    <span
      class="mb-1.5 block text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-text-muted)]"
    >
      {label}
    </span>
  {/if}
  <textarea class={cn(fieldClass, className)} bind:value {...rest}></textarea>
</label>
