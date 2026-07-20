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
  import SegmentedBar from "@/shared/ui/SegmentedBar.svelte";
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
      class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between"
    >
      <div class="min-w-0">
        <div class="mb-2 text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-500">
          fizza / stats
        </div>
        <div class="flex flex-wrap items-baseline gap-3">
          {#if stats}
            <span class="font-display text-4xl tabular-nums leading-none tracking-tight text-white sm:text-5xl">
              {donePct}
            </span>
            <span class="text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-500">
              % done
            </span>
          {/if}
          <h1 class="text-base tracking-tight text-white sm:text-[18px]">
            Progress
          </h1>
          <span class="text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-500">
            {scopeLabel()}
          </span>
        </div>
      </div>
      <div class="flex w-full flex-col gap-3 sm:flex-row sm:items-end sm:gap-4 lg:w-auto">
        <div class="w-full min-w-0 sm:w-48">
          <Select
            label="Project"
            size="sm"
            value={projectFilter}
            options={projectOptions}
            onchange={onProjectChange}
          />
        </div>
        <div class="w-full min-w-0 sm:w-48">
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
        class="mt-4 flex flex-wrap items-baseline gap-x-5 gap-y-2 font-mono text-base"
      >
        {#if !projectFilter}
          <span>
            <span class="tabular-nums text-white">{totals.projects}</span>
            <span class="ml-1.5 text-[11px] uppercase tracking-[0.08em] text-neutral-500">projects</span>
          </span>
        {/if}
        {#if !boardFilter}
          <span>
            <span class="tabular-nums text-white">{totals.boards}</span>
            <span class="ml-1.5 text-[11px] uppercase tracking-[0.08em] text-neutral-500">boards</span>
          </span>
        {/if}
        <span>
          <span class="tabular-nums text-white">{totals.tasks}</span>
          <span class="ml-1.5 text-[11px] uppercase tracking-[0.08em] text-neutral-500">active</span>
        </span>
        <span>
          <span class="tabular-nums text-white">{totals.open}</span>
          <span class="ml-1.5 text-[11px] uppercase tracking-[0.08em] text-neutral-500">open</span>
        </span>
        <span>
          <span class="tabular-nums text-[var(--color-ok)]">{totals.done}</span>
          <span class="ml-1.5 text-[11px] uppercase tracking-[0.08em] text-neutral-500">done</span>
        </span>
        <span>
          <span
            class="tabular-nums"
            class:text-[var(--color-accent)]={totals.overdue > 0}
            class:text-white={totals.overdue === 0}
          >{totals.overdue}</span>
          <span class="ml-1.5 text-[11px] uppercase tracking-[0.08em] text-neutral-500">overdue</span>
        </span>
        <span>
          <span class="tabular-nums text-white">{totals.archived ?? 0}</span>
          <span class="ml-1.5 text-[11px] uppercase tracking-[0.08em] text-neutral-500">archived</span>
        </span>
      </div>
    {/if}
  </header>

  <main class="min-h-0 flex-1 overflow-y-auto">
    {#if statsQuery.isPending}
      <div class="p-8 text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-500">[LOADING]</div>
    {:else if statsQuery.isError}
      <div class="p-8 text-[11px] font-mono uppercase tracking-[0.08em] text-[var(--color-accent)]">
        [ERROR] {statsQuery.error.message}
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
        <div class="space-y-8 p-4 sm:space-y-10 sm:p-6">
          <div class="grid grid-cols-1 gap-8 lg:grid-cols-3 lg:gap-10">
            <div class="flex flex-col items-center justify-center py-2">
              <ProgressRing
                value={donePct}
                label="Completion"
                sublabel="{t.done} of {t.tasks} active done"
              />
            </div>

            <div class="min-w-0">
              <h2 class="mb-4 text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-400">
                By priority
              </h2>
              <HBarChart
                rows={stats.by_priority}
                colorFor={priorityColor}
                labelFor={formatStatusLabel}
                emptyLabel="No tasks yet"
              />
            </div>

            <div class="min-w-0">
              <h2 class="mb-4 text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-400">
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

          <div class="grid grid-cols-1 gap-8 border-t border-neutral-800 pt-8 lg:grid-cols-2 lg:gap-10">
            <div class="min-w-0">
              <h2 class="mb-1 text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-400">
                Tasks created
              </h2>
              <p class="mb-4 text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-500">
                Last 30 days
              </p>
              <DayChart
                rows={stats.created_by_day}
                color="var(--color-text-display)"
              />
            </div>
            <div class="min-w-0">
              <h2 class="mb-1 text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-400">
                Activity
              </h2>
              <p class="mb-4 text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-500">
                Creates, updates, moves · last 30 days
              </p>
              <DayChart
                rows={stats.activity_by_day}
                color="var(--color-text-display)"
              />
            </div>
          </div>

          {#if stats.by_project?.length}
            <div class="border-t border-neutral-800 pt-8">
              <h2 class="mb-4 text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-400">
                By project
              </h2>
              <div class="overflow-x-auto">
                <table class="w-full min-w-[28rem] text-left text-base">
                  <thead>
                    <tr
                      class="border-b border-neutral-700 text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-500"
                    >
                      <th class="pb-3 pr-3 font-normal">Project</th>
                      <th class="pb-3 pr-3 font-normal">Boards</th>
                      <th class="pb-3 pr-3 font-normal">Active</th>
                      <th class="pb-3 pr-3 font-normal">Done</th>
                      <th class="pb-3 pr-3 font-normal">Open</th>
                      <th class="pb-3 pr-3 font-normal">Overdue</th>
                      <th class="pb-3 pr-3 font-normal">Archived</th>
                      <th class="pb-3 font-normal">Progress</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each stats.by_project as row (row.name)}
                      {@const p = pct(row.done, row.tasks)}
                      <tr class="border-b border-neutral-800 last:border-0">
                        <td class="py-3 pr-3 text-neutral-200"
                          >{formatStatusLabel(row.name)}</td
                        >
                        <td class="py-3 pr-3 font-mono tabular-nums text-neutral-500"
                          >{row.boards}</td
                        >
                        <td class="py-3 pr-3 font-mono tabular-nums text-white">{row.tasks}</td>
                        <td class="py-3 pr-3 font-mono tabular-nums text-[var(--color-ok)]"
                          >{row.done}</td
                        >
                        <td class="py-3 pr-3 font-mono tabular-nums text-neutral-200">{row.open}</td>
                        <td
                          class="py-3 pr-3 font-mono tabular-nums"
                          class:text-[var(--color-accent)]={row.overdue > 0}
                          class:text-neutral-200={row.overdue === 0}
                        >{row.overdue}</td>
                        <td class="py-3 pr-3 font-mono tabular-nums text-neutral-500"
                          >{row.archived ?? 0}</td
                        >
                        <td class="py-3">
                          <div class="flex min-w-[7rem] items-center gap-2">
                            <SegmentedBar
                              value={p}
                              max={100}
                              segments={12}
                              fill="var(--color-ok)"
                              size="sm"
                              class="flex-1"
                            />
                            <span
                              class="w-8 shrink-0 text-right font-mono text-[11px] tabular-nums text-neutral-400"
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
            <div class="border-t border-neutral-800 pt-8">
              <h2 class="mb-4 text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-400">
                By board
              </h2>
              <div class="overflow-x-auto">
                <table class="w-full min-w-[28rem] text-left text-base">
                  <thead>
                    <tr
                      class="border-b border-neutral-700 text-[11px] font-mono uppercase tracking-[0.08em] text-neutral-500"
                    >
                      {#if !projectFilter}
                        <th class="pb-3 pr-3 font-normal">Project</th>
                      {/if}
                      <th class="pb-3 pr-3 font-normal">Board</th>
                      <th class="pb-3 pr-3 font-normal">Active</th>
                      <th class="pb-3 pr-3 font-normal">Done</th>
                      <th class="pb-3 pr-3 font-normal">Open</th>
                      <th class="pb-3 pr-3 font-normal">Overdue</th>
                      <th class="pb-3 pr-3 font-normal">Archived</th>
                      <th class="pb-3 font-normal">Progress</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each stats.by_board as row (`${row.project}/${row.name}`)}
                      {@const p = pct(row.done, row.tasks)}
                      <tr class="border-b border-neutral-800 last:border-0">
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
                        <td class="py-3 pr-3 font-mono tabular-nums text-[var(--color-ok)]"
                          >{row.done}</td
                        >
                        <td class="py-3 pr-3 font-mono tabular-nums text-neutral-200">{row.open}</td>
                        <td
                          class="py-3 pr-3 font-mono tabular-nums"
                          class:text-[var(--color-accent)]={row.overdue > 0}
                          class:text-neutral-200={row.overdue === 0}
                        >{row.overdue}</td>
                        <td class="py-3 pr-3 font-mono tabular-nums text-neutral-500"
                          >{row.archived ?? 0}</td
                        >
                        <td class="py-3">
                          <div class="flex min-w-[7rem] items-center gap-2">
                            <SegmentedBar
                              value={p}
                              max={100}
                              segments={12}
                              fill="var(--color-ok)"
                              size="sm"
                              class="flex-1"
                            />
                            <span
                              class="w-8 shrink-0 text-right font-mono text-[11px] tabular-nums text-neutral-400"
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
