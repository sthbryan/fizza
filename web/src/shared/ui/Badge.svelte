<script lang="ts">
  import { cn } from "@/lib/cn";
  import type { Priority } from "@/lib/api";

  interface Props {
    priority: Priority | string;
  }

  let { priority }: Props = $props();

  const styles: Record<string, string> = {
    low: "border-neutral-700 text-neutral-400",
    medium: "border-neutral-600 text-neutral-200",
    high: "border-neutral-500 text-neutral-200",
    urgent: "border-red-500 text-red-500",
  };

  const key = $derived(String(priority || "medium").toLowerCase());
</script>

<span
  class={cn(
    "inline-flex items-center rounded-full border bg-transparent px-3 py-0.5 text-[11px] font-mono uppercase tracking-[0.08em]",
    styles[key] || styles.medium
  )}
>
  {key}
</span>