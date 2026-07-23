<script lang="ts">
  import { createMutation, useQueryClient } from "@tanstack/svelte-query";
  import Modal from "@/shared/ui/Modal.svelte";
  import Input from "@/shared/ui/Input.svelte";
  import TextArea from "@/shared/ui/TextArea.svelte";
  import { queryKeys } from "@/lib/api";
  import { showStatus } from "@/lib/status/status.svelte";
  import { boardPath, navigate, rememberBoard } from "@/lib/router/router.svelte";
  import { projectsApi } from "./api";

  interface Props {
    open: boolean;
    onclose: () => void;
  }

  let { open, onclose }: Props = $props();

  let name = $state("");
  let description = $state("");
  const queryClient = useQueryClient();

  $effect(() => {
    if (open) {
      name = "";
      description = "";
    }
  });

  const mutation = createMutation(() => ({
    mutationFn: () => projectsApi.create(name.trim(), description.trim()),
    onSuccess: async (project) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.projects });
      rememberBoard(project.name, "main");
      showStatus(`Project “${project.name}” created`);
      onclose();
      navigate(boardPath(project.name, "main"));
    },
    onError: (err) => {
      showStatus(err instanceof Error ? err.message : String(err), "error");
    },
  }));

  async function submit() {
    if (!name.trim()) {
      showStatus("Name is required", "error");
      return;
    }
    await mutation.mutateAsync();
  }
</script>

<Modal
  {open}
  title="New project"
  submitLabel="Create"
  busy={mutation.isPending}
  onclose={onclose}
  onsubmit={submit}
>
  <Input label="Name" bind:value={name} placeholder="myapp" maxlength={64} />
  <TextArea
    label="Description"
    bind:value={description}
    placeholder="optional"
  />
</Modal>
