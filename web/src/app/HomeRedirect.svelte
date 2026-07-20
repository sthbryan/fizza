<script lang="ts">
  import { createQuery } from "@tanstack/svelte-query";
  import { queryKeys } from "@/lib/api";
  import { projectsApi } from "@/features/projects/api";
  import { boardApi } from "@/features/board/api";
  import {
    boardPath,
    lastBoardHint,
    navigate,
  } from "@/lib/router/router.svelte";

  const projectsQuery = createQuery(() => ({
    queryKey: queryKeys.projects,
    queryFn: () => projectsApi.list(),
  }));

  $effect(() => {
    if (projectsQuery.isPending) return;
    if (projectsQuery.isError) {
      navigate("/projects", true);
      return;
    }

    const list = projectsQuery.data || [];
    if (!list.length) {
      navigate("/projects", true);
      return;
    }

    const hint = lastBoardHint();
    if (hint && list.some((p) => p.name === hint.project)) {
      void (async () => {
        try {
          const boards = (await boardApi.list(hint.project)) || [];
          const board =
            boards.find((b) => b.name === hint.board)?.name ||
            boards.find((b) => b.is_default)?.name ||
            boards[0]?.name;
          if (board) {
            navigate(boardPath(hint.project, board), true);
            return;
          }
        } catch {
          /* fall through */
        }
        navigate(boardPath(list[0].name, "main"), true);
      })();
      return;
    }

    void (async () => {
      const first = list[0].name;
      try {
        const boards = (await boardApi.list(first)) || [];
        const board =
          boards.find((b) => b.is_default)?.name || boards[0]?.name || "main";
        navigate(boardPath(first, board), true);
      } catch {
        navigate(boardPath(first, "main"), true);
      }
    })();
  });
</script>

<div class="flex h-dvh items-center justify-center text-base text-text-muted">
  Loading…
</div>
