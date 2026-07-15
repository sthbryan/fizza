<script lang="ts">
  import AppShell from "@/shared/layout/AppShell.svelte";
  import Button from "@/shared/ui/Button.svelte";
  import {
    boardPath,
    lastBoardHint,
    navigate,
  } from "@/lib/router/router.svelte";

  const hint = lastBoardHint();

  function goBoard() {
    if (hint) navigate(boardPath(hint.project, hint.board));
    else navigate("/projects");
  }
</script>

<AppShell>
  <main
    class="flex min-h-0 flex-1 flex-col items-center justify-center px-4 py-12 sm:px-6"
  >
    <div
      class="w-full max-w-md rounded-md border border-neutral-800 bg-neutral-950 px-6 py-10 text-center sm:px-8 sm:py-12"
    >
      <div
        class="mb-4 font-mono text-[10px] uppercase tracking-[0.15em] text-red-500"
      >
        Error 404
      </div>
      <div
        class="mb-4 font-mono text-6xl font-medium tracking-tight text-white sm:text-7xl"
        style="font-family: 'Doto', 'Space Mono', monospace"
      >
        404
      </div>
      <h1 class="mb-2 text-lg font-medium tracking-tight text-neutral-100 sm:text-xl">
        Page not found
      </h1>
      <p class="mb-8 text-sm text-neutral-500">
        This URL is not a fizza page. Check the path or jump back into the app.
      </p>
      <div class="flex flex-col items-stretch gap-2 sm:flex-row sm:justify-center">
        <Button variant="primary" onclick={goBoard}>
          {hint ? "Open board" : "Go to projects"}
        </Button>
        <Button variant="secondary" onclick={() => navigate("/projects")}>
          Projects
        </Button>
        <Button variant="ghost" onclick={() => navigate("/stats")}>
          Stats
        </Button>
      </div>
    </div>
  </main>
</AppShell>