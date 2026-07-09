<script lang="ts">
  import { createMutation, useQueryClient } from "@tanstack/svelte-query";
  import Modal from "@/shared/ui/Modal.svelte";
  import Input from "@/shared/ui/Input.svelte";
  import TextArea from "@/shared/ui/TextArea.svelte";
  import Select from "@/shared/ui/Select.svelte";
  import { queryKeys } from "@/lib/api";
  import { showToast } from "@/lib/toast/toast.svelte";
  import { tasksApi } from "./api";

  const PRIORITY_OPTIONS = [
    { value: "low", label: "LOW" },
    { value: "medium", label: "MEDIUM" },
    { value: "high", label: "HIGH" },
    { value: "urgent", label: "URGENT" },
  ];

  interface Props {
    open: boolean;
    project: string;
    board: string;
    columns: { value: string; label: string }[];
    defaultColumn?: string;
    onclose: () => void;
  }

  let {
    open,
    project,
    board,
    columns,
    defaultColumn = "",
    onclose,
  }: Props = $props();

  let title = $state("");
  let description = $state("");
  let column = $state("");
  let priority = $state("medium");
  let due = $state("");
  const queryClient = useQueryClient();

  $effect(() => {
    if (open) {
      title = "";
      description = "";
      column = defaultColumn || columns[0]?.value || "";
      priority = "medium";
      due = "";
    }
  });

  const mutation = createMutation({
    mutationFn: () =>
      tasksApi.create(project, board, {
        title: title.trim(),
        description: description.trim() || undefined,
        column: column || undefined,
        priority,
        due: due || undefined,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: queryKeys.snapshot(project, board),
      });
      showToast("Task added");
      onclose();
    },
    onError: (err) => {
      showToast(err instanceof Error ? err.message : String(err), "error");
    },
  });

  async function submit() {
    if (!title.trim()) {
      showToast("Title is required", "error");
      return;
    }
    await $mutation.mutateAsync();
  }
</script>

<Modal
  {open}
  title="New task"
  submitLabel="Create"
  busy={$mutation.isPending}
  onclose={onclose}
  onsubmit={submit}
>
  <Input label="Title" bind:value={title} placeholder="What needs doing?" />
  <TextArea
    label="Description"
    bind:value={description}
    placeholder="optional details"
  />
  <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
    <Select
      label="Column"
      value={column}
      options={columns}
      onchange={(v) => (column = v)}
      placeholder="Column"
    />
    <Select
      label="Priority"
      value={priority}
      options={PRIORITY_OPTIONS}
      onchange={(v) => (priority = v)}
    />
  </div>
  <Input label="Due date" type="date" bind:value={due} />
</Modal>
