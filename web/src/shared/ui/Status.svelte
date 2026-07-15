<script lang="ts">
  import { getStatus } from "@/lib/status/status.svelte";

  const status = $derived(getStatus());
</script>

{#if status}
  <div
    class="px-4 py-2 text-[11px] font-mono uppercase tracking-[0.1em] text-neutral-500"
    class:text-red-500={status.kind === "error"}
    role="status"
  >
    [{status.kind === "error" ? "ERROR" : "SAVED"}] {status.message}
  </div>
{/if}