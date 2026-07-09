<script lang="ts">
  import { createMutation, useQueryClient } from "@tanstack/svelte-query";
  import type { Project } from "@/lib/api";
  import Modal from "@/shared/ui/Modal.svelte";
  import Input from "@/shared/ui/Input.svelte";
  import TextArea from "@/shared/ui/TextArea.svelte";
  import { queryKeys } from "@/lib/api";
  import { showToast } from "@/lib/toast/toast.svelte";
  import { lastBoardHint, rememberBoard } from "@/lib/router/router.svelte";
  import { projectsApi } from "./api";

  interface Props {
    project: Project | null;
    open: boolean;
    onclose: () => void;
  }

  let { project, open, onclose }: Props = $props();

  let name = $state("");
  let description = $state("");
  const queryClient = useQueryClient();

  $effect(() => {
    if (open && project) {
      name = project.name;
      description = project.description?.trim() || "";
    }
  });

  const mutation = createMutation(() => ({
    mutationFn: () => {
      if (!project) throw new Error("No project");
      return projectsApi.update(project.name, {
        name: name.trim(),
        description: description.trim(),
      });
    },
    onSuccess: async (updated, _vars) => {
      const stored = lastBoardHint();
      if (project && stored?.project === project.name) {
        rememberBoard(updated.name, stored.board);
      }
      await queryClient.invalidateQueries({ queryKey: queryKeys.projects });
      await queryClient.invalidateQueries({ queryKey: queryKeys.boards(updated.name) });
      if (project && project.name !== updated.name) {
        await queryClient.invalidateQueries({
          queryKey: queryKeys.boards(project.name),
        });
      }
      showToast(`Project “${updated.name}” updated`);
      onclose();
    },
    onError: (err) => {
      showToast(err instanceof Error ? err.message : String(err), "error");
    },
  }));

  async function submit() {
    if (!name.trim()) {
      showToast("Name is required", "error");
      return;
    }
    await mutation.mutateAsync();
  }
</script>

<Modal
  {open}
  title="Edit project"
  submitLabel="Save"
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
