import type { QueryClient } from "@tanstack/svelte-query";

/**
 * Subscribe to server-sent change events and invalidate TanStack Query caches.
 * MCP/CLI/HTTP writes land in the same SQLite events table, so agent task
 * updates show up on the board without replacing REST.
 */
export function subscribeLiveInvalidation(queryClient: QueryClient): () => void {
  const es = new EventSource("/v1/events");

  const onChange = () => {
    void queryClient.invalidateQueries();
  };

  es.addEventListener("change", onChange);

  return () => {
    es.removeEventListener("change", onChange);
    es.close();
  };
}
