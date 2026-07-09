<script lang="ts">
  import Button from "./Button.svelte";

  interface Props {
    title: string;
    description: string;
    actionLabel?: string;
    onaction?: () => void;
  }

  let { title, description, actionLabel, onaction }: Props = $props();
</script>

<div class="mx-auto flex max-w-sm flex-col items-center px-6 py-24 text-center">
  <div
    class="mb-4 flex h-12 w-12 items-center justify-center rounded-2xl border border-[var(--color-border)] bg-[var(--color-bg-soft)] text-lg text-[var(--color-accent)]"
  >
    ◈
  </div>
  <h2 class="text-lg font-semibold tracking-tight">{title}</h2>
  <p class="mt-2 text-sm leading-relaxed text-[var(--color-text-muted)]">
    {description}
  </p>
  {#if actionLabel && onaction}
    <Button class="mt-6" variant="primary" onclick={onaction}>
      {actionLabel}
    </Button>
  {/if}
</div>
