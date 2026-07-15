<script lang="ts">
  import type { Snippet } from "svelte";
  import { onMount } from "svelte";
  import Button from "./Button.svelte";

  interface Props {
    open: boolean;
    title: string;
    submitLabel?: string;
    cancelLabel?: string;
    submitVariant?: "primary" | "destructive";
    busy?: boolean;
    onclose: () => void;
    onsubmit?: () => void | Promise<void>;
    children: Snippet;
  }

  let {
    open,
    title,
    submitLabel = "Save",
    cancelLabel = "Cancel",
    submitVariant = "primary",
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
  <div class="fixed inset-0 z-50 flex items-end justify-center p-4 sm:items-center sm:p-6">
    <button
      type="button"
      aria-label="Close dialog"
      class="absolute inset-0 bg-black/80"
      onclick={onclose}
    ></button>
    <div
      role="dialog"
      aria-modal="true"
      class="relative z-10 w-full max-w-lg overflow-hidden rounded-xl border border-neutral-700 bg-neutral-950"
    >
      <div
        class="flex items-center justify-between border-b border-neutral-800 px-6 py-4"
      >
        <span class="text-[10px] font-mono font-medium uppercase tracking-[0.1em] text-neutral-400">
          {title}
        </span>
        <button
          type="button"
          aria-label="Close"
          onclick={onclose}
          class="flex h-8 w-8 cursor-pointer items-center justify-center rounded-md text-base font-mono text-neutral-500 transition-colors hover:bg-neutral-900 hover:text-neutral-200"
        >
          ×
        </button>
      </div>
      <div class="space-y-4 px-6 py-5">
        {@render children()}
      </div>
      <div
        class="flex justify-end gap-2 border-t border-neutral-800 px-6 py-4"
      >
        <Button variant="ghost" onclick={onclose} disabled={busy}>{cancelLabel}</Button>
        {#if onsubmit}
          <Button
            variant={submitVariant}
            disabled={busy}
            onclick={() => void onsubmit()}
          >
            {busy ? "Working…" : submitLabel}
          </Button>
        {/if}
      </div>
    </div>
  </div>
{/if}