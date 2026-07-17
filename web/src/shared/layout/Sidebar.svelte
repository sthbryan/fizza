<script lang="ts">
  import { cn } from "@/lib/cn";
  import { getRoute, navigate, boardPath, lastBoardHint } from "@/lib/router/router.svelte";

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
      "group flex cursor-pointer flex-col items-center gap-1.5 rounded-md py-2.5 transition-colors",
      active
        ? "text-white"
        : "text-neutral-500 hover:bg-neutral-900 hover:text-neutral-300"
    );

  const labelClass = "text-[9px] font-mono font-medium uppercase tracking-[0.1em]";
</script>

<aside
  class="flex w-20 shrink-0 flex-col items-stretch border-r border-neutral-800 bg-black py-5"
>
  <div
    class="mb-6 flex items-center justify-center font-mono text-sm font-medium tracking-tight text-white"
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
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <rect x="3" y="4" width="7" height="16" rx="1" stroke="currentColor" stroke-width="1.5" />
          <rect x="14" y="4" width="7" height="10" rx="1" stroke="currentColor" stroke-width="1.5" />
        </svg>
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
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <path
            d="M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V7z"
            stroke="currentColor"
            stroke-width="1.5"
          />
        </svg>
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
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <path
            d="M4 19V9M10 19V5M16 19v-7M22 19H2"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
          />
        </svg>
      </span>
      <span class={labelClass}>Stats</span>
    </button>
  </nav>
</aside>