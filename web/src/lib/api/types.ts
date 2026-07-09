export type Priority = "low" | "medium" | "high" | "urgent";

export interface Project {
  id: number;
  name: string;
  description?: string;
  board_count?: number;
}

export interface Board {
  id: number;
  project_id: number;
  name: string;
  is_default?: boolean;
}

export interface Task {
  id: number;
  board_id: number;
  column_id: number;
  status: string;
  title: string;
  description?: string;
  priority: Priority | string;
  due_date?: string | null;
  completed_at?: string | null;
  archived_at?: string | null;
  parent_id?: number | null;
}

export interface ColumnSnapshot {
  id: number;
  name: string;
  position: number;
  wip_limit?: number | null;
  tasks: Task[];
  task_count?: number;
  truncated?: boolean;
}

export interface BoardSnapshot {
  project: string;
  board: Board;
  columns: ColumnSnapshot[];
  archived_count?: number;
}

export interface Column {
  id: number;
  name: string;
  position: number;
}

export interface NamedCount {
  name: string;
  count: number;
}

export interface DayCount {
  date: string;
  count: number;
}

export interface ProjectStatsRow {
  name: string;
  boards: number;
  tasks: number;
  done: number;
  open: number;
  overdue: number;
}

export interface BoardStatsRow {
  project: string;
  name: string;
  tasks: number;
  done: number;
  open: number;
  overdue: number;
}

export interface Stats {
  scope: { project?: string; board?: string };
  totals: {
    projects: number;
    boards: number;
    tasks: number;
    done: number;
    open: number;
    overdue: number;
  };
  by_priority: NamedCount[];
  by_column: NamedCount[];
  by_project?: ProjectStatsRow[];
  by_board?: BoardStatsRow[];
  created_by_day: DayCount[];
  activity_by_day: DayCount[];
}
