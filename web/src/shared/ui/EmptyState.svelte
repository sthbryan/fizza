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
  <h2 class="mb-3 text-2xl font-semibold tracking-tight text-neutral-200">
    {title}
  </h2>
  <p class="max-w-sm text-sm leading-relaxed text-neutral-500">
    {description}
  </p>
  {#if actionLabel && onaction}
    <Button class="mt-7" variant="primary" onclick={onaction}>
      {actionLabel}
    </Button>
  {/if}
</div>