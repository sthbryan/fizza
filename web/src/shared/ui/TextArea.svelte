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
    "w-full min-h-28 resize-y border border-neutral-700 bg-transparent px-3 py-2.5 text-base leading-relaxed text-neutral-200 placeholder:text-neutral-500 transition-colors " +
    "rounded-lg font-mono focus:border-neutral-200 focus:outline-none";
</script>

<label class="block min-w-0">
  {#if label}
    <span
      class="mb-2 block text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-400"
    >
      {label}
    </span>
  {/if}
  <textarea class={cn(fieldClass, className)} bind:value {...rest}></textarea>
</label>