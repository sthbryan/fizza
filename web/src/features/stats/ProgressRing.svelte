<script lang="ts">
  interface Props {
    value: number;
    size?: number;
    stroke?: number;
    label?: string;
    sublabel?: string;
  }

  let {
    value,
    size = 160,
    stroke = 4,
    label = "",
    sublabel = "",
  }: Props = $props();

  const r = $derived((size - stroke) / 2);
  const c = $derived(2 * Math.PI * r);
  const clamped = $derived(Math.max(0, Math.min(100, value)));
  const offset = $derived(c - (clamped / 100) * c);

  const ticks = Array.from({ length: 60 }, (_, i) => i);
</script>

<div class="flex flex-col items-center gap-3">
  <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} class="block">
    {#each ticks as i}
      {@const a = (i / ticks.length) * Math.PI * 2 - Math.PI / 2}
      {@const x1 = size / 2 + Math.cos(a) * (size / 2 - 4)}
      {@const y1 = size / 2 + Math.sin(a) * (size / 2 - 4)}
      {@const x2 = size / 2 + Math.cos(a) * (size / 2 - 10)}
      {@const y2 = size / 2 + Math.sin(a) * (size / 2 - 10)}
      <line
        {x1}
        {y1}
        {x2}
        {y2}
        stroke="var(--color-border-visible)"
        stroke-width="1"
        opacity={i % 5 === 0 ? 1 : 0.4}
      />
    {/each}
    <circle
      cx={size / 2}
      cy={size / 2}
      {r}
      fill="none"
      stroke="var(--color-border-subtle)"
      stroke-width={stroke}
    />
    <circle
      cx={size / 2}
      cy={size / 2}
      {r}
      fill="none"
      stroke="var(--color-text-display)"
      stroke-width={stroke}
      stroke-linecap="butt"
      stroke-dasharray={c}
      stroke-dashoffset={offset}
      transform={`rotate(-90 ${size / 2} ${size / 2})`}
      style="transition: stroke-dashoffset 0.4s ease-out"
    />
    <text
      x="50%"
      y="50%"
      text-anchor="middle"
      dominant-baseline="central"
      font-family="'Doto', 'Space Mono', monospace"
      font-weight="500"
      font-size="32"
      fill="var(--color-text-display)"
    >
      {clamped}
    </text>
    <text
      x="50%"
      y="62%"
      text-anchor="middle"
      dominant-baseline="central"
      font-family="'Space Mono', monospace"
      font-size="9"
      letter-spacing="0.1em"
      fill="var(--color-text-secondary)"
    >
      PERCENT
    </text>
  </svg>
  {#if label}
    <div class="text-center">
      <div class="text-[10px] font-mono font-medium uppercase tracking-[0.1em] text-neutral-400">
        {label}
      </div>
      {#if sublabel}
        <div class="mt-1 text-xs text-neutral-500">{sublabel}</div>
      {/if}
    </div>
  {/if}
</div>