<script lang="ts">
  import type { Snippet } from "svelte";
  import { onMount } from "svelte";
  import Button from "./Button.svelte";

  interface Props {
    open: boolean;
    title: string;
    submitLabel?: string;
    busy?: boolean;
    onclose: () => void;
    onsubmit?: () => void | Promise<void>;
    children: Snippet;
  }

  let {
    open,
    title,
    submitLabel = "Save",
    busy = false,
    onclose,
    onsubmit,
    children,
  }: Props = $props();

  onMount(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" && open && !busy) onclose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });
</script>

{#if open}
  <div class="fixed inset-0 z-50 flex items-end justify-center p-4 sm:items-center">
    <button
      type="button"
      aria-label="Close dialog"
      class="absolute inset-0 bg-black/70 backdrop-blur-sm"
      onclick={onclose}
    ></button>
    <div
      role="dialog"
      aria-modal="true"
      class="relative z-10 w-full max-w-md overflow-hidden rounded-3xl border border-[var(--color-border)] bg-[var(--color-bg-elevated)] shadow-[0_24px_64px_rgba(0,0,0,0.55)]"
    >
      <div
        class="flex items-center justify-between border-b border-[var(--color-border-subtle)] px-5 py-4"
      >
        <h2 class="text-base font-semibold tracking-tight">{title}</h2>
        <Button variant="icon" size="sm" onclick={onclose} aria-label="Close"
          >×</Button
        >
      </div>
      <div class="space-y-3.5 px-5 py-5">
        {@render children()}
      </div>
      <div
        class="flex justify-end gap-2 border-t border-[var(--color-border-subtle)] px-5 py-4"
      >
        <Button variant="ghost" onclick={onclose} disabled={busy}>Cancel</Button>
        {#if onsubmit}
          <Button
            variant="primary"
            disabled={busy}
            onclick={() => void onsubmit()}
          >
            {busy ? "Saving…" : submitLabel}
          </Button>
        {/if}
      </div>
    </div>
  </div>
{/if}
