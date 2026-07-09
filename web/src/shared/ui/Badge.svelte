<script lang="ts">
  import { cn } from "@/lib/cn";
  import type { Priority } from "@/lib/api";

  interface Props {
    priority: Priority | string;
  }

  let { priority }: Props = $props();

  const styles: Record<string, string> = {
    low: "bg-[var(--color-pri-low)]/12 text-[var(--color-pri-low)]",
    medium: "bg-[var(--color-pri-medium)]/15 text-[var(--color-pri-medium)]",
    high: "bg-[var(--color-pri-high)]/15 text-[var(--color-pri-high)]",
    urgent: "bg-[var(--color-pri-urgent)]/15 text-[var(--color-pri-urgent)]",
  };

  const key = $derived(String(priority || "medium").toLowerCase());
</script>

<span
  class={cn(
    "inline-flex items-center rounded-lg px-2 py-0.5 text-[11px] font-medium capitalize",
    styles[key] || styles.medium
  )}
>
  {key}
</span>
