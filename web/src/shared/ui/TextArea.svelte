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
    "w-full rounded-md border border-[var(--color-border-visible)] bg-[var(--color-bg-elevated)] px-3.5 py-2.5 text-base text-[var(--color-text)] placeholder:text-[var(--color-text-muted)] transition-colors " +
    "font-mono focus:border-[var(--color-text-display)] focus:outline-none min-h-28 resize-y leading-relaxed";
</script>

<label class="block min-w-0">
  {#if label}
    <span
      class="mb-2 block text-[10px] font-mono font-medium uppercase tracking-[0.1em] text-[var(--color-text-secondary)]"
    >
      {label}
    </span>
  {/if}
  <textarea class={cn(fieldClass, className)} bind:value {...rest}></textarea>
</label>