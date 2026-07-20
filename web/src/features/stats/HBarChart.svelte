<script lang="ts">
  import type { NamedCount } from "@/lib/api";
  import SegmentedBar from "@/shared/ui/SegmentedBar.svelte";
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
  <p class="py-6 text-center text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-500">
    {emptyLabel}
  </p>
{:else}
  <div class="flex flex-col gap-4">
    {#each rows as row (row.name)}
      <div class="min-w-0">
        <div class="mb-2 flex items-baseline justify-between gap-2">
          <span class="truncate text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-400">
            {labelFor(row.name)}
          </span>
          <span class="shrink-0 font-mono text-base tabular-nums text-white">
            {row.count}
          </span>
        </div>
        <SegmentedBar
          value={row.count}
          max={max || 1}
          segments={16}
          fill={colorFor(row.name)}
          size="sm"
        />
      </div>
    {/each}
  </div>
{/if}