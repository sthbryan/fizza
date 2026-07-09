<script lang="ts">
  import type { DayCount } from "@/lib/api";
  import { fillDaySeries, formatDayLabel, maxCount } from "./utils";

  interface Props {
    rows: DayCount[];
    days?: number;
    color?: string;
    emptyLabel?: string;
  }

  let {
    rows,
    days = 30,
    color = "var(--color-accent)",
    emptyLabel = "No activity in this period",
  }: Props = $props();

  const series = $derived(fillDaySeries(rows, days));
  const max = $derived(maxCount(series));
  const hasAny = $derived(series.some((r) => r.count > 0));
  const total = $derived(series.reduce((s, r) => s + r.count, 0));
</script>

{#if !hasAny}
  <p class="py-8 text-center text-sm text-[var(--color-text-muted)]">
    {emptyLabel}
  </p>
{:else}
  <div class="flex flex-col gap-3">
    <div class="flex h-32 items-end gap-0.5 sm:h-36 sm:gap-1">
      {#each series as day (day.date)}
        {@const h = max > 0 ? Math.max((day.count / max) * 100, day.count > 0 ? 6 : 0) : 0}
        <div
          class="group relative flex min-w-0 flex-1 flex-col items-center justify-end"
          title="{day.date}: {day.count}"
        >
          <div
            class="w-full max-w-3 rounded-t-sm transition-all duration-200 group-hover:opacity-90"
            style:height="{h}%"
            style:background={day.count > 0 ? color : "var(--color-border-subtle)"}
            style:min-height={day.count > 0 ? "3px" : "1px"}
          ></div>
        </div>
      {/each}
    </div>
    <div
      class="flex justify-between text-[10px] text-[var(--color-text-muted)] sm:text-xs"
    >
      <span>{formatDayLabel(series[0]?.date ?? "")}</span>
      <span class="text-[var(--color-text-secondary)]">{total} total</span>
      <span>{formatDayLabel(series[series.length - 1]?.date ?? "")}</span>
    </div>
  </div>
{/if}
