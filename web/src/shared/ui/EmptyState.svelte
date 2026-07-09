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

<div class="mx-auto flex max-w-md flex-col items-center px-6 py-28 text-center">
  <div
    class="mb-5 flex h-14 w-14 items-center justify-center rounded-2xl border border-[var(--color-border)] bg-[var(--color-bg-soft)] text-xl text-[var(--color-accent)]"
  >
    ◈
  </div>
  <h2 class="text-xl font-semibold tracking-tight">{title}</h2>
  <p class="mt-2.5 text-base leading-relaxed text-[var(--color-text-muted)]">
    {description}
  </p>
  {#if actionLabel && onaction}
    <Button class="mt-7" variant="primary" size="lg" onclick={onaction}>
      {actionLabel}
    </Button>
  {/if}
</div>
