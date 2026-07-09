<script lang="ts">
  import { createMutation, useQueryClient } from "@tanstack/svelte-query";
  import Modal from "@/shared/ui/Modal.svelte";
  import Input from "@/shared/ui/Input.svelte";
  import { queryKeys } from "@/lib/api";
  import { showToast } from "@/lib/toast/toast.svelte";
  import { boardApi } from "./api";

  interface Props {
    open: boolean;
    project: string;
    board: string;
    onclose: () => void;
  }

  let { open, project, board, onclose }: Props = $props();

  let name = $state("");
  const queryClient = useQueryClient();

  $effect(() => {
    if (open) name = "";
  });

  const mutation = createMutation({
    mutationFn: () => boardApi.createColumn(project, board, name.trim()),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: queryKeys.snapshot(project, board),
      });
      showToast(`Column “${name.trim()}” added`);
      onclose();
    },
    onError: (err) => {
      showToast(err instanceof Error ? err.message : String(err), "error");
    },
  });

  async function submit() {
    if (!name.trim()) {
      showToast("Column name is required", "error");
      return;
    }
    await $mutation.mutateAsync();
  }
</script>

<Modal
  {open}
  title="New column"
  submitLabel="Create"
  busy={$mutation.isPending}
  onclose={onclose}
  onsubmit={submit}
>
  <Input label="Name" bind:value={name} placeholder="blocked" maxlength={32} />
</Modal>
