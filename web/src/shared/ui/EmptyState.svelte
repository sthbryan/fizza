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

<div class="mx-auto flex max-w-md flex-col items-center px-6 py-24 text-center sm:py-28">
  <h2 class="mb-3 text-base tracking-tight text-neutral-400">
    {title}
  </h2>
  <p class="max-w-sm text-[11px] font-mono uppercase tracking-[0.08em] leading-relaxed text-neutral-600">
    {description}
  </p>
  {#if actionLabel && onaction}
    <Button class="mt-8" variant="primary" onclick={onaction}>
      {actionLabel}
    </Button>
  {/if}
</div>