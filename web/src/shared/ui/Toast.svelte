<script lang="ts">
  import { cn } from "@/lib/cn";
  import { getToast } from "@/lib/toast/toast.svelte";

  const toast = $derived(getToast());
</script>

{#if toast}
  <div
    class={cn(
      "fixed bottom-6 right-6 z-[60] max-w-md rounded-2xl border px-5 py-3.5 text-base font-medium shadow-lg",
      "bg-[var(--color-bg-elevated)]",
      toast.kind === "error"
        ? "border-[var(--color-danger)]/30 text-[var(--color-danger)]"
        : "border-[var(--color-ok)]/30 text-[var(--color-ok)]"
    )}
    role="status"
  >
    {toast.message}
  </div>
{/if}
