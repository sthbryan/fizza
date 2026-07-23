<script lang="ts">
  import { fade } from "svelte/transition";
  import { cubicOut } from "svelte/easing";
  import { getStatus } from "@/lib/status/status.svelte";

  const status = $derived(getStatus());
</script>

{#if status}
  <div
    class="px-4 py-2 text-label font-mono uppercase text-neutral-500"
    class:text-accent={status.kind === "error"}
    role="status"
    in:fade={{ duration: 150, easing: cubicOut }}
    out:fade={{ duration: 120, easing: cubicOut }}
  >
    [{status.kind === "error" ? "ERROR" : "SAVED"}] {status.message}
  </div>
{/if}