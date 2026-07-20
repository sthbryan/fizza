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
    color = "var(--color-text-display)",
    emptyLabel = "No activity in this period",
  }: Props = $props();

  const series = $derived(fillDaySeries(rows, days));
  const max = $derived(maxCount(series));
  const hasAny = $derived(series.some((r) => r.count > 0));
  const total = $derived(series.reduce((s, r) => s + r.count, 0));
</script>

{#if !hasAny}
  <p class="py-8 text-center text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-500">
    {emptyLabel}
  </p>
{:else}
  <div class="flex flex-col gap-3">
    <div class="flex h-28 gap-0.5 sm:h-32 sm:gap-1">
      {#each series as day (day.date)}
        {@const h = max > 0 ? Math.max((day.count / max) * 100, day.count > 0 ? 6 : 0) : 0}
        <div
          class="group relative flex min-w-0 flex-1 flex-col items-center justify-end"
          title="{day.date}: {day.count}"
        >
          <div
            class="w-full max-w-3 rounded-none transition-opacity duration-200 group-hover:opacity-70"
            style:height="{h}%"
            style:background={day.count > 0 ? color : "var(--color-border-subtle)"}
            style:min-height={day.count > 0 ? "2px" : "1px"}
          ></div>
        </div>
      {/each}
    </div>
    <div
      class="flex justify-between text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-500"
    >
      <span>{formatDayLabel(series[0]?.date ?? "")}</span>
      <span class="tabular-nums text-white">{total} total</span>
      <span>{formatDayLabel(series[series.length - 1]?.date ?? "")}</span>
    </div>
  </div>
{/if}