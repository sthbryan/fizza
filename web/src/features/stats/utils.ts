import type { DayCount } from "@/lib/api";

export function fillDaySeries(rows: DayCount[], days = 30): DayCount[] {
  const map = new Map(rows.map((r) => [r.date, r.count]));
  const out: DayCount[] = [];
  const now = new Date();
  for (let i = days - 1; i >= 0; i--) {
    const d = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() - i));
    const key = d.toISOString().slice(0, 10);
    out.push({ date: key, count: map.get(key) ?? 0 });
  }
  return out;
}

export function pct(part: number, whole: number): number {
  if (!whole || whole <= 0) return 0;
  return Math.round((part / whole) * 100);
}

export function maxCount(rows: { count: number }[]): number {
  return rows.reduce((m, r) => Math.max(m, r.count), 0);
}

const COLUMN_COLORS: Record<string, string> = {
  todo: "var(--color-col-todo)",
  in_progress: "var(--color-col-progress)",
  "in progress": "var(--color-col-progress)",
  in_review: "var(--color-col-review)",
  "in review": "var(--color-col-review)",
  done: "var(--color-col-done)",
  completed: "var(--color-col-done)",
  closed: "var(--color-col-done)",
};

const PRIORITY_COLORS: Record<string, string> = {
  low: "var(--color-pri-low)",
  medium: "var(--color-pri-medium)",
  high: "var(--color-pri-high)",
  urgent: "var(--color-pri-urgent)",
};

export function columnColor(name: string): string {
  return COLUMN_COLORS[name.toLowerCase()] ?? "var(--color-col-default)";
}

export function priorityColor(name: string): string {
  return PRIORITY_COLORS[name.toLowerCase()] ?? "var(--color-accent)";
}

export function formatDayLabel(iso: string): string {
  const [, m, d] = iso.split("-");
  return `${Number(m)}/${Number(d)}`;
}

/** in_progress → IN PROGRESS; keeps charts/selects readable */
export function formatStatusLabel(name: string): string {
  return name.replaceAll(/[_-]+/g, " ").replaceAll(/\s+/g, " ").trim().toUpperCase();
}
