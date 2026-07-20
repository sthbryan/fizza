<script lang="ts">
  import { cn } from "@/lib/cn";
  import { getRoute, navigate, boardPath, lastBoardHint } from "@/lib/router/router.svelte";
  import Kanban from "lucide-svelte/icons/kanban";
  import Folder from "lucide-svelte/icons/folder";
  import ChartColumn from "lucide-svelte/icons/chart-column";

  const route = $derived(getRoute());
  const boardActive = $derived(route.name === "board" || route.name === "home");
  const projectsActive = $derived(route.name === "projects");
  const statsActive = $derived(route.name === "stats");

  function goBoard() {
    if (route.name === "board") return;
    const hint = lastBoardHint();
    if (hint) navigate(boardPath(hint.project, hint.board));
    else navigate("/projects");
  }

  const itemClass = (active: boolean) =>
    cn(
      "group flex min-h-11 cursor-pointer flex-col items-center gap-1.5 py-2.5 transition-colors",
      active
        ? "text-white"
        : "text-neutral-500 hover:text-neutral-300"
    );

  const labelClass = "text-[11px] font-mono uppercase tracking-[0.08em]";
</script>

<aside
  class="hidden w-20 shrink-0 flex-col items-stretch border-r border-neutral-800 bg-black py-5 md:flex"
>
  <div
    class="mb-8 flex items-center justify-center font-display text-lg tracking-tight text-white"
  >
    fizza
  </div>

  <nav class="flex flex-1 flex-col items-stretch gap-1 px-3">
    <button
      type="button"
      title="Board"
      onclick={(e) => {
        e.preventDefault();
        goBoard();
      }}
      class={itemClass(boardActive)}
    >
      <span class="relative">
        {#if boardActive}
          <span
            class="absolute -left-1.5 top-1/2 h-1 w-1 -translate-y-1/2 rounded-full bg-white"
          ></span>
        {/if}
        <Kanban size={20} strokeWidth={1.5} />
      </span>
      <span class={labelClass}>Board</span>
    </button>

    <button
      type="button"
      title="Projects"
      onclick={(e) => {
        e.preventDefault();
        navigate("/projects");
      }}
      class={itemClass(projectsActive)}
    >
      <span class="relative">
        {#if projectsActive}
          <span
            class="absolute -left-1.5 top-1/2 h-1 w-1 -translate-y-1/2 rounded-full bg-white"
          ></span>
        {/if}
        <Folder size={20} strokeWidth={1.5} />
      </span>
      <span class={labelClass}>Projects</span>
    </button>

    <button
      type="button"
      title="Stats"
      onclick={(e) => {
        e.preventDefault();
        navigate("/stats");
      }}
      class={itemClass(statsActive)}
    >
      <span class="relative">
        {#if statsActive}
          <span
            class="absolute -left-1.5 top-1/2 h-1 w-1 -translate-y-1/2 rounded-full bg-white"
          ></span>
        {/if}
        <ChartColumn size={20} strokeWidth={1.5} />
      </span>
      <span class={labelClass}>Stats</span>
    </button>
  </nav>
</aside>