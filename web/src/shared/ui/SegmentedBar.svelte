<script lang="ts">
  import { cn } from "@/lib/cn";

  interface Props {
    value: number;
    max?: number;
    segments?: number;
    fill?: string;
    empty?: string;
    size?: "sm" | "md" | "lg";
    class?: string;
  }

  let {
    value,
    max = 100,
    segments = 20,
    fill = "var(--color-text-display)",
    empty = "var(--color-border-subtle)",
    size = "md",
    class: className = "",
  }: Props = $props();

  const filled = $derived(
    Math.max(0, Math.min(segments, Math.round((value / Math.max(max, 1)) * segments)))
  );

  const height = $derived(
    size === "lg" ? "h-4" : size === "sm" ? "h-1.5" : "h-2.5"
  );
</script>

<div
  class={cn("flex w-full gap-0.5", height, className)}
  role="img"
  aria-label={`${value} of ${max}`}
>
  {#each Array.from({ length: segments }, (_, i) => i) as i (i)}
    <div
      class="min-w-0 flex-1 rounded-none"
      style:background={i < filled ? fill : empty}
    ></div>
  {/each}
</div>
