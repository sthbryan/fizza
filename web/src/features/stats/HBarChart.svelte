<script lang="ts">
  import type { NamedCount } from "@/lib/api";
  import { maxCount } from "./utils";

  interface Props {
    rows: NamedCount[];
    colorFor?: (name: string) => string;
    emptyLabel?: string;
  }

  let {
    rows,
    colorFor = () => "var(--color-accent)",
    emptyLabel = "No data yet",
  }: Props = $props();

  const max = $derived(maxCount(rows));
  const hasAny = $derived(rows.some((r) => r.count > 0));
</script>

{#if !hasAny}
  <p class="py-6 text-center text-sm text-[var(--color-text-muted)]">
    {emptyLabel}
  </p>
{:else}
  <div class="flex flex-col gap-3">
    {#each rows as row (row.name)}
      {@const width = max > 0 ? (row.count / max) * 100 : 0}
      <div class="min-w-0">
        <div class="mb-1 flex items-baseline justify-between gap-2">
          <span class="truncate text-sm text-[var(--color-text-secondary)]">
            {row.name}
          </span>
          <span class="shrink-0 font-mono text-xs text-[var(--color-text-muted)]">
            {row.count}
          </span>
        </div>
        <div
          class="h-2.5 overflow-hidden rounded-full bg-[var(--color-bg-soft)]"
        >
          <div
            class="h-full rounded-full transition-all duration-300"
            style:width="{width}%"
            style:background={colorFor(row.name)}
          ></div>
        </div>
      </div>
    {/each}
  </div>
{/if}
