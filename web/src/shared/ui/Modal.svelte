<script lang="ts">
  import type { Snippet } from "svelte";
  import { onMount } from "svelte";
  import { fade } from "svelte/transition";
  import { cubicOut } from "svelte/easing";
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
      in:fade={{ duration: 150, easing: cubicOut }}
      out:fade={{ duration: 100, easing: cubicOut }}
      onclick={onclose}
    ></button>
    <div
      role="dialog"
      aria-modal="true"
      class="relative z-10 m-auto w-full max-w-lg overflow-hidden rounded-2xl border border-neutral-700 bg-neutral-950"
      in:fade={{ duration: 200, easing: cubicOut }}
      out:fade={{ duration: 150, easing: cubicOut }}
    >
      <div
        class="flex items-center justify-between border-b border-neutral-800 px-4 py-4 sm:px-6"
      >
        <span class="text-label font-mono uppercase text-neutral-400">
          {title}
        </span>
        <button
          type="button"
          aria-label="Close"
          onclick={onclose}
          class="min-h-11 cursor-pointer px-2 text-label font-mono uppercase text-neutral-500 transition-colors hover:text-neutral-200"
        >
          [ X ]
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