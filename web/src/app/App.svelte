<script lang="ts">
  import { onMount } from "svelte";
  import { QueryClient, QueryClientProvider } from "@tanstack/svelte-query";
  import { getRoute } from "@/lib/router/router.svelte";
  import { subscribeLiveInvalidation } from "@/lib/api";
  import HomeRedirect from "./HomeRedirect.svelte";
  import ProjectsPage from "@/features/projects/ProjectsPage.svelte";
  import BoardPage from "@/features/board/BoardPage.svelte";
  import AppShell from "@/shared/layout/AppShell.svelte";

  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 5_000,
        retry: 1,
        refetchOnWindowFocus: false,
      },
    },
  });

  onMount(() => subscribeLiveInvalidation(queryClient));

  const route = $derived(getRoute());
</script>

<QueryClientProvider client={queryClient}>
  {#if route.name === "home"}
    <HomeRedirect />
  {:else if route.name === "projects"}
    <ProjectsPage />
  {:else if route.name === "board"}
    {#key `${route.project}/${route.board}`}
      <BoardPage project={route.project} board={route.board} />
    {/key}
  {:else}
    <AppShell>
      <div class="flex flex-1 flex-col items-center justify-center gap-3 p-8">
        <h1 class="text-lg font-semibold">Not found</h1>
        <p class="text-sm text-[var(--color-text-muted)]">
          No page at this URL.
        </p>
        <a
          href="/projects"
          class="text-sm text-[var(--color-accent)] hover:underline"
        >
          Go to projects
        </a>
      </div>
    </AppShell>
  {/if}
</QueryClientProvider>
