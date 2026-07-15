<script lang="ts">
  import type { Snippet } from "svelte";
  import Sidebar from "./Sidebar.svelte";
  import Status from "@/shared/ui/Status.svelte";

  interface Props {
    children: Snippet;
  }

  let { children }: Props = $props();
</script>

<div class="flex h-full min-h-dvh bg-black">
  <Sidebar />
  <div class="flex min-w-0 flex-1 flex-col">
    <Status />
    {@render children()}
  </div>
</div>