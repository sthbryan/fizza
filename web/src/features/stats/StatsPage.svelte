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
    return "ALL PROJECTS";
  }
</script>

<AppShell>
  <header
    class="border-b border-[var(--color-border-subtle)] bg-[var(--color-bg)] px-4 py-4 sm:px-6 sm:py-5"
  >
    <div
      class="flex flex-col gap-3.5 lg:flex-row lg:items-end lg:justify-between"
    >
      <div class="min-w-0">
        <div class="mb-1.5 text-sm text-[var(--color-text-muted)]">
          fizza / stats
        </div>
        <div class="flex flex-wrap items-baseline gap-2 sm:gap-3">
          <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">
            Progress
          </h1>
          <span class="text-base text-[var(--color-text-muted)]">
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
          <div
            class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-7 sm:gap-4"
          >
            {#each [
              { label: "Projects", value: t.projects, show: !projectFilter },
              { label: "Boards", value: t.boards, show: !boardFilter },
              { label: "Active", value: t.tasks, show: true },
              { label: "Open", value: t.open, show: true },
              { label: "Done", value: t.done, show: true },
              {
                label: "Overdue",
                value: t.overdue,
                show: true,
                danger: t.overdue > 0,
              },
              {
                label: "Archived",
                value: t.archived ?? 0,
                show: true,
              },
            ] as card (card.label)}
              {#if card.show}
                <div
                  class="rounded-3xl border border-[var(--color-border-subtle)] bg-[var(--color-bg-card)] px-4 py-4 sm:px-5 sm:py-5"
                >
                  <div
                    class="mb-1 text-xs font-medium uppercase tracking-[0.06em] text-[var(--color-text-muted)]"
                  >
                    {card.label}
                  </div>
                  <div
                    class="text-2xl font-semibold tracking-tight sm:text-3xl"
                    class:text-[var(--color-danger)]={card.danger}
                  >
                    {card.value}
                  </div>
                </div>
              {/if}
            {/each}
          </div>

          <div class="grid grid-cols-1 gap-4 lg:grid-cols-3 lg:gap-5">
            <div
              class="flex flex-col items-center justify-center rounded-3xl border border-[var(--color-border-subtle)] bg-[var(--color-bg-card)] p-6"
            >
              <ProgressRing
                value={donePct}
                label="Completion"
                sublabel="{t.done} of {t.tasks} active done"
              />
            </div>

            <div
              class="rounded-2xl border border-[var(--color-border-subtle)] bg-[var(--color-bg-card)] p-5 sm:p-6"
            >
              <h2 class="mb-4 text-base font-semibold tracking-tight">
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
              class="rounded-2xl border border-[var(--color-border-subtle)] bg-[var(--color-bg-card)] p-5 sm:p-6"
            >
              <h2 class="mb-4 text-base font-semibold tracking-tight">
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
              class="rounded-2xl border border-[var(--color-border-subtle)] bg-[var(--color-bg-card)] p-5 sm:p-6"
            >
              <h2 class="mb-1 text-base font-semibold tracking-tight">
                Tasks created
              </h2>
              <p class="mb-4 text-sm text-[var(--color-text-muted)]">
                Last 30 days
              </p>
              <DayChart
                rows={stats.created_by_day}
                color="var(--color-accent)"
              />
            </div>
            <div
              class="rounded-2xl border border-[var(--color-border-subtle)] bg-[var(--color-bg-card)] p-5 sm:p-6"
            >
              <h2 class="mb-1 text-base font-semibold tracking-tight">
                Activity
              </h2>
              <p class="mb-4 text-sm text-[var(--color-text-muted)]">
                Creates, updates, moves · last 30 days
              </p>
              <DayChart
                rows={stats.activity_by_day}
                color="var(--color-ok)"
              />
            </div>
          </div>

          {#if stats.by_project?.length}
            <div
              class="rounded-2xl border border-[var(--color-border-subtle)] bg-[var(--color-bg-card)] p-5 sm:p-6"
            >
              <h2 class="mb-4 text-base font-semibold tracking-tight">
                By project
              </h2>
              <div class="overflow-x-auto">
                <table class="w-full min-w-[28rem] text-left text-sm">
                  <thead>
                    <tr
                      class="border-b border-[var(--color-border-subtle)] text-xs uppercase tracking-[0.06em] text-[var(--color-text-muted)]"
                    >
                      <th class="pb-2.5 pr-3 font-medium">Project</th>
                      <th class="pb-2.5 pr-3 font-medium">Boards</th>
                      <th class="pb-2.5 pr-3 font-medium">Active</th>
                      <th class="pb-2.5 pr-3 font-medium">Done</th>
                      <th class="pb-2.5 pr-3 font-medium">Open</th>
                      <th class="pb-2.5 pr-3 font-medium">Overdue</th>
                      <th class="pb-2.5 pr-3 font-medium">Archived</th>
                      <th class="pb-2.5 font-medium">Progress</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each stats.by_project as row (row.name)}
                      {@const p = pct(row.done, row.tasks)}
                      <tr
                        class="border-b border-[var(--color-border-subtle)]/60 last:border-0"
                      >
                        <td class="py-3 pr-3 font-medium"
                          >{formatStatusLabel(row.name)}</td
                        >
                        <td class="py-3 pr-3 text-[var(--color-text-muted)]"
                          >{row.boards}</td
                        >
                        <td class="py-3 pr-3">{row.tasks}</td>
                        <td class="py-3 pr-3 text-[var(--color-ok)]"
                          >{row.done}</td
                        >
                        <td class="py-3 pr-3">{row.open}</td>
                        <td
                          class="py-3 pr-3"
                          class:text-[var(--color-danger)]={row.overdue > 0}
                          >{row.overdue}</td
                        >
                        <td class="py-3 pr-3 text-[var(--color-text-muted)]"
                          >{row.archived ?? 0}</td
                        >
                        <td class="py-3">
                          <div class="flex min-w-[6rem] items-center gap-2">
                            <div
                              class="h-1.5 flex-1 overflow-hidden rounded-full bg-[var(--color-bg-soft)]"
                            >
                              <div
                                class="h-full rounded-full bg-[var(--color-ok)]"
                                style:width="{p}%"
                              ></div>
                            </div>
                            <span
                              class="w-8 shrink-0 text-right font-mono text-xs text-[var(--color-text-muted)]"
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
              class="rounded-2xl border border-[var(--color-border-subtle)] bg-[var(--color-bg-card)] p-5 sm:p-6"
            >
              <h2 class="mb-4 text-base font-semibold tracking-tight">
                By board
              </h2>
              <div class="overflow-x-auto">
                <table class="w-full min-w-[28rem] text-left text-sm">
                  <thead>
                    <tr
                      class="border-b border-[var(--color-border-subtle)] text-xs uppercase tracking-[0.06em] text-[var(--color-text-muted)]"
                    >
                      {#if !projectFilter}
                        <th class="pb-2.5 pr-3 font-medium">Project</th>
                      {/if}
                      <th class="pb-2.5 pr-3 font-medium">Board</th>
                      <th class="pb-2.5 pr-3 font-medium">Active</th>
                      <th class="pb-2.5 pr-3 font-medium">Done</th>
                      <th class="pb-2.5 pr-3 font-medium">Open</th>
                      <th class="pb-2.5 pr-3 font-medium">Overdue</th>
                      <th class="pb-2.5 pr-3 font-medium">Archived</th>
                      <th class="pb-2.5 font-medium">Progress</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each stats.by_board as row (`${row.project}/${row.name}`)}
                      {@const p = pct(row.done, row.tasks)}
                      <tr
                        class="border-b border-[var(--color-border-subtle)]/60 last:border-0"
                      >
                        {#if !projectFilter}
                          <td
                            class="py-3 pr-3 text-[var(--color-text-secondary)]"
                            >{formatStatusLabel(row.project)}</td
                          >
                        {/if}
                        <td class="py-3 pr-3 font-medium"
                          >{formatStatusLabel(row.name)}</td
                        >
                        <td class="py-3 pr-3">{row.tasks}</td>
                        <td class="py-3 pr-3 text-[var(--color-ok)]"
                          >{row.done}</td
                        >
                        <td class="py-3 pr-3">{row.open}</td>
                        <td
                          class="py-3 pr-3"
                          class:text-[var(--color-danger)]={row.overdue > 0}
                          >{row.overdue}</td
                        >
                        <td class="py-3 pr-3 text-[var(--color-text-muted)]"
                          >{row.archived ?? 0}</td
                        >
                        <td class="py-3">
                          <div class="flex min-w-[6rem] items-center gap-2">
                            <div
                              class="h-1.5 flex-1 overflow-hidden rounded-full bg-[var(--color-bg-soft)]"
                            >
                              <div
                                class="h-full rounded-full bg-[var(--color-ok)]"
                                style:width="{p}%"
                              ></div>
                            </div>
                            <span
                              class="w-8 shrink-0 text-right font-mono text-xs text-[var(--color-text-muted)]"
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
