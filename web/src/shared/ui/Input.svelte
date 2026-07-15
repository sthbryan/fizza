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
    "w-full rounded-md border border-neutral-700 bg-neutral-900 px-3.5 py-2.5 text-base text-neutral-200 placeholder:text-neutral-500 transition-colors " +
    "font-mono focus:border-white focus:outline-none";
</script>

<label class="block min-w-0">
  {#if label}
    <span
      class="mb-2 block text-[10px] font-mono font-medium uppercase tracking-[0.1em] text-neutral-400"
    >
      {label}
    </span>
  {/if}
  <input class={cn(fieldClass, className)} bind:value {...rest} />
</label>