export type Priority = "low" | "medium" | "high" | "urgent";

export interface Project {
  id: number;
  name: string;
  description?: string;
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
  parent_id?: number | null;
}

export interface ColumnSnapshot {
  id: number;
  name: string;
  position: number;
  wip_limit?: number | null;
  tasks: Task[];
}

export interface BoardSnapshot {
  project: string;
  board: Board;
  columns: ColumnSnapshot[];
}

export interface Column {
  id: number;
  name: string;
  position: number;
}
