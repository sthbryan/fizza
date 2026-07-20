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
      "group flex flex-1 cursor-pointer flex-col items-center justify-center gap-1 transition-colors",
      active ? "text-white" : "text-neutral-500 active:text-neutral-300"
    );

  const labelClass = "text-[11px] font-mono uppercase tracking-[0.08em]";
</script>

<nav
  aria-label="Primary"
  class="sticky bottom-0 z-40 grid h-16 grid-cols-3 border-t border-neutral-800 bg-black pb-[env(safe-area-inset-bottom)] md:hidden"
>
  <button
    type="button"
    aria-label="Board"
    aria-current={boardActive ? "page" : undefined}
    onclick={(e) => {
      e.preventDefault();
      goBoard();
    }}
    class={itemClass(boardActive)}
  >
    <Kanban size={20} strokeWidth={1.5} />
    <span class={labelClass}>Board</span>
  </button>

  <button
    type="button"
    aria-label="Projects"
    aria-current={projectsActive ? "page" : undefined}
    onclick={(e) => {
      e.preventDefault();
      navigate("/projects");
    }}
    class={itemClass(projectsActive)}
  >
    <Folder size={20} strokeWidth={1.5} />
    <span class={labelClass}>Projects</span>
  </button>

  <button
    type="button"
    aria-label="Stats"
    aria-current={statsActive ? "page" : undefined}
    onclick={(e) => {
      e.preventDefault();
      navigate("/stats");
    }}
    class={itemClass(statsActive)}
  >
    <ChartColumn size={20} strokeWidth={1.5} />
    <span class={labelClass}>Stats</span>
  </button>
</nav>