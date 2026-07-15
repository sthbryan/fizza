<script lang="ts">
  import { createQuery } from "@tanstack/svelte-query";
  import AppShell from "@/shared/layout/AppShell.svelte";
  import Select from "@/shared/ui/Select.svelte";
  import EmptyState from "@/shared/ui/EmptyState.svelte";
  import { fizzaApi, queryKeys, type Board, type Project } from "@/lib/api";
  import { lastBoardHint, navigate } from "@/lib/router/router.svelte";
  import { statsApi } from "./api";
  import ProgressRing from "./ProgressRing.svelte";
  import HBarChart from "./HBarChart.svelte";
  import DayChart from "./DayChart.svelte";
  import {
    columnColor,
    formatStatusLabel,
    pct,
    priorityColor,
  } from "./utils";

  const hint = lastBoardHint();
  let projectFilter = $state(hint?.project ?? "");
  let boardFilter = $state("");

  const projectsQuery = createQuery(() => ({
    queryKey: queryKeys.projects,
    queryFn: () => fizzaApi.listProjects(),
  }));

  const boardsQuery = createQuery(() => ({
    queryKey: queryKeys.boards(projectFilter || "__none__"),
    queryFn: () =>
      projectFilter
        ? fizzaApi.listBoards(projectFilter)
        : Promise.resolve([] as Board[]),
    enabled: !!projectFilter,
  }));

  const statsQuery = createQuery(() => ({
    queryKey: queryKeys.stats(projectFilter, boardFilter),
    queryFn: () =>
      statsApi.get(projectFilter || undefined, boardFilter || undefined),
  }));

  const projectOptions = $derived([
    { value: "", label: "ALL PROJECTS" },
    ...((projectsQuery.data as Project[] | undefined) ?? []).map((p) => ({
      value: p.name,
      label: formatStatusLabel(p.name),
    })),
  ]);

  const boardOptions = $derived([
    {
      value: "",
      label: projectFilter ? "ALL BOARDS" : "PICK A PROJECT FIRST",
    },
    ...((boardsQuery.data as Board[] | undefined) ?? []).map((b) => ({
      value: b.name,
      label: formatStatusLabel(b.name),
    })),
  ]);

  const stats = $derived(statsQuery.data);
  const donePct = $derived(
    stats ? pct(stats.totals.done, stats.totals.tasks) : 0
  );

  function onProjectChange(v: string) {
    projectFilter = v;
    boardFilter = "";
  }

  function scopeLabel() {
    if (projectFilter && boardFilter) {
      return `${formatStatusLabel(projectFilter)} / ${formatStatusLabel(boardFilter)}`;
    }
    if (projectFilter) return formatStatusLabel(projectFilter);
    return "All projects";
  }
</script>

