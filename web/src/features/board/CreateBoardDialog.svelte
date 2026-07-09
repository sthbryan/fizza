<script lang="ts">
  import { createMutation, useQueryClient } from "@tanstack/svelte-query";
  import Modal from "@/shared/ui/Modal.svelte";
  import Input from "@/shared/ui/Input.svelte";
  import { queryKeys } from "@/lib/api";
  import { showToast } from "@/lib/toast/toast.svelte";
  import { boardPath, navigate, rememberBoard } from "@/lib/router/router.svelte";
  import { boardApi } from "./api";

  interface Props {
    open: boolean;
    project: string;
    onclose: () => void;
  }

  let { open, project, onclose }: Props = $props();

  let name = $state("");
  let columns = $state("todo,in_progress,in_review,done");
  const queryClient = useQueryClient();

  $effect(() => {
    if (open) {
      name = "";
      columns = "todo,in_progress,in_review,done";
    }
  });

  const mutation = createMutation({
    mutationFn: () =>
      boardApi.create(project, name.trim(), columns.trim() || undefined),
    onSuccess: async (board) => {
      await queryClient.invalidateQueries({
        queryKey: queryKeys.boards(project),
      });
      rememberBoard(project, board.name);
      showToast(`Board “${board.name}” created`);
      onclose();
      navigate(boardPath(project, board.name));
    },
    onError: (err) => {
      showToast(err instanceof Error ? err.message : String(err), "error");
    },
  });

  async function submit() {
    if (!name.trim()) {
      showToast("Name is required", "error");
      return;
    }
    await $mutation.mutateAsync();
  }
</script>

<Modal
  {open}
  title="New board"
  submitLabel="Create"
  busy={$mutation.isPending}
  onclose={onclose}
  onsubmit={submit}
>
  <Input label="Name" bind:value={name} placeholder="sprint-1" maxlength={64} />
  <Input
    label="Columns"
    bind:value={columns}
    placeholder="todo,in_progress,in_review,done"
  />
</Modal>
