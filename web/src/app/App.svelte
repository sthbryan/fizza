<script lang="ts">
  import { onMount } from "svelte";
  import { QueryClient, QueryClientProvider } from "@tanstack/svelte-query";
  import { getRoute } from "@/lib/router/router.svelte";
  import { subscribeLiveInvalidation } from "@/lib/api";
  import HomeRedirect from "./HomeRedirect.svelte";
  import NotFoundPage from "./NotFoundPage.svelte";
  import ProjectsPage from "@/features/projects/ProjectsPage.svelte";
  import StatsPage from "@/features/stats/StatsPage.svelte";
  import BoardPage from "@/features/board/BoardPage.svelte";

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
  {:else if route.name === "stats"}
    <StatsPage />
  {:else if route.name === "board"}
    {#key `${route.project}/${route.board}`}
      <BoardPage project={route.project} board={route.board} />
    {/key}
  {:else}
    <NotFoundPage />
  {/if}
</QueryClientProvider>
