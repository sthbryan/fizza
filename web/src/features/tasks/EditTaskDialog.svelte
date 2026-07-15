<script lang="ts">
  import { createMutation, useQueryClient } from "@tanstack/svelte-query";
  import type { Task } from "@/lib/api";
  import Modal from "@/shared/ui/Modal.svelte";
  import Input from "@/shared/ui/Input.svelte";
  import TextArea from "@/shared/ui/TextArea.svelte";
  import Select from "@/shared/ui/Select.svelte";
  import { queryKeys } from "@/lib/api";
  import { showStatus } from "@/lib/status/status.svelte";
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
    task: Task | null;
    columns: { value: string; label: string }[];
    onclose: () => void;
  }

  let { open, project, board, task, columns, onclose }: Props = $props();

  let title = $state("");
  let description = $state("");
  let column = $state("");
  let priority = $state("medium");
  let due = $state("");
  const queryClient = useQueryClient();

  $effect(() => {
    if (open && task) {
      title = task.title;
      description = task.description || "";
      column = task.status;
      priority = String(task.priority || "medium").toLowerCase();
      due = task.due_date ? String(task.due_date).slice(0, 10) : "";
    }
  });

  const mutation = createMutation(() => ({
    mutationFn: async () => {
      if (!task) throw new Error("No task");
      const patch: {
        title: string;
        desc: string;
        priority: string;
        due?: string;
        clear_due?: boolean;
      } = {
        title: title.trim(),
        desc: description,
        priority,
      };
      if (due) patch.due = due;
      else patch.clear_due = true;

      await tasksApi.update(task.id, patch);
      if (column && column !== task.status) {
        await tasksApi.move(task.id, { project, board, column });
      }
      return task.id;
    },
    onSuccess: async (id) => {
      await queryClient.invalidateQueries({
        queryKey: queryKeys.snapshot(project, board),
      });
      showStatus(`Task #${id} updated`);
      onclose();
    },
    onError: (err) => {
      showStatus(err instanceof Error ? err.message : String(err), "error");
    },
  }));

  async function submit() {
    if (!title.trim()) {
      showStatus("Title is required", "error");
      return;
    }
    await mutation.mutateAsync();
  }
</script>

<Modal
  {open}
  title={task ? `Edit #${task.id}` : "Edit task"}
  submitLabel="Save"
  busy={mutation.isPending}
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
