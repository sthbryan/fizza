<script lang="ts">
  import type { Snippet } from "svelte";
  import Sidebar from "./Sidebar.svelte";
  import Toast from "@/shared/ui/Toast.svelte";

  interface Props {
    children: Snippet;
  }

  let { children }: Props = $props();
</script>

<div class="flex h-full min-h-dvh bg-[var(--color-bg)]">
  <Sidebar />
  <div class="flex min-w-0 flex-1 flex-col">
    {@render children()}
  </div>
  <Toast />
</div>
