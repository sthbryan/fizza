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

  function goNewTask() {
    if (route.name === "board") {
      window.dispatchEvent(new CustomEvent("fizza:new-task"));
      return;
    }
    const hint = lastBoardHint();
    if (hint) {
      navigate(boardPath(hint.project, hint.board));
      queueMicrotask(() =>
        window.dispatchEvent(new CustomEvent("fizza:new-task"))
      );
    } else {
      navigate("/projects");
    }
  }
</script>

<aside
  class="flex w-16 shrink-0 flex-col items-center border-r border-[var(--color-border-subtle)] bg-[var(--color-bg)] py-4 sm:w-[4.5rem] sm:py-5"
>
  <div
    class="mb-5 flex h-11 w-11 cursor-default items-center justify-center rounded-2xl bg-[var(--color-accent)] text-base font-bold text-white sm:mb-7 sm:h-12 sm:w-12"
  >
    f
  </div>

  <nav class="flex flex-1 flex-col items-center gap-1.5">
    <a
      href={lastBoardHint()
        ? boardPath(lastBoardHint()!.project, lastBoardHint()!.board)
        : "/projects"}
      title="Board"
      class={cn(
        "flex h-11 w-11 items-center justify-center rounded-2xl transition sm:h-12 sm:w-12",
        boardActive
          ? "bg-[var(--color-bg-hover)] text-[var(--color-text)]"
          : "text-[var(--color-text-muted)] hover:bg-[var(--color-bg-soft)] hover:text-[var(--color-text-secondary)]"
      )}
      onclick={(e) => {
        e.preventDefault();
        goBoard();
      }}
    >
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <rect
          x="3"
          y="4"
          width="7"
          height="16"
          rx="2"
          stroke="currentColor"
          stroke-width="1.6"
        />
        <rect
          x="14"
          y="4"
          width="7"
          height="10"
          rx="2"
          stroke="currentColor"
          stroke-width="1.6"
        />
      </svg>
    </a>
    <a
      href="/projects"
      title="Projects"
      class={cn(
        "flex h-11 w-11 items-center justify-center rounded-2xl transition sm:h-12 sm:w-12",
        projectsActive
          ? "bg-[var(--color-bg-hover)] text-[var(--color-text)]"
          : "text-[var(--color-text-muted)] hover:bg-[var(--color-bg-soft)] hover:text-[var(--color-text-secondary)]"
      )}
      onclick={(e) => {
        e.preventDefault();
        navigate("/projects");
      }}
    >
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path
          d="M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V7z"
          stroke="currentColor"
          stroke-width="1.6"
        />
      </svg>
    </a>
    <a
      href="/stats"
      title="Stats"
      class={cn(
        "flex h-11 w-11 items-center justify-center rounded-2xl transition sm:h-12 sm:w-12",
        statsActive
          ? "bg-[var(--color-bg-hover)] text-[var(--color-text)]"
          : "text-[var(--color-text-muted)] hover:bg-[var(--color-bg-soft)] hover:text-[var(--color-text-secondary)]"
      )}
      onclick={(e) => {
        e.preventDefault();
        navigate("/stats");
      }}
    >
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path
          d="M4 19V9M10 19V5M16 19v-7M22 19H2"
          stroke="currentColor"
          stroke-width="1.6"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
    </a>
  </nav>

  <button
    type="button"
    title="New task"
    onclick={goNewTask}
    class="mt-auto flex h-11 w-11 cursor-pointer items-center justify-center rounded-full border border-[var(--color-border)] text-lg text-[var(--color-text-secondary)] transition hover:border-[var(--color-accent)] hover:text-[var(--color-accent)] sm:h-12 sm:w-12"
  >
    +
  </button>
</aside>
