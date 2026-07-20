<script lang="ts">
  import Modal from "./Modal.svelte";

  interface Props {
    open: boolean;
    title: string;
    description?: string;
    confirmLabel?: string;
    cancelLabel?: string;
    busy?: boolean;
    onclose: () => void;
    onconfirm: () => void | Promise<void>;
  }

  let {
    open,
    title,
    description = "",
    confirmLabel = "Confirm",
    cancelLabel = "Cancel",
    busy = false,
    onclose,
    onconfirm,
  }: Props = $props();
</script>

<Modal
  {open}
  {title}
  {busy}
  {cancelLabel}
  submitLabel={confirmLabel}
  submitVariant="destructive"
  onclose={onclose}
  onsubmit={onconfirm}
>
  {#if description}
    <p class="text-base leading-relaxed text-text-secondary">
      {description}
    </p>
  {/if}
</Modal>
