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
    size = 128,
    stroke = 10,
    label = "",
    sublabel = "",
  }: Props = $props();

  const r = $derived((size - stroke) / 2);
  const c = $derived(2 * Math.PI * r);
  const clamped = $derived(Math.max(0, Math.min(100, value)));
  const offset = $derived(c - (clamped / 100) * c);
</script>

<div class="flex flex-col items-center gap-2">
  <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} class="block">
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
      stroke="var(--color-ok)"
      stroke-width={stroke}
      stroke-linecap="round"
      stroke-dasharray={c}
      stroke-dashoffset={offset}
      transform={`rotate(-90 ${size / 2} ${size / 2})`}
      style="transition: stroke-dashoffset 0.4s ease"
    />
    <text
      x="50%"
      y="50%"
      text-anchor="middle"
      dominant-baseline="central"
      class="fill-[var(--color-text)] text-2xl font-semibold"
      style="font-size: 1.5rem; font-weight: 600; fill: var(--color-text)"
    >
      {clamped}%
    </text>
  </svg>
  {#if label}
    <div class="text-center">
      <div class="text-sm font-medium text-[var(--color-text-secondary)]">
        {label}
      </div>
      {#if sublabel}
        <div class="text-xs text-[var(--color-text-muted)]">{sublabel}</div>
      {/if}
    </div>
  {/if}
</div>
