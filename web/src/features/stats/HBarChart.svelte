<script lang="ts">
  import type { NamedCount } from "@/lib/api";
  import { maxCount } from "./utils";

  interface Props {
    rows: NamedCount[];
    colorFor?: (name: string) => string;
    labelFor?: (name: string) => string;
    emptyLabel?: string;
  }

  let {
    rows,
    colorFor = () => "var(--color-text-display)",
    labelFor = (name: string) => name,
    emptyLabel = "No data yet",
  }: Props = $props();

  const max = $derived(maxCount(rows));
  const hasAny = $derived(rows.some((r) => r.count > 0));
</script>

{#if !hasAny}
  <p class="py-6 text-center text-xs font-mono uppercase tracking-[0.1em] text-neutral-500">
    {emptyLabel}
  </p>
{:else}
  <div class="flex flex-col gap-3">
    {#each rows as row (row.name)}
      {@const width = max > 0 ? (row.count / max) * 100 : 0}
      <div class="min-w-0">
        <div class="mb-1.5 flex items-baseline justify-between gap-2">
          <span class="truncate text-[10px] font-mono uppercase tracking-[0.1em] text-neutral-400">
            {labelFor(row.name)}
          </span>
          <span class="shrink-0 font-mono text-xs tabular-nums text-white">
            {row.count}
          </span>
        </div>
        <div
          class="h-1 overflow-hidden bg-neutral-800"
        >
          <div
            class="h-full transition-all duration-300"
            style:width="{width}%"
            style:background={colorFor(row.name)}
          ></div>
        </div>
      </div>
    {/each}
  </div>
{/if}