<AppShell>
  <header
    class="border-b border-neutral-800 bg-black px-4 py-4 sm:px-6 sm:py-5"
  >
    <div
      class="flex flex-col gap-3.5 lg:flex-row lg:items-end lg:justify-between"
    >
      <div class="min-w-0">
        <div class="mb-1.5 text-[10px] font-mono uppercase tracking-[0.1em] text-neutral-500">
          fizza / stats
        </div>
        <div class="flex flex-wrap items-baseline gap-2 sm:gap-3">
          <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl text-white">
            Progress
          </h1>
          <span class="text-sm text-neutral-500">
            {scopeLabel()}
          </span>
        </div>
      </div>
      <div class="flex flex-wrap items-end gap-3 sm:gap-4">
        <div class="w-full min-w-[10rem] sm:w-48">
          <Select
            label="Project"
            size="sm"
            value={projectFilter}
            options={projectOptions}
            onchange={onProjectChange}
          />
        </div>
        <div class="w-full min-w-[10rem] sm:w-48">
          <Select
            label="Board"
            size="sm"
            value={boardFilter}
            options={boardOptions}
            onchange={(v) => (boardFilter = v)}
            disabled={!projectFilter}
          />
        </div>
      </div>
    </div>
    {#if stats}
      {@const totals = stats.totals}
      <div
        class="mt-4 flex flex-wrap items-baseline gap-x-5 gap-y-2 font-mono text-sm"
      >
        {#if !projectFilter}
          <span>
            <span class="text-base font-semibold tabular-nums text-white">{totals.projects}</span>
            <span class="ml-1.5 text-[10px] uppercase tracking-[0.1em] text-neutral-500">projects</span>
          </span>
        {/if}
        {#if !boardFilter}
          <span>
            <span class="text-base font-semibold tabular-nums text-white">{totals.boards}</span>
            <span class="ml-1.5 text-[10px] uppercase tracking-[0.1em] text-neutral-500">boards</span>
          </span>
        {/if}
        <span>
          <span class="text-base font-semibold tabular-nums text-white">{totals.tasks}</span>
          <span class="ml-1.5 text-[10px] uppercase tracking-[0.1em] text-neutral-500">active</span>
        </span>
        <span>
          <span class="text-base font-semibold tabular-nums text-white">{totals.open}</span>
          <span class="ml-1.5 text-[10px] uppercase tracking-[0.1em] text-neutral-500">open</span>
        </span>
        <span>
          <span class="text-base font-semibold tabular-nums text-emerald-500">{totals.done}</span>
          <span class="ml-1.5 text-[10px] uppercase tracking-[0.1em] text-neutral-500">done</span>
        </span>
        <span>
          <span
            class="text-base font-semibold tabular-nums"
            class:text-red-500={totals.overdue > 0}
            class:text-white={totals.overdue === 0}
          >{totals.overdue}</span>
          <span class="ml-1.5 text-[10px] uppercase tracking-[0.1em] text-neutral-500">overdue</span>
        </span>
        <span>
          <span class="text-base font-semibold tabular-nums text-white">{totals.archived ?? 0}</span>
          <span class="ml-1.5 text-[10px] uppercase tracking-[0.1em] text-neutral-500">archived</span>
        </span>
      </div>
    {/if}
  </header>

  <main class="min-h-0 flex-1 overflow-y-auto">
    {#if statsQuery.isPending}
      <div class="p-8 text-base text-[var(--color-text-muted)]">Loading…</div>
    {:else if statsQuery.isError}
      <div class="p-8 text-base text-[var(--color-danger)]">
        {statsQuery.error.message}
      </div>
    {:else if stats}
      {@const t = stats.totals}
      {#if t.projects === 0 && !projectFilter}
        <EmptyState
          title="Nothing to measure yet"
          description="Create a project and a few tasks to see progress charts."
          actionLabel="Go to projects"
          onaction={() => navigate("/projects")}
        />
      {:else}
        <div class="space-y-5 p-4 sm:space-y-6 sm:p-6">
          <div class="grid grid-cols-1 gap-4 lg:grid-cols-3 lg:gap-5">
            <div
              class="flex flex-col items-center justify-center rounded-md border border-neutral-800 bg-neutral-950 p-6"
            >
              <ProgressRing
                value={donePct}
                label="Completion"
                sublabel="{t.done} of {t.tasks} active done"
              />
            </div>

            <div
              class="rounded-md border border-neutral-800 bg-neutral-950 p-5 sm:p-6"
            >
              <h2 class="mb-4 text-[10px] font-mono font-medium uppercase tracking-[0.1em] text-neutral-400">
                By priority
              </h2>
              <HBarChart
                rows={stats.by_priority}
                colorFor={priorityColor}
                labelFor={formatStatusLabel}
                emptyLabel="No tasks yet"
              />
            </div>

            <div
              class="rounded-md border border-neutral-800 bg-neutral-950 p-5 sm:p-6"
            >
              <h2 class="mb-4 text-[10px] font-mono font-medium uppercase tracking-[0.1em] text-neutral-400">
                By column
              </h2>
              <HBarChart
                rows={stats.by_column}
                colorFor={columnColor}
                labelFor={formatStatusLabel}
                emptyLabel="No tasks yet"
              />
            </div>
          </div>

          <div class="grid grid-cols-1 gap-4 lg:grid-cols-2 lg:gap-5">
            <div
              class="rounded-md border border-neutral-800 bg-neutral-950 p-5 sm:p-6"
            >
              <h2 class="mb-1 text-[10px] font-mono font-medium uppercase tracking-[0.1em] text-neutral-400">
                Tasks created
              </h2>
              <p class="mb-4 text-xs text-neutral-500">
                Last 30 days
              </p>
              <DayChart
                rows={stats.created_by_day}
                color="var(--color-text-display)"
              />
            </div>
            <div
              class="rounded-md border border-neutral-800 bg-neutral-950 p-5 sm:p-6"
            >
              <h2 class="mb-1 text-[10px] font-mono font-medium uppercase tracking-[0.1em] text-neutral-400">
                Activity
              </h2>
              <p class="mb-4 text-xs text-neutral-500">
                Creates, updates, moves · last 30 days
              </p>
              <DayChart
                rows={stats.activity_by_day}
                color="var(--color-text-display)"
              />
            </div>
          </div>

          {#if stats.by_project?.length}
            <div
              class="rounded-md border border-neutral-800 bg-neutral-950 p-5 sm:p-6"
            >
              <h2 class="mb-4 text-[10px] font-mono font-medium uppercase tracking-[0.1em] text-neutral-400">
                By project
              </h2>
              <div class="overflow-x-auto">
                <table class="w-full min-w-[28rem] text-left text-sm">
                  <thead>
                    <tr
                      class="border-b border-neutral-800 text-[10px] font-mono font-medium uppercase tracking-[0.1em] text-neutral-500"
                    >
                      <th class="pb-2.5 pr-3">Project</th>
                      <th class="pb-2.5 pr-3">Boards</th>
                      <th class="pb-2.5 pr-3">Active</th>
                      <th class="pb-2.5 pr-3">Done</th>
                      <th class="pb-2.5 pr-3">Open</th>
                      <th class="pb-2.5 pr-3">Overdue</th>
                      <th class="pb-2.5 pr-3">Archived</th>
                      <th class="pb-2.5">Progress</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each stats.by_project as row (row.name)}
                      {@const p = pct(row.done, row.tasks)}
                      <tr
                        class="border-b border-neutral-900 last:border-0"
                      >
                        <td class="py-3 pr-3 text-neutral-200"
                          >{formatStatusLabel(row.name)}</td
                        >
                        <td class="py-3 pr-3 font-mono tabular-nums text-neutral-500"
                          >{row.boards}</td
                        >
                        <td class="py-3 pr-3 font-mono tabular-nums text-white">{row.tasks}</td>
                        <td class="py-3 pr-3 font-mono tabular-nums text-emerald-500"
                          >{row.done}</td
                        >
                        <td class="py-3 pr-3 font-mono tabular-nums text-neutral-200">{row.open}</td>
                        <td
                          class="py-3 pr-3 font-mono tabular-nums"
                          class:text-red-500={row.overdue > 0}
                          class:text-neutral-200={row.overdue === 0}
                        >{row.overdue}</td>
                        <td class="py-3 pr-3 font-mono tabular-nums text-neutral-500"
                          >{row.archived ?? 0}</td
                        >
                        <td class="py-3">
                          <div class="flex min-w-[6rem] items-center gap-2">
                            <div
                              class="h-px flex-1 bg-neutral-800"
                            >
                              <div
                                class="h-full bg-emerald-500"
                                style:width="{p}%"
                              ></div>
                            </div>
                            <span
                              class="w-8 shrink-0 text-right font-mono text-xs tabular-nums text-neutral-400"
                              >{p}%</span
                            >
                          </div>
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            </div>
          {/if}

          {#if stats.by_board?.length}
            <div
              class="rounded-md border border-neutral-800 bg-neutral-950 p-5 sm:p-6"
            >
              <h2 class="mb-4 text-[10px] font-mono font-medium uppercase tracking-[0.1em] text-neutral-400">
                By board
              </h2>
              <div class="overflow-x-auto">
                <table class="w-full min-w-[28rem] text-left text-sm">
                  <thead>
                    <tr
                      class="border-b border-neutral-800 text-[10px] font-mono font-medium uppercase tracking-[0.1em] text-neutral-500"
                    >
                      {#if !projectFilter}
                        <th class="pb-2.5 pr-3">Project</th>
                      {/if}
                      <th class="pb-2.5 pr-3">Board</th>
                      <th class="pb-2.5 pr-3">Active</th>
                      <th class="pb-2.5 pr-3">Done</th>
                      <th class="pb-2.5 pr-3">Open</th>
                      <th class="pb-2.5 pr-3">Overdue</th>
                      <th class="pb-2.5 pr-3">Archived</th>
                      <th class="pb-2.5">Progress</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each stats.by_board as row (`${row.project}/${row.name}`)}
                      {@const p = pct(row.done, row.tasks)}
                      <tr
                        class="border-b border-neutral-900 last:border-0"
                      >
                        {#if !projectFilter}
                          <td
                            class="py-3 pr-3 text-neutral-400"
                            >{formatStatusLabel(row.project)}</td
                          >
                        {/if}
                        <td class="py-3 pr-3 text-neutral-200"
                          >{formatStatusLabel(row.name)}</td
                        >
                        <td class="py-3 pr-3 font-mono tabular-nums text-white">{row.tasks}</td>
                        <td class="py-3 pr-3 font-mono tabular-nums text-emerald-500"
                          >{row.done}</td
                        >
                        <td class="py-3 pr-3 font-mono tabular-nums text-neutral-200">{row.open}</td>
                        <td
                          class="py-3 pr-3 font-mono tabular-nums"
                          class:text-red-500={row.overdue > 0}
                          class:text-neutral-200={row.overdue === 0}
                        >{row.overdue}</td>
                        <td class="py-3 pr-3 font-mono tabular-nums text-neutral-500"
                          >{row.archived ?? 0}</td
                        >
                        <td class="py-3">
                          <div class="flex min-w-[6rem] items-center gap-2">
                            <div
                              class="h-px flex-1 bg-neutral-800"
                            >
                              <div
                                class="h-full bg-emerald-500"
                                style:width="{p}%"
                              ></div>
                            </div>
                            <span
                              class="w-8 shrink-0 text-right font-mono text-xs tabular-nums text-neutral-400"
                              >{p}%</span
                            >
                          </div>
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            </div>
          {/if}
        </div>
      {/if}
    {/if}
  </main>
</AppShell>
