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
    <div class="w-full max-w-md px-4 py-10 text-center sm:px-0 sm:py-12">
      <div
        class="mb-4 text-[11px] font-mono uppercase tracking-[0.08em] text-[var(--color-accent)]"
      >
        [ERROR] 404
      </div>
      <div
        class="mb-4 font-display text-6xl leading-none tracking-tight text-white sm:text-7xl"
      >
        404
      </div>
      <h1 class="mb-2 text-base tracking-tight text-neutral-400">
        Page not found
      </h1>
      <p class="mb-8 text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-600">
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